package hcloud

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/discovery"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/blang/semver/v4"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// LoadBalancerOps defines the Load Balancer related operations required by
// the hcloud-cloud-controller-manager.
type LoadBalancerOps interface {
	GetByName(ctx context.Context, name string) (*hcloud.LoadBalancer, error)
	GetByID(ctx context.Context, id int64) (*hcloud.LoadBalancer, error)
	GetByK8SServiceUID(ctx context.Context, svc *corev1.Service) (*hcloud.LoadBalancer, error)
	Create(ctx context.Context, lbName string, service *corev1.Service) (*hcloud.LoadBalancer, error)
	Delete(ctx context.Context, lb *hcloud.LoadBalancer) error
	ReconcileHCLB(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error)
	ReconcileHCLBTargets(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service, nodes []*corev1.Node) (bool, error)
	ReconcileHCLBServices(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error)
}

type loadBalancers struct {
	lbOps                        LoadBalancerOps
	disablePrivateIngressDefault bool
	disableIPv6Default           bool
	useProxyProtocolDefault      bool
}

func newLoadBalancers(lbOps LoadBalancerOps, disablePrivateIngressDefault bool, disableIPv6Default bool) *loadBalancers {
	return &loadBalancers{
		lbOps:                        lbOps,
		disablePrivateIngressDefault: disablePrivateIngressDefault,
		disableIPv6Default:           disableIPv6Default,
	}
}

func matchNodeSelector(svc *corev1.Service, nodes []*corev1.Node) ([]*corev1.Node, error) {
	var (
		err           error
		selectedNodes []*corev1.Node
	)

	selector := labels.Everything()
	if v, ok := annotation.LBNodeSelector.StringFromService(svc); ok {
		selector, err = labels.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("unable to parse the node-selector annotation: %w", err)
		}
	}

	for _, n := range nodes {
		if selector.Matches(labels.Set(n.GetLabels())) {
			selectedNodes = append(selectedNodes, n)
		}
	}

	return selectedNodes, nil
}

func (l *loadBalancers) GetLoadBalancer(
	ctx context.Context, _ string, service *corev1.Service,
) (status *corev1.LoadBalancerStatus, exists bool, err error) {
	const op = "hcloud/loadBalancers.GetLoadBalancer"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	lb, err := l.lbOps.GetByK8SServiceUID(ctx, service)
	if err != nil {
		if errors.Is(err, hcops.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%s: %v", op, err)
	}

	if v, ok := annotation.LBHostname.StringFromService(service); ok {
		return &corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{{Hostname: v}},
		}, true, nil
	}

	ingress, err := l.buildLoadBalancerStatusIngress(lb, service)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %v", op, err)
	}

	return &corev1.LoadBalancerStatus{Ingress: ingress}, true, nil
}

func (l *loadBalancers) GetLoadBalancerName(_ context.Context, _ string, service *corev1.Service) string {
	if v, ok := annotation.LBName.StringFromService(service); ok {
		return v
	}
	return cloudprovider.DefaultLoadBalancerName(service)
}

