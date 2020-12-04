package hcloud

import (
	"context"
	"errors"
	"fmt"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-go/hcloud"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

// LoadBalancerOps defines the Load Balancer related operations required by
// the hcloud-cloud-controller-manager.
type LoadBalancerOps interface {
	GetByName(ctx context.Context, name string) (*hcloud.LoadBalancer, error)
	GetByID(ctx context.Context, id int) (*hcloud.LoadBalancer, error)
	GetByK8SServiceUID(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error)
	Create(ctx context.Context, lbName string, service *v1.Service) (*hcloud.LoadBalancer, error)
	Delete(ctx context.Context, lb *hcloud.LoadBalancer) error
	ReconcileHCLB(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error)
	ReconcileHCLBTargets(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service, nodes []*v1.Node) (bool, error)
	ReconcileHCLBServices(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error)
}

type loadBalancers struct {
	lbOps LoadBalancerOps
	ac    hcops.HCloudActionClient // Deprecated: should only be referenced by hcops types
}

func newLoadBalancers(lbOps LoadBalancerOps, ac hcops.HCloudActionClient) *loadBalancers {
	return &loadBalancers{lbOps: lbOps, ac: ac}
}

func (l *loadBalancers) GetLoadBalancer(
	ctx context.Context, _ string, service *v1.Service,
) (status *v1.LoadBalancerStatus, exists bool, err error) {
	const op = "hcloud/loadBalancers.GetLoadBalancer"

	lb, err := l.lbOps.GetByK8SServiceUID(ctx, service)
	if err != nil {
		if errors.Is(err, hcops.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%s: %v", op, err)
	}

	if v, ok := annotation.LBHostname.StringFromService(service); ok {
		return &v1.LoadBalancerStatus{
			Ingress: []v1.LoadBalancerIngress{{Hostname: v}},
		}, true, nil
	}

	return &v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{
		{
			IP: lb.PublicNet.IPv4.IP.String(),
		},
		// {
		// 	IP: loadBalancer.PublicNet.IPv6.IP.String(),
		// },
	}}, true, nil
}

func (l *loadBalancers) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	if v, ok := annotation.LBName.StringFromService(service); ok {
		return v
	}
	return cloudprovider.DefaultLoadBalancerName(service)
}

func (l *loadBalancers) EnsureLoadBalancer(
	ctx context.Context, clusterName string, svc *v1.Service, nodes []*v1.Node,
) (*v1.LoadBalancerStatus, error) {
	const op = "hcloud/loadBalancers.EnsureLoadBalancer"
	var (
		reload bool
		lb     *hcloud.LoadBalancer
		err    error
	)

	nodeNames := make([]string, len(nodes))
	for i, n := range nodes {
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

	targetsChanged, err := l.lbOps.ReconcileHCLBTargets(ctx, lb, svc, nodes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	reload = reload || targetsChanged

	servicesChanged, err := l.lbOps.ReconcileHCLBServices(ctx, lb, svc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	reload = reload || servicesChanged

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
		return &v1.LoadBalancerStatus{
			Ingress: []v1.LoadBalancerIngress{{Hostname: v}},
		}, nil
	}

	var ingress []v1.LoadBalancerIngress

	disablePubNet, err := annotation.LBDisablePublicNetwork.BoolFromService(svc)
	if err != nil && !errors.Is(err, annotation.ErrNotSet) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !disablePubNet {
		ingress = append(ingress,
			v1.LoadBalancerIngress{
				IP: lb.PublicNet.IPv4.IP.String(),
			},
		// v1.LoadBalancerIngress{
		// 	IP: loadBalancer.PublicNet.IPv6.IP.String(),
		// }
		)
	}

	disablePrivIngress, err := annotation.LBDisablePrivateIngress.BoolFromService(svc)
	if err != nil && !errors.Is(err, annotation.ErrNotSet) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !disablePrivIngress {
		for _, nw := range lb.PrivateNet {
			ingress = append(ingress, v1.LoadBalancerIngress{IP: nw.IP.String()})
		}
	}

	return &v1.LoadBalancerStatus{Ingress: ingress}, nil
}

func (l *loadBalancers) UpdateLoadBalancer(
	ctx context.Context, clusterName string, svc *v1.Service, nodes []*v1.Node,
) error {
	const op = "hcloud/loadBalancers.UpdateLoadBalancer"
	var (
		lb  *hcloud.LoadBalancer
		err error
	)

	nodeNames := make([]string, len(nodes))
	for i, n := range nodes {
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
	if _, err = l.lbOps.ReconcileHCLBTargets(ctx, lb, svc, nodes); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if _, err = l.lbOps.ReconcileHCLBServices(ctx, lb, svc); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (l *loadBalancers) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	const op = "hcloud/loadBalancers.EnsureLoadBalancerDeleted"

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
