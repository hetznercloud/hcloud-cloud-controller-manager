package hcloud

import (
	"context"
	"errors"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/lbspec"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sliceutil"
)

// LoadBalancerOps defines the Load Balancer related operations required by
// the hcloud-cloud-controller-manager.
type LoadBalancerOps interface {
	GetByName(ctx context.Context, name string) (*hcloud.LoadBalancer, error)
	GetByID(ctx context.Context, id int64) (*hcloud.LoadBalancer, error)
	GetByK8SServiceUID(ctx context.Context, svc *corev1.Service) (*hcloud.LoadBalancer, error)
	Create(ctx context.Context, service *corev1.Service) (*hcloud.LoadBalancer, error)
	Delete(ctx context.Context, lb *hcloud.LoadBalancer) error
	ReconcileHCLB(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error)
	ReconcileHCLBTargets(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service, nodes []*corev1.Node) (bool, error)
	ReconcileHCLBServices(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error)
}

type loadBalancers struct {
	lbOps    LoadBalancerOps
	cfg      *config.LoadBalancerConfiguration
	recorder record.EventRecorder
}

func newLoadBalancers(lbOps LoadBalancerOps, lbCfg *config.LoadBalancerConfiguration, recorder record.EventRecorder) *loadBalancers {
	return &loadBalancers{
		lbOps:    lbOps,
		cfg:      lbCfg,
		recorder: recorder,
	}
}

func (l *loadBalancers) GetLoadBalancer(
	ctx context.Context, _ string, service *corev1.Service,
) (status *corev1.LoadBalancerStatus, exists bool, err error) {
	const op = "hcloud/loadBalancers.GetLoadBalancer"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	// The lookup comes before resolving the annotations on purpose. The service
	// controller calls us before deleting a Load Balancer and only removes its
	// finalizer once we returned without an error, so reporting that a Load
	// Balancer does not exist must not depend on the annotations being valid.
	// Otherwise a Service that never got a Load Balancer because one of its
	// annotations is invalid could not be deleted either.
	lb, err := l.lbOps.GetByK8SServiceUID(ctx, service)
	if err != nil {
		if errors.Is(err, hcops.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%s: %w", op, err)
	}

	spec, err := lbspec.Resolve(l.recorder, service, *l.cfg)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", op, err)
	}

	return &corev1.LoadBalancerStatus{Ingress: l.buildLoadBalancerStatusIngress(lb, spec)}, true, nil
}

func (l *loadBalancers) GetLoadBalancerName(_ context.Context, _ string, service *corev1.Service) string {
	return lbspec.Name(service)
}

func (l *loadBalancers) EnsureLoadBalancer(
	ctx context.Context, _ string, svc *corev1.Service, nodes []*corev1.Node,
) (*corev1.LoadBalancerStatus, error) {
	const op = "hcloud/loadBalancers.EnsureLoadBalancer"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var (
		reload bool
		lb     *hcloud.LoadBalancer
		err    error
	)

	spec, err := lbspec.Resolve(l.recorder, svc, *l.cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	selectedNodes := filterNodes(spec.NodeSelector, nodes)
	nodeNames := sliceutil.Transform(selectedNodes, func(n *corev1.Node) string {
		return n.GetName()
	})
	klog.InfoS("ensure Load Balancer", "op", op, "service", svc.Name, "nodes", nodeNames)

	// Try the load balancer's name if we were not able to find it using the
	// service UID. This is required for two reasons:
	//
	// 1. Migration of load balancers which where created before identification
	// via the service UID was introduced.
	//
	// 2. Import of load balancers which were created by other means but
	// should be re-used by the cloud controller manager.
	lb, err = l.lbOps.GetByK8SServiceUID(ctx, svc)
	if err != nil && !errors.Is(err, hcops.ErrNotFound) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Not found by UID label; try name
	if errors.Is(err, hcops.ErrNotFound) {
		lb, err = l.lbOps.GetByName(ctx, spec.Name)
	}

	// New Load Balancer -> create it
	if errors.Is(err, hcops.ErrNotFound) {
		lb, err = l.lbOps.Create(ctx, svc)
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
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

	return &corev1.LoadBalancerStatus{Ingress: l.buildLoadBalancerStatusIngress(lb, spec)}, nil
}

// buildLoadBalancerStatusIngress reports the addresses the Service is reachable
// on. A configured hostname replaces the IPs entirely.
// See: https://github.com/kubernetes/kubernetes/issues/66607
func (l *loadBalancers) buildLoadBalancerStatusIngress(lb *hcloud.LoadBalancer, spec lbspec.Spec) []corev1.LoadBalancerIngress {
	const op = "hcloud/loadBalancers.getLoadBalancerStatusIngress"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if spec.Hostname != "" {
		return []corev1.LoadBalancerIngress{{Hostname: spec.Hostname}}
	}

	ipMode := corev1.LoadBalancerIPModeVIP
	if spec.Service.ProxyProtocol != nil && *spec.Service.ProxyProtocol {
		ipMode = corev1.LoadBalancerIPModeProxy
	}

	var ingress []corev1.LoadBalancerIngress

	if lb.PublicNet.Enabled {
		ingress = append(ingress, corev1.LoadBalancerIngress{
			IP:     lb.PublicNet.IPv4.IP.String(),
			IPMode: &ipMode,
		})

		if spec.IPv6 {
			ingress = append(ingress, corev1.LoadBalancerIngress{
				IP:     lb.PublicNet.IPv6.IP.String(),
				IPMode: &ipMode,
			})
		}
	}

	if spec.PrivateIngress {
		for _, privateNet := range lb.PrivateNet {
			ingress = append(ingress, corev1.LoadBalancerIngress{
				IP:     privateNet.IP.String(),
				IPMode: &ipMode,
			})
		}
	}

	return ingress
}

func (l *loadBalancers) UpdateLoadBalancer(
	ctx context.Context,
	_ string,
	svc *corev1.Service,
	nodes []*corev1.Node,
) error {
	const op = "hcloud/loadBalancers.UpdateLoadBalancer"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var (
		lb  *hcloud.LoadBalancer
		err error
	)

	spec, err := lbspec.Resolve(l.recorder, svc, *l.cfg)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	selectedNodes := filterNodes(spec.NodeSelector, nodes)

	nodeNames := make([]string, len(selectedNodes))
	for i, n := range selectedNodes {
		nodeNames[i] = n.Name
	}
	klog.InfoS("update Load Balancer", "op", op, "service", svc.Name, "nodes", nodeNames)

	lb, err = l.lbOps.GetByK8SServiceUID(ctx, svc)
	if errors.Is(err, hcops.ErrNotFound) {
		lb, err = l.lbOps.GetByName(ctx, spec.Name)
	}
	switch {
	case errors.Is(err, hcops.ErrNotFound):
		// Nothing to do, the Load Balancer does not exist.
		return nil
	case err != nil:
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

func filterNodes(selector labels.Selector, nodes []*corev1.Node) []*corev1.Node {
	if selector.Empty() {
		return nodes
	}

	return slices.DeleteFunc(slices.Clone(nodes), func(n *corev1.Node) bool {
		return !selector.Matches(labels.Set(n.GetLabels()))
	})
}