func (l *loadBalancers) EnsureLoadBalancer(
	ctx context.Context, clusterName string, svc *corev1.Service, nodes []*corev1.Node,
) (*corev1.LoadBalancerStatus, error) {
	const op = "hcloud/loadBalancers.EnsureLoadBalancer"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var (
		reload        bool
		lb            *hcloud.LoadBalancer
		err           error
		selectedNodes []*corev1.Node
	)

	selectedNodes, err = matchNodeSelector(svc, nodes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	nodeNames := make([]string, len(selectedNodes))
	for i, n := range selectedNodes {
		nodeNames[i] = n.Name
	}
	klog.InfoS("ensure Load Balancer", "op", op, "service", svc.Name, "nodes", nodeNames)

	lb, err = l.lbOps.GetByK8SServiceUID(ctx, svc)
	if err != nil && !errors.Is(err, hcops.ErrNotFound) {
		return nil, fmt.Errorf("%s: %v", op, err)
	}

	// Try the load balancer's name if we were not able to find it using the
	// service UID. This is required for two reasons:
	//
	// 1. Migration of load balancers which where created before identification
	// via the service UID was introduced.
	//
	// 2. Import of load balancers which were created by other means but
	// should be re-used by the cloud controller manager.
	lbName := l.GetLoadBalancerName(ctx, clusterName, svc)
	if errors.Is(err, hcops.ErrNotFound) {
		lb, err = l.lbOps.GetByName(ctx, lbName)
		if err != nil && !errors.Is(err, hcops.ErrNotFound) {
			return nil, fmt.Errorf("%s: %v", op, err)
		}
	}

	// If we were still not able to find the load balancer we create it.
	if errors.Is(err, hcops.ErrNotFound) {
		lb, err = l.lbOps.Create(ctx, lbName, svc)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	lbChanged, err := l.lbOps.ReconcileHCLB(ctx, lb, svc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	reload = reload || lbChanged

	// Reload early here if reload is true.
	// If the load balancer private network ip changed,
	// the load balancer would be detached and re-attached to the network
	// As a result all of the private network targets would have been
	// removed and we should make sure the lb state here matches the actual
	// lb state so that we can re-attach the targets if needed
	if reload {
		klog.InfoS("reload HC Load Balancer", "op", op, "loadBalancerID", lb.ID)
		lb, err = l.lbOps.GetByID(ctx, lb.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		reload = false
	}

	servicesChanged, err := l.lbOps.ReconcileHCLBServices(ctx, lb, svc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	reload = reload || servicesChanged

	targetsChanged, err := l.lbOps.ReconcileHCLBTargets(ctx, lb, svc, selectedNodes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	reload = reload || targetsChanged

	if reload {
		klog.InfoS("reload HC Load Balancer", "op", op, "loadBalancerID", lb.ID)
		lb, err = l.lbOps.GetByID(ctx, lb.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := annotation.LBToService(svc, lb); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Either set the Hostname or the IPs (below).
	// See: https://github.com/kubernetes/kubernetes/issues/66607
	if v, ok := annotation.LBHostname.StringFromService(svc); ok {
		return &corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{{Hostname: v}},
		}, nil
	}

	ingress, err := l.buildLoadBalancerStatusIngress(lb, svc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &corev1.LoadBalancerStatus{Ingress: ingress}, nil
}

func (l *loadBalancers) buildLoadBalancerStatusIngress(lb *hcloud.LoadBalancer, svc *corev1.Service) ([]corev1.LoadBalancerIngress, error) {
	const op = "hcloud/loadBalancers.getLoadBalancerStatusIngress"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var ingress []corev1.LoadBalancerIngress
	ipMode := corev1.LoadBalancerIPModeVIP

	useProxyProtocol, err := l.getUseProxyProtocol(svc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if useProxyProtocol {
		ipMode = corev1.LoadBalancerIPModeProxy
	}

	// "IPMode" was introduced in 1.29
	supportsIPModeField, err := checkIPModeSupport()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	disablePubNet, err := annotation.LBDisablePublicNetwork.BoolFromService(svc)
	if err != nil && !errors.Is(err, annotation.ErrNotSet) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if !disablePubNet {
		i := corev1.LoadBalancerIngress{IP: lb.PublicNet.IPv4.IP.String()}

		if supportsIPModeField {
			i.IPMode = &ipMode
		}

		ingress = append(ingress, i)

		disableIPV6, err := l.getDisableIPv6(svc)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if !disableIPV6 {
			i := corev1.LoadBalancerIngress{IP: lb.PublicNet.IPv6.IP.String()}

			if supportsIPModeField {
				i.IPMode = &ipMode
			}

			ingress = append(ingress, i)
		}
	}

	disablePrivIngress, err := l.getDisablePrivateIngress(svc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if !disablePrivIngress {
		for _, privateNet := range lb.PrivateNet {
			i := corev1.LoadBalancerIngress{IP: privateNet.IP.String()}

			if supportsIPModeField {
				i.IPMode = &ipMode
			}

			ingress = append(ingress, i)
		}
	}

	return ingress, nil
}

func (l *loadBalancers) getDisablePrivateIngress(svc *corev1.Service) (bool, error) {
	disable, err := annotation.LBDisablePrivateIngress.BoolFromService(svc)
	if err == nil {
		return disable, nil
	}
	if errors.Is(err, annotation.ErrNotSet) {
		return l.disablePrivateIngressDefault, nil
	}
	return false, err
}

func (l *loadBalancers) getUseProxyProtocol(svc *corev1.Service) (bool, error) {
	disable, err := annotation.LBSvcProxyProtocol.BoolFromService(svc)
	if err == nil {
		return disable, nil
	}
	if errors.Is(err, annotation.ErrNotSet) {
		return l.useProxyProtocolDefault, nil
	}
	return false, err
}

func (l *loadBalancers) getDisableIPv6(svc *corev1.Service) (bool, error) {
	disable, err := annotation.LBIPv6Disabled.BoolFromService(svc)
	if err == nil {
		return disable, nil
	}
	if errors.Is(err, annotation.ErrNotSet) {
		return l.disableIPv6Default, nil
	}
	return false, err
}

func (l *loadBalancers) UpdateLoadBalancer(
	ctx context.Context, clusterName string, svc *corev1.Service, nodes []*corev1.Node,
) error {
	const op = "hcloud/loadBalancers.UpdateLoadBalancer"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var (
		lb            *hcloud.LoadBalancer
		err           error
		selectedNodes []*corev1.Node
	)

	selectedNodes, err = matchNodeSelector(svc, nodes)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	nodeNames := make([]string, len(selectedNodes))
	for i, n := range selectedNodes {
		nodeNames[i] = n.Name
	}
	klog.InfoS("update Load Balancer", "op", op, "service", svc.Name, "nodes", nodeNames)

	lb, err = l.lbOps.GetByK8SServiceUID(ctx, svc)
	if errors.Is(err, hcops.ErrNotFound) {
		lbName := l.GetLoadBalancerName(ctx, clusterName, svc)

		lb, err = l.lbOps.GetByName(ctx, lbName)
		if errors.Is(err, hcops.ErrNotFound) {
			return nil
		}
		// further error types handled below
	}
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if _, err = l.lbOps.ReconcileHCLB(ctx, lb, svc); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if _, err = l.lbOps.ReconcileHCLBTargets(ctx, lb, svc, selectedNodes); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if _, err = l.lbOps.ReconcileHCLBServices(ctx, lb, svc); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (l *loadBalancers) EnsureLoadBalancerDeleted(ctx context.Context, _ string, service *corev1.Service) error {
	const op = "hcloud/loadBalancers.EnsureLoadBalancerDeleted"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	loadBalancer, err := l.lbOps.GetByK8SServiceUID(ctx, service)
	if errors.Is(err, hcops.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if loadBalancer.Protection.Delete {
		klog.InfoS("ignored: load balancer deletion protected", "op", op, "loadBalancerID", loadBalancer.ID)
		return nil
	}

	klog.InfoS("delete Load Balancer", "op", op, "loadBalancerID", loadBalancer.ID)
	err = l.lbOps.Delete(ctx, loadBalancer)
	if errors.Is(err, hcops.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func checkIPModeSupport() (bool, error) {
	config, err := config.GetConfig()
	if err != nil {
		return false, err
	}

	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	version, err := client.ServerVersion()
	if err != nil {
		return false, err
	}

	// ParseTolerant is necessary due to the leading "v"
	semverVersion, err := semver.ParseTolerant(version.GitVersion)
	if err != nil {
		return false, err
	}

	minimumVersion := semver.MustParse("1.29.0")

	return semverVersion.GE(minimumVersion), nil
}
