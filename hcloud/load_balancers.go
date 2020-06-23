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
	Create(ctx context.Context, lbName string, service *v1.Service) (*hcloud.LoadBalancer, error)
	ReconcileHCLB(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error)
	ReconcileHCLBTargets(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service, nodes []*v1.Node) (bool, error)
	ReconcileHCLBServices(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error)
}

type loadBalancers struct {
	lbOps LoadBalancerOps
	lbs   hcops.HCloudLoadBalancerClient // Deprecated: should only be referenced by hcops types
	ac    hcops.HCloudActionClient       // Deprecated: should only be referenced by hcops types
}

func newLoadBalancers(
	lbOps LoadBalancerOps, lbs hcops.HCloudLoadBalancerClient, ac hcops.HCloudActionClient,
) *loadBalancers {
	return &loadBalancers{lbOps: lbOps, lbs: lbs, ac: ac}
}

func (l *loadBalancers) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	loadBalancer, err := l.lbOps.GetByName(ctx, l.GetLoadBalancerName(ctx, clusterName, service))
	if err != nil {
		if errors.Is(err, hcops.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	if v, ok := annotation.LBHostname.StringFromService(service); ok {
		return &v1.LoadBalancerStatus{
			Ingress: []v1.LoadBalancerIngress{{Hostname: v}},
		}, true, nil
	}

	return &v1.LoadBalancerStatus{Ingress: []v1.LoadBalancerIngress{
		{
			IP: loadBalancer.PublicNet.IPv4.IP.String(),
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
	ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node,
) (*v1.LoadBalancerStatus, error) {
	const op = "hcloud/loadBalancers.EnsureLoadBalancer"
	var reload bool

	nodeNames := make([]string, len(nodes))
	for i, n := range nodes {
		nodeNames[i] = n.Name
	}
	klog.InfoS("ensure Load Balancer", "op", op, "service", service.Name, "nodes", nodeNames)

	lbName := l.GetLoadBalancerName(ctx, clusterName, service)
	loadBalancer, err := l.lbOps.GetByName(ctx, lbName)
	if err != nil && !errors.Is(err, hcops.ErrNotFound) {
		return nil, err
	}

	if errors.Is(err, hcops.ErrNotFound) {
		var err error

		loadBalancer, err = l.lbOps.Create(ctx, lbName, service)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	lbChanged, err := l.lbOps.ReconcileHCLB(ctx, loadBalancer, service)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	reload = reload || lbChanged

	targetsChanged, err := l.lbOps.ReconcileHCLBTargets(ctx, loadBalancer, service, nodes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	reload = reload || targetsChanged

	servicesChanged, err := l.lbOps.ReconcileHCLBServices(ctx, loadBalancer, service)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	reload = reload || servicesChanged

	if reload {
		klog.InfoS("reload HC Load Balancer", "op", op, "loadBalancerID", loadBalancer.ID)
		loadBalancer, err = l.lbOps.GetByID(ctx, loadBalancer.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := annotation.LBToService(service, loadBalancer); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Either set the Hostname or the IPs (below).
	// See: https://github.com/kubernetes/kubernetes/issues/66607
	if v, ok := annotation.LBHostname.StringFromService(service); ok {
		return &v1.LoadBalancerStatus{
			Ingress: []v1.LoadBalancerIngress{{Hostname: v}},
		}, nil
	}

	var ingress []v1.LoadBalancerIngress

	disablePubNet, err := annotation.LBDisablePublicNetwork.BoolFromService(service)
	if err != nil && !errors.Is(err, annotation.ErrNotSet) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !disablePubNet {
		ingress = append(ingress,
			v1.LoadBalancerIngress{
				IP: loadBalancer.PublicNet.IPv4.IP.String(),
			},
		// v1.LoadBalancerIngress{
		// 	IP: loadBalancer.PublicNet.IPv6.IP.String(),
		// }
		)
	}

	disablePrivIngress, err := annotation.LBDisablePrivateIngress.BoolFromService(service)
	if err != nil && !errors.Is(err, annotation.ErrNotSet) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !disablePrivIngress {
		for _, nw := range loadBalancer.PrivateNet {
			ingress = append(ingress, v1.LoadBalancerIngress{IP: nw.IP.String()})
		}
	}

	return &v1.LoadBalancerStatus{Ingress: ingress}, nil
}

func (l *loadBalancers) UpdateLoadBalancer(
	ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node,
) error {
	const op = "hcloud/loadBalancers.UpdateLoadBalancer"

	nodeNames := make([]string, len(nodes))
	for i, n := range nodes {
		nodeNames[i] = n.Name
	}
	klog.InfoS("update Load Balancer", "op", op, "service", service.Name, "nodes", nodeNames)

	loadBalancer, err := l.lbOps.GetByName(ctx, l.GetLoadBalancerName(ctx, clusterName, service))
	if errors.Is(err, hcops.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if _, err := l.lbOps.ReconcileHCLB(ctx, loadBalancer, service); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if _, err := l.lbOps.ReconcileHCLBTargets(ctx, loadBalancer, service, nodes); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if _, err := l.lbOps.ReconcileHCLBServices(ctx, loadBalancer, service); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (l *loadBalancers) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	const op = "hcloud/loadBalancer.EnsureLoadBalancerDeleted"

	loadBalancer, err := l.lbOps.GetByName(ctx, l.GetLoadBalancerName(ctx, clusterName, service))
	if errors.Is(err, hcops.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	klog.InfoS("delete Load Balancer", "op", op, "loadBalancerID", loadBalancer.ID)
	_, err = l.lbs.Delete(ctx, loadBalancer)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return nil
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
