package hcops

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/providerid"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/robot"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// LabelServiceUID is a label added to the Hetzner Cloud backend to uniquely
// identify a load balancer managed by Hetzner Cloud Cloud Controller Manager.
const LabelServiceUID = "hcloud-ccm/service-uid"

// HCloudLoadBalancerClient defines the hcloud-go functions required by the
// Load Balancer operations type.
type HCloudLoadBalancerClient interface {
	GetByID(ctx context.Context, id int64) (*hcloud.LoadBalancer, *hcloud.Response, error)
	GetByName(ctx context.Context, name string) (*hcloud.LoadBalancer, *hcloud.Response, error)

	Create(ctx context.Context, opts hcloud.LoadBalancerCreateOpts) (hcloud.LoadBalancerCreateResult, *hcloud.Response, error)
	Update(
		ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerUpdateOpts,
	) (*hcloud.LoadBalancer, *hcloud.Response, error)
	Delete(ctx context.Context, lb *hcloud.LoadBalancer) (*hcloud.Response, error)

	AddService(
		ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServiceOpts,
	) (*hcloud.Action, *hcloud.Response, error)
	UpdateService(
		ctx context.Context, lb *hcloud.LoadBalancer, listenPort int, opts hcloud.LoadBalancerUpdateServiceOpts,
	) (*hcloud.Action, *hcloud.Response, error)
	DeleteService(
		ctx context.Context, lb *hcloud.LoadBalancer, listenPort int,
	) (*hcloud.Action, *hcloud.Response, error)

	ChangeAlgorithm(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerChangeAlgorithmOpts) (*hcloud.Action, *hcloud.Response, error)
	ChangeType(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerChangeTypeOpts) (*hcloud.Action, *hcloud.Response, error)
	ChangeDNSPtr(ctx context.Context, lb *hcloud.LoadBalancer, ip string, ptr *string) (*hcloud.Action, *hcloud.Response, error)

	AddServerTarget(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServerTargetOpts) (*hcloud.Action, *hcloud.Response, error)
	RemoveServerTarget(ctx context.Context, lb *hcloud.LoadBalancer, server *hcloud.Server) (*hcloud.Action, *hcloud.Response, error)

	AddIPTarget(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddIPTargetOpts) (*hcloud.Action, *hcloud.Response, error)
	RemoveIPTarget(ctx context.Context, lb *hcloud.LoadBalancer, server net.IP) (*hcloud.Action, *hcloud.Response, error)

	AttachToNetwork(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAttachToNetworkOpts) (*hcloud.Action, *hcloud.Response, error)
	DetachFromNetwork(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerDetachFromNetworkOpts) (*hcloud.Action, *hcloud.Response, error)

	EnablePublicInterface(
		ctx context.Context, loadBalancer *hcloud.LoadBalancer,
	) (*hcloud.Action, *hcloud.Response, error)
	DisablePublicInterface(
		ctx context.Context, loadBalancer *hcloud.LoadBalancer,
	) (*hcloud.Action, *hcloud.Response, error)

	AllWithOpts(ctx context.Context, opts hcloud.LoadBalancerListOpts) ([]*hcloud.LoadBalancer, error)
}

// LoadBalancerOps implements all operations regarding Hetzner Cloud Load Balancers.
type LoadBalancerOps struct {
	LBClient      HCloudLoadBalancerClient
	ActionClient  HCloudActionClient
	NetworkClient HCloudNetworkClient
	RobotClient   robot.Client
	CertOps       *CertificateOps
	RetryDelay    time.Duration
	NetworkID     int64
	Cfg           config.HCCMConfiguration
	Recorder      record.EventRecorder
}

// GetByK8SServiceUID tries to find a Load Balancer by its Kubernetes service
// UID.
//
// If no Load Balancer could be found ErrNotFound is returned. Likewise,
// ErrNonUniqueResult is returned if more than one matching Load Balancer is
// found.
func (l *LoadBalancerOps) GetByK8SServiceUID(ctx context.Context, svc *corev1.Service) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.GetByK8SServiceUID"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	opts := hcloud.LoadBalancerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: fmt.Sprintf("%s=%s", LabelServiceUID, svc.ObjectMeta.UID),
		},
	}
	lbs, err := l.LBClient.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: api error: %v", op, err)
	}
	if len(lbs) == 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrNotFound)
	}
	if len(lbs) > 1 {
		return nil, fmt.Errorf("%s: %w", op, ErrNonUniqueResult)
	}

	return lbs[0], nil
}

// GetByName retrieves a Hetzner Cloud Load Balancer by name.
//
// If no Load Balancer with name could be found, a wrapped ErrNotFound is
// returned.
func (l *LoadBalancerOps) GetByName(ctx context.Context, name string) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.GetByName"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	lb, _, err := l.LBClient.GetByName(ctx, name)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return nil, fmt.Errorf("%s: %s: %w", op, name, ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if lb == nil {
		return nil, fmt.Errorf("%s: %s: %w", op, name, ErrNotFound)
	}
	return lb, nil
}

// GetByID retrieves a Hetzner Cloud Load Balancer by id.
//
// If no Load Balancer with id could be found, a wrapped ErrNotFound is
// returned.
func (l *LoadBalancerOps) GetByID(ctx context.Context, id int64) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.GetByName"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	lb, _, err := l.LBClient.GetByID(ctx, id)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return nil, fmt.Errorf("%s: %d: %w", op, id, ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if lb == nil {
		return nil, fmt.Errorf("%s: %d: %w", op, id, ErrNotFound)
	}
	return lb, nil
}

// Create creates a new Load Balancer using the Hetzner Cloud API.
//
// It adds annotations identifying the HC Load Balancer to svc.
func (l *LoadBalancerOps) Create(
	ctx context.Context, lbName string, svc *corev1.Service,
) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.Create"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	opts := hcloud.LoadBalancerCreateOpts{
		Name:             lbName,
		LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
		Labels: map[string]string{
			LabelServiceUID: string(svc.ObjectMeta.UID),
		},
	}
	if v, ok := annotation.LBType.StringFromService(svc); ok {
		opts.LoadBalancerType.Name = v
	}
	if l.Cfg.LoadBalancer.Location != "" {
		opts.Location = &hcloud.Location{Name: l.Cfg.LoadBalancer.Location}
	}
	if v, ok := annotation.LBLocation.StringFromService(svc); ok {
		if v == "" {
			// Allow resetting the location in case someone wants to specify a network zone in an annotation
			// and a location as default.
			opts.Location = nil
		} else {
			opts.Location = &hcloud.Location{Name: v}
		}
	}
	opts.NetworkZone = hcloud.NetworkZone(l.Cfg.LoadBalancer.NetworkZone)
	if v, ok := annotation.LBNetworkZone.StringFromService(svc); ok {
		opts.NetworkZone = hcloud.NetworkZone(v)
	}
	if opts.Location == nil && opts.NetworkZone == "" {
		return nil, fmt.Errorf("%s: neither %s nor %s set", op, annotation.LBLocation, annotation.LBNetworkZone)
	}
	if opts.Location != nil && opts.NetworkZone != "" {
		opts.NetworkZone = ""
	}

	algType, err := annotation.LBAlgorithmType.LBAlgorithmTypeFromService(svc)
	if err != nil && !errors.Is(err, annotation.ErrNotSet) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !errors.Is(err, annotation.ErrNotSet) {
		opts.Algorithm = &hcloud.LoadBalancerAlgorithm{Type: algType}
	}

	if l.NetworkID > 0 {
		nw, _, err := l.NetworkClient.GetByID(ctx, l.NetworkID)
		if err != nil {
			return nil, fmt.Errorf("%s: get network %d: %w", op, l.NetworkID, err)
		}
		if nw == nil {
			return nil, fmt.Errorf("%s: get network %d: %w", op, l.NetworkID, ErrNotFound)
		}
		opts.Network = nw
	}
	disablePubIface, err := annotation.LBDisablePublicNetwork.BoolFromService(svc)
	if err != nil && !errors.Is(err, annotation.ErrNotSet) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if disablePubIface && !errors.Is(err, annotation.ErrNotSet) {
		opts.PublicInterface = hcloud.Ptr(false)
	}

	result, _, err := l.LBClient.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := WatchAction(ctx, l.ActionClient, result.Action); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	lb, err := l.GetByID(ctx, result.LoadBalancer.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: get Load Balancer: %d: %w", op, result.LoadBalancer.ID, err)
	}
	return lb, nil
}

// Delete removes a Hetzner Cloud load balancer from the backend.
func (l *LoadBalancerOps) Delete(ctx context.Context, lb *hcloud.LoadBalancer) error {
	const op = "hcops/LoadBalancerOps.Delete"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	_, err := l.LBClient.Delete(ctx, lb)
	if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	return nil
}

// ReconcileHCLB configures the Hetzner Cloud Load Balancer to match what is
// defined for the K8S Load Balancer svc.
func (l *LoadBalancerOps) ReconcileHCLB(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLB"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var changed bool

	labelSet, err := l.changeHCLBInfo(ctx, lb, svc)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || labelSet

	ipv4RDNSChanged, err := l.changeIPv4RDNS(ctx, lb, svc)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || ipv4RDNSChanged

	ipv6RDNSChanged, err := l.changeIPv6RDNS(ctx, lb, svc)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || ipv6RDNSChanged

	algorithmChanged, err := l.changeAlgorithm(ctx, lb, svc)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || algorithmChanged

	typeChanged, err := l.changeType(ctx, lb, svc)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || typeChanged

	networkDetached, err := l.detachFromNetwork(ctx, lb)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || networkDetached

	networkAttached, err := l.attachToNetwork(ctx, lb)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || networkAttached

	pubIfaceToggled, err := l.togglePublicInterface(ctx, lb, svc)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || pubIfaceToggled

	return changed, nil
}

// changeHCLBInfo changes a Load Balancers name and sets the service UID label
// if necessary.
//
// This is implemented in one method as both changes need to be made using
// hcloud.LoadBalancerUpdateOpts. Using one method reduces the number of API
// requests should more than one change be necessary.
func (l *LoadBalancerOps) changeHCLBInfo(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeHCLBInfo"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var (
		update bool
		opts   hcloud.LoadBalancerUpdateOpts
	)

	if lb.Labels[LabelServiceUID] != string(svc.ObjectMeta.UID) {
		// Make a defensive copy of labels. This way we do not modify lb unless
		// updating is really successful.
		labels := make(map[string]string, len(lb.Labels)+1)
		labels[LabelServiceUID] = string(svc.ObjectMeta.UID)
		for k, v := range lb.Labels {
			labels[k] = v
		}
		opts.Labels = labels
		update = true
	}

	if lbName, ok := annotation.LBName.StringFromService(svc); ok && lbName != lb.Name {
		opts.Name = lbName
		update = true
	}

	if !update {
		return false, nil
	}

	updated, _, err := l.LBClient.Update(ctx, lb, opts)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	lb.Name = updated.Name
	lb.Labels = updated.Labels

	return true, nil
}

func (l *LoadBalancerOps) changeIPv4RDNS(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeIPv4RDNS"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	rdns, ok := annotation.LBPublicIPv4RDNS.StringFromService(svc)
	// If the annotation is not set, no changes are needed
	if !ok {
		return false, nil
	}
	// If the annotation and the actual value match, no changes are needed
	if rdns == lb.PublicNet.IPv4.DNSPtr {
		return false, nil
	}

	action, _, err := l.LBClient.ChangeDNSPtr(ctx, lb, lb.PublicNet.IPv4.IP.String(), &rdns)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	err = WatchAction(ctx, l.ActionClient, action)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (l *LoadBalancerOps) changeIPv6RDNS(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeIPv6RDNS"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	rdns, ok := annotation.LBPublicIPv6RDNS.StringFromService(svc)
	// If the annotation is not set, no changes are needed
	if !ok {
		return false, nil
	}
	// If the annotation and the actual value match, no changes are needed
	if rdns == lb.PublicNet.IPv6.DNSPtr {
		return false, nil
	}

	action, _, err := l.LBClient.ChangeDNSPtr(ctx, lb, lb.PublicNet.IPv6.IP.String(), &rdns)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	err = WatchAction(ctx, l.ActionClient, action)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (l *LoadBalancerOps) changeAlgorithm(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeAlgorithm"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	at, err := annotation.LBAlgorithmType.LBAlgorithmTypeFromService(svc)
	if errors.Is(err, annotation.ErrNotSet) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	if at == lb.Algorithm.Type {
		return false, nil
	}

	opts := hcloud.LoadBalancerChangeAlgorithmOpts{Type: at}
	action, _, err := l.LBClient.ChangeAlgorithm(ctx, lb, opts)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	err = WatchAction(ctx, l.ActionClient, action)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (l *LoadBalancerOps) changeType(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeType"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	lt, ok := annotation.LBType.StringFromService(svc)
	if !ok {
		return false, nil
	}
	if lt == lb.LoadBalancerType.Name {
		return false, nil
	}

	opts := hcloud.LoadBalancerChangeTypeOpts{LoadBalancerType: &hcloud.LoadBalancerType{Name: lt}}
	action, _, err := l.LBClient.ChangeType(ctx, lb, opts)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	err = WatchAction(ctx, l.ActionClient, action)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (l *LoadBalancerOps) detachFromNetwork(ctx context.Context, lb *hcloud.LoadBalancer) (bool, error) {
	const op = "hcops/LoadBalancerOps.detachFromNetwork"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var changed bool

	for _, lbpn := range lb.PrivateNet {
		// Don't detach the Load Balancer from the network it is supposed to
		// be attached to.
		if l.NetworkID == lbpn.Network.ID {
			continue
		}
		klog.InfoS("detach from network", "op", op, "loadBalancerID", lb.ID, "networkID", lbpn.Network.ID)

		opts := hcloud.LoadBalancerDetachFromNetworkOpts{Network: lbpn.Network}
		a, _, err := l.LBClient.DetachFromNetwork(ctx, lb, opts)
		if err != nil {
			return changed, fmt.Errorf("%s: %w", op, err)
		}
		if err := WatchAction(ctx, l.ActionClient, a); err != nil {
			return changed, fmt.Errorf("%s: %w", op, err)
		}
		changed = true
	}
	return changed, nil
}

func (l *LoadBalancerOps) attachToNetwork(ctx context.Context, lb *hcloud.LoadBalancer) (bool, error) {
	const op = "hcops/LoadBalancerOps.attachToNetwork"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	// Don't attach the Load Balancer if network is not set, or the load
	// balancer is already attached.
	if l.NetworkID == 0 || lbAttached(lb, l.NetworkID) {
		return false, nil
	}
	klog.InfoS("attach to network", "op", op, "loadBalancerID", lb.ID, "networkID", l.NetworkID)

	nw, _, err := l.NetworkClient.GetByID(ctx, l.NetworkID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	if nw == nil || hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
		return false, fmt.Errorf("%s: %d: not found", op, l.NetworkID)
	}

	retryDelay := l.RetryDelay
	if retryDelay == 0 {
		retryDelay = time.Second
	}
	opts := hcloud.LoadBalancerAttachToNetworkOpts{Network: nw}
	a, _, err := l.LBClient.AttachToNetwork(ctx, lb, opts)
	if hcloud.IsError(err, hcloud.ErrorCodeConflict) || hcloud.IsError(err, hcloud.ErrorCodeLocked) {
		klog.InfoS("retry due to conflict or lock",
			"op", op, "delay", fmt.Sprintf("%v", retryDelay), "err", fmt.Sprintf("%v", err))

		time.Sleep(retryDelay)
		a, _, err = l.LBClient.AttachToNetwork(ctx, lb, opts)
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if err := WatchAction(ctx, l.ActionClient, a); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (l *LoadBalancerOps) togglePublicInterface(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.togglePublicInterface"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var a *hcloud.Action

	disable, err := annotation.LBDisablePublicNetwork.BoolFromService(svc)
	if errors.Is(err, annotation.ErrNotSet) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if disable == !lb.PublicNet.Enabled {
		return false, nil
	}

	if disable {
		a, _, err = l.LBClient.DisablePublicInterface(ctx, lb)
	} else {
		a, _, err = l.LBClient.EnablePublicInterface(ctx, lb)
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if err := WatchAction(ctx, l.ActionClient, a); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

// ReconcileHCLBTargets adds or removes target nodes from the Hetzner Cloud
// Load Balancer when nodes are added or removed to the K8S cluster.
func (l *LoadBalancerOps) ReconcileHCLBTargets(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service, nodes []*corev1.Node,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLBTargets"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var (
		// Set of all K8S server IDs currently assigned as nodes to this
		// cluster.
		k8sNodeIDsHCloud = make(map[int64]bool)
		k8sNodeIDsRobot  = make(map[int]bool)
		k8sNodes         = make(map[int64]*corev1.Node)

		robotIPsToIDs = make(map[string]int)
		robotIDToIPv4 = make(map[int]string)
		// Set of server IDs assigned as targets to the HC Load Balancer. Some
		// of the entries may get deleted during reconcilement. In this case
		// the hclbTargetIDs[id] is always false. If hclbTargetIDs[id] is true,
		// the node with this server id is assigned to the K8S cluster.
		hclbTargetIDs = make(map[int64]bool)

		// Set of server IPs assigned as targets to the HC Load Balancer. Some
		// of the entries may get deleted during reconcilement. In this case
		// the hclbTargetIPs[id] is always false. If hclbTargetIPs[id] is true,
		// the node with this server id is assigned to the K8S cluster.
		hclbTargetIPs = make(map[string]bool)

		changed bool
	)

	usePrivateIP, err := l.getUsePrivateIP(svc)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	if usePrivateIP && l.NetworkID == 0 {
		return changed, fmt.Errorf("%s: use private ip: missing network id", op)
	}

	// Extract HC server IDs of all K8S nodes assigned to the K8S cluster.
	for _, node := range nodes {
		id, isCloudServer, err := providerid.ToServerID(node.Spec.ProviderID)
		if err != nil {
			return changed, fmt.Errorf("%s: %w", op, err)
		}
		if isCloudServer {
			k8sNodeIDsHCloud[id] = true
		} else {
			k8sNodeIDsRobot[int(id)] = true
		}
		k8sNodes[id] = node
	}

	// List all robot servers to check whether the ip targets of the load balancer
	// correspond to a dedicated server

	if l.Cfg.Robot.Enabled {
		dedicatedServers, err := l.RobotClient.ServerGetList()
		if err != nil {
			return changed, fmt.Errorf("%s: failed to get list of dedicated servers: %w", op, err)
		}

		for _, s := range dedicatedServers {
			robotIPsToIDs[s.ServerIP] = s.ServerNumber
			robotIDToIPv4[s.ServerNumber] = s.ServerIP
		}
	}

	numberOfTargets := len(lb.Targets)

	// Extract IDs of the hc Load Balancer's server targets. Along the way,
	// Remove all server targets from the HC Load Balancer which are currently
	// not assigned as nodes to the K8S Load Balancer.
	for _, target := range lb.Targets {
		if target.Type == hcloud.LoadBalancerTargetTypeServer {
			id := target.Server.Server.ID
			recreate := target.UsePrivateIP != usePrivateIP
			hclbTargetIDs[id] = k8sNodeIDsHCloud[id] && !recreate
			if hclbTargetIDs[id] {
				continue
			}

			// k8sNodes[id] can be nil if the node is currently being deleted.
			var nodeName string
			if node := k8sNodes[id]; node != nil {
				nodeName = node.Name
			} else {
				nodeName = fmt.Sprintf("%d", id)
			}

			klog.InfoS("remove target", "op", op, "service", svc.ObjectMeta.Name, "targetName", nodeName)
			// Target needs to be re-created or node currently not in use by k8s
			// Load Balancer. Remove it from the HC Load Balancer
			a, _, err := l.LBClient.RemoveServerTarget(ctx, lb, target.Server.Server)
			if err != nil {
				return changed, fmt.Errorf("%s: target: %s: %w", op, nodeName, err)
			}
			if err := WatchAction(ctx, l.ActionClient, a); err != nil {
				return changed, fmt.Errorf("%s: target: %s: %w", op, nodeName, err)
			}
			changed = true
			numberOfTargets--
		}

		// Cleanup of IP Targets happens whether Robot Support is enabled or not.
		// If it is not enabled, we remove all IP targets.
		if target.Type == hcloud.LoadBalancerTargetTypeIP {
			ip := target.IP.IP
			id, foundServer := robotIPsToIDs[ip]
			hclbTargetIPs[ip] = foundServer && k8sNodeIDsRobot[id]
			if hclbTargetIPs[ip] {
				continue
			}

			// k8sNodes[id] can be nil if the node is currently being deleted.
			var nodeName string
			if node := k8sNodes[int64(id)]; node != nil {
				nodeName = node.Name
			} else {
				nodeName = fmt.Sprintf("%d", id)
			}

			klog.InfoS("remove target", "op", op, "service", svc.ObjectMeta.Name, "targetName", nodeName)
			// Node currently not in use by k8s Load Balancer. Remove it from the HC Load Balancer.
			a, _, err := l.LBClient.RemoveIPTarget(ctx, lb, net.ParseIP(ip))
			if err != nil {
				var e error
				if foundServer {
					e = fmt.Errorf("%s: target: %s: %w", op, nodeName, err)
				} else {
					e = fmt.Errorf("%s: targetIP: %s: %w", op, ip, err)
				}
				return changed, e
			}
			if err := WatchAction(ctx, l.ActionClient, a); err != nil {
				var e error
				if foundServer {
					e = fmt.Errorf("%s: target: %s: %w", op, nodeName, err)
				} else {
					e = fmt.Errorf("%s: targetIP: %s: %w", op, ip, err)
				}
				return changed, e
			}
			changed = true
			numberOfTargets--
		}
	}

	// Assign the servers which are currently assigned as nodes
	// to the K8S Load Balancer as server targets to the HC Load Balancer.
	for id := range k8sNodeIDsHCloud {
		// Don't assign the node again if it is already assigned to the HC load
		// balancer.
		if hclbTargetIDs[id] {
			continue
		}
		node := k8sNodes[id]

		if numberOfTargets >= lb.LoadBalancerType.MaxTargets {
			l.emitMaxTargetsReachedError(node, svc, op)
			continue
		}

		klog.InfoS("add target", "op", op, "service", svc.ObjectMeta.Name, "targetName", node.Name)
		opts := hcloud.LoadBalancerAddServerTargetOpts{
			Server:       &hcloud.Server{ID: id},
			UsePrivateIP: &usePrivateIP,
		}
		a, _, err := l.LBClient.AddServerTarget(ctx, lb, opts)
		if err != nil {
			if hcloud.IsError(err, hcloud.ErrorCodeResourceLimitExceeded) {
				l.emitMaxTargetsReachedError(node, svc, op)
				// Continue loop so that error is emitted for each node
				continue
			}
			return changed, fmt.Errorf("%s: target %s: %w", op, node.Name, err)
		}
		if err := WatchAction(ctx, l.ActionClient, a); err != nil {
			return changed, fmt.Errorf("%s: target %s: %w", op, node.Name, err)
		}
		changed = true
		numberOfTargets++
	}

	if l.Cfg.Robot.Enabled {
		// Assign the dedicated servers which are currently assigned as nodes
		// to the K8S Load Balancer as IP targets to the HC Load Balancer.
		for id := range k8sNodeIDsRobot {
			ip := robotIDToIPv4[id]
			node := k8sNodes[int64(id)]

			// Don't assign the node again if it is already assigned to the HC load
			// balancer.
			if hclbTargetIPs[ip] {
				continue
			}
			if ip == "" {
				l.Recorder.Eventf(node, corev1.EventTypeWarning, "ServerNotFound", "No server with id %d was found in Robot", id)
				klog.InfoS("k8s node found but no corresponding server in robot", "id", id)
				continue
			}

			if numberOfTargets >= lb.LoadBalancerType.MaxTargets {
				l.emitMaxTargetsReachedError(node, svc, op)
				continue
			}

			klog.InfoS("add target", "op", op, "service", svc.ObjectMeta.Name, "targetName", node, "ip", ip)
			opts := hcloud.LoadBalancerAddIPTargetOpts{
				IP: net.ParseIP(ip),
			}
			a, _, err := l.LBClient.AddIPTarget(ctx, lb, opts)
			if err != nil {
				if hcloud.IsError(err, hcloud.ErrorCodeResourceLimitExceeded) {
					l.emitMaxTargetsReachedError(node, svc, op)
					continue
				}
				return changed, fmt.Errorf("%s: target %s: %w", op, node, err)
			}
			if err := WatchAction(ctx, l.ActionClient, a); err != nil {
				return changed, fmt.Errorf("%s: target %s: %w", op, node, err)
			}
			changed = true
			numberOfTargets++
		}
	}

	return changed, nil
}

//nolint:unparam // op might get set to different values in the future
func (l *LoadBalancerOps) emitMaxTargetsReachedError(node *corev1.Node, svc *corev1.Service, op string) {
	l.Recorder.Eventf(node, corev1.EventTypeWarning, "MaxTargetsReached",
		"Node could not be added to Load Balancer for service %s because the max number of targets has been reached",
		svc.ObjectMeta.Name)
	klog.InfoS("cannot add server target because max number of targets have been reached", "op", op, "service", svc.ObjectMeta.Name, "targetName", node.Name)
}

func (l *LoadBalancerOps) getUsePrivateIP(svc *corev1.Service) (bool, error) {
	usePrivateIP, err := annotation.LBUsePrivateIP.BoolFromService(svc)
	if err != nil {
		if errors.Is(err, annotation.ErrNotSet) {
			return l.Cfg.LoadBalancer.UsePrivateIP, nil
		}
		return false, err
	}
	return usePrivateIP, nil
}

// ReconcileHCLBServices synchronizes services exposed by the Hetzner Cloud
// Load Balancer with the kubernetes cluster.
func (l *LoadBalancerOps) ReconcileHCLBServices(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLBServices"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var changed bool

	if err := l.reconcileManagedCertificate(ctx, svc); err != nil {
		return false, fmt.Errorf("%s: %v", op, err)
	}

	hclbListenPorts := make(map[int]bool, len(lb.Services))
	for _, hclbService := range lb.Services {
		hclbListenPorts[hclbService.ListenPort] = true
	}

	// Add all ports exposed by the K8S Load Balancer service to the HC load
	// balancer. Remove the ports from the set of HC Load Balancer listen
	// ports.
	for _, port := range svc.Spec.Ports {
		var (
			addOpts hcloud.LoadBalancerAddServiceOpts
			updOpts hcloud.LoadBalancerUpdateServiceOpts
			action  *hcloud.Action

			err error
		)

		portNo := int(port.Port)
		portExists := hclbListenPorts[portNo]
		delete(hclbListenPorts, portNo)

		b := &hclbServiceOptsBuilder{Port: port, Service: svc, CertOps: l.CertOps}
		if portExists {
			klog.InfoS("update service", "op", op, "port", portNo, "loadBalancerID", lb.ID)

			updOpts, err = b.buildUpdateServiceOpts()
			if err != nil {
				return changed, fmt.Errorf("%s: %w", op, err)
			}
			action, _, err = l.LBClient.UpdateService(ctx, lb, b.listenPort, updOpts)
			if err != nil {
				return changed, fmt.Errorf("%s: %w", op, err)
			}
		} else {
			klog.InfoS("add service", "op", op, "port", portNo, "loadBalancerID", lb.ID)

			addOpts, err = b.buildAddServiceOpts()
			if err != nil {
				return changed, fmt.Errorf("%s: %w", op, err)
			}
			action, _, err = l.LBClient.AddService(ctx, lb, addOpts)
			if err != nil {
				return changed, fmt.Errorf("%s: %w", op, err)
			}
		}

		if err = WatchAction(ctx, l.ActionClient, action); err != nil {
			return changed, fmt.Errorf("%s: %w", op, err)
		}
		changed = true
	}

	// Remove any left-over services from the hc Load Balancer.
	for p := range hclbListenPorts {
		klog.InfoS("remove service", "op", op, "port", p, "loadBalancerID", lb.ID)
		a, _, err := l.LBClient.DeleteService(ctx, lb, p)
		if err != nil {
			return changed, fmt.Errorf("%s: port %d: %w", op, p, err)
		}
		err = WatchAction(ctx, l.ActionClient, a)
		if err != nil {
			return changed, fmt.Errorf("%s: port: %d: %w", op, p, err)
		}
		changed = true
	}

	return changed, nil
}

func (l *LoadBalancerOps) reconcileManagedCertificate(ctx context.Context, svc *corev1.Service) error {
	const op = "hcops/LoadBalancerOps.reconcileManagedCertificate"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if typ, ok := annotation.LBSvcHTTPCertificateType.StringFromService(svc); !ok || typ != string(hcloud.CertificateTypeManaged) {
		return nil
	}
	name, ok := annotation.LBSvcHTTPManagedCertificateName.StringFromService(svc)
	if !ok || name == "" {
		name = fmt.Sprintf("ccm-managed-certificate-%s", svc.ObjectMeta.UID)
	}
	domains, err := annotation.LBSvcHTTPManagedCertificateDomains.StringsFromService(svc)
	if errors.Is(err, annotation.ErrNotSet) {
		return fmt.Errorf("%s: no domains for managed certificate", op)
	}
	labels := map[string]string{
		LabelServiceUID: string(svc.ObjectMeta.UID),
	}
	// It's ok to ignore the error here. We are only interested if the
	// annotation is set and parseable as a truthy boolean. Anything else tells
	// us we do not want to use ACME staging.
	if ok, _ := annotation.LBSvcHTTPManagedCertificateUseACMEStaging.BoolFromService(svc); ok {
		labels["HC-Use-Staging-CA"] = "true"
	}
	err = l.CertOps.CreateManagedCertificate(ctx, name, domains, labels)
	if errors.Is(err, ErrAlreadyExists) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	return nil
}

type hclbServiceOptsBuilder struct {
	Port    corev1.ServicePort
	Service *corev1.Service
	CertOps *CertificateOps

	listenPort      int
	destinationPort int
	proxyProtocol   *bool
	protocol        hcloud.LoadBalancerServiceProtocol
	httpOpts        struct {
		CookieName     *string
		CookieLifetime *time.Duration
		Certificates   []*hcloud.Certificate
		RedirectHTTP   *bool
		StickySessions *bool
	}
	addHTTP         bool
	healthCheckOpts struct {
		Protocol hcloud.LoadBalancerServiceProtocol
		Port     *int
		Interval *time.Duration
		Timeout  *time.Duration
		Retries  *int
		httpOpts struct {
			Domain      *string
			Path        *string
			Response    *string
			StatusCodes []string
			TLS         *bool
		}
	}
	addHealthCheck bool

	once sync.Once
	err  error
}

func (b *hclbServiceOptsBuilder) extract() {
	const op = "hcops/hclbServiceOptsBuilder.extract"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	b.listenPort = int(b.Port.Port)
	b.destinationPort = int(b.Port.NodePort)

	b.do(func() error {
		pp, err := annotation.LBSvcProxyProtocol.BoolFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.proxyProtocol = hcloud.Ptr(pp)
		return nil
	})

	b.protocol = hcloud.LoadBalancerServiceProtocolTCP
	b.do(func() error {
		p, err := annotation.LBSvcProtocol.LBSvcProtocolFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.protocol = p
		return nil
	})

	if v, ok := annotation.LBSvcHTTPCookieName.StringFromService(b.Service); ok {
		b.httpOpts.CookieName = &v
		b.addHTTP = true
	}

	b.do(func() error {
		lt, err := annotation.LBSvcHTTPCookieLifetime.DurationFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.httpOpts.CookieLifetime = &lt
		b.addHTTP = true
		return nil
	})

	b.do(func() error {
		certtyp, ok := annotation.LBSvcHTTPCertificateType.StringFromService(b.Service)
		if ok && certtyp == string(hcloud.CertificateTypeManaged) {
			// Continue with managed certificates below
			return nil
		}

		certs, err := annotation.LBSvcHTTPCertificates.CertificatesFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		certs, err = b.resolveCertsByNameOrID(ctx, certs)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.httpOpts.Certificates = certs
		b.addHTTP = true
		return nil
	})

	b.do(func() error {
		certtyp, ok := annotation.LBSvcHTTPCertificateType.StringFromService(b.Service)
		if !ok || certtyp != string(hcloud.CertificateTypeManaged) {
			// Not a a managed certificate.
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		svcUID := b.Service.ObjectMeta.UID
		cert, err := b.CertOps.GetCertificateByLabel(ctx, fmt.Sprintf("%s=%s", LabelServiceUID, svcUID))
		if err != nil {
			return err
		}
		b.httpOpts.Certificates = []*hcloud.Certificate{{ID: cert.ID}}
		b.addHTTP = true
		return nil
	})

	b.do(func() error {
		redirectHTTP, err := annotation.LBSvcRedirectHTTP.BoolFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.httpOpts.RedirectHTTP = &redirectHTTP
		b.addHTTP = true
		return nil
	})

	b.do(func() error {
		stickySessions, err := annotation.LBSvcHTTPStickySessions.BoolFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.httpOpts.StickySessions = &stickySessions
		b.addHTTP = true
		return nil
	})

	b.extractHealthCheck()
}

func (b *hclbServiceOptsBuilder) resolveCertsByNameOrID(ctx context.Context, cs []*hcloud.Certificate) ([]*hcloud.Certificate, error) {
	const op = "hcops/hclbServiceOptsBuilder.resolveCertsByNameOrID"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	resolved := make([]*hcloud.Certificate, len(cs))
	for i, c := range cs {
		if c.ID != 0 {
			resolved[i] = c
			continue
		}

		c, err := b.CertOps.GetCertificateByNameOrID(ctx, c.Name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		resolved[i] = &hcloud.Certificate{ID: c.ID}
	}
	return resolved, nil
}

func (b *hclbServiceOptsBuilder) extractHealthCheck() {
	const op = "hcops/hclbServiceOptsBuilder.extractHealthCheck"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	b.do(func() error {
		p, err := annotation.LBSvcHealthCheckProtocol.LBSvcProtocolFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			// Set the service protocol but do not set the addHealthCheck flag.
			// This way the health check is configured using the service
			// protocol only if at least one health check annotation is
			// present.
			b.healthCheckOpts.Protocol = b.protocol
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.healthCheckOpts.Protocol = p
		b.addHealthCheck = true
		return nil
	})

	b.do(func() error {
		hcPort, err := annotation.LBSvcHealthCheckPort.IntFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.healthCheckOpts.Port = hcloud.Ptr(hcPort)
		b.addHealthCheck = true
		return nil
	})

	b.do(func() error {
		hcInterval, err := annotation.LBSvcHealthCheckInterval.DurationFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.healthCheckOpts.Interval = hcloud.Ptr(hcInterval)
		b.addHealthCheck = true
		return nil
	})

	b.do(func() error {
		t, err := annotation.LBSvcHealthCheckTimeout.DurationFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.healthCheckOpts.Timeout = hcloud.Ptr(t)
		b.addHealthCheck = true
		return nil
	})

	b.do(func() error {
		v, err := annotation.LBSvcHealthCheckRetries.IntFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.healthCheckOpts.Retries = hcloud.Ptr(v)
		b.addHealthCheck = true
		return nil
	})

	if b.healthCheckOpts.Protocol == hcloud.LoadBalancerServiceProtocolTCP {
		return
	}

	if v, ok := annotation.LBSvcHealthCheckHTTPDomain.StringFromService(b.Service); ok {
		b.healthCheckOpts.httpOpts.Domain = &v
	}

	if v, ok := annotation.LBSvcHealthCheckHTTPPath.StringFromService(b.Service); ok {
		b.healthCheckOpts.httpOpts.Path = &v
	}

	b.do(func() error {
		tls, err := annotation.LBSvcHealthCheckHTTPValidateCertificate.BoolFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.healthCheckOpts.httpOpts.TLS = &tls
		return nil
	})

	b.do(func() error {
		scs, err := annotation.LBSvcHealthCheckHTTPStatusCodes.StringsFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.healthCheckOpts.httpOpts.StatusCodes = scs
		return nil
	})
}

func (b *hclbServiceOptsBuilder) initialize() error {
	b.once.Do(b.extract)
	return b.err
}

func (b *hclbServiceOptsBuilder) do(f func() error) {
	if b.err != nil {
		return
	}
	b.err = f()
}

func (b *hclbServiceOptsBuilder) buildAddServiceOpts() (hcloud.LoadBalancerAddServiceOpts, error) {
	const op = "hcops/hclbServiceOptsBuilder.buildAddServiceOpts"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if err := b.initialize(); err != nil {
		return hcloud.LoadBalancerAddServiceOpts{}, fmt.Errorf("%s: %w", op, err)
	}

	opts := hcloud.LoadBalancerAddServiceOpts{
		ListenPort:      hcloud.Ptr(b.listenPort),
		DestinationPort: hcloud.Ptr(b.destinationPort),
		Protocol:        b.protocol,
		Proxyprotocol:   b.proxyProtocol,
	}
	if b.addHTTP {
		opts.HTTP = &hcloud.LoadBalancerAddServiceOptsHTTP{
			CookieName:     b.httpOpts.CookieName,
			CookieLifetime: b.httpOpts.CookieLifetime,
			Certificates:   b.httpOpts.Certificates,
			RedirectHTTP:   b.httpOpts.RedirectHTTP,
			StickySessions: b.httpOpts.StickySessions,
		}
	}
	if b.addHealthCheck {
		port := b.healthCheckOpts.Port
		if port == nil {
			port = hcloud.Ptr(b.destinationPort)
		}
		opts.HealthCheck = &hcloud.LoadBalancerAddServiceOptsHealthCheck{
			Protocol: b.healthCheckOpts.Protocol,
			Interval: b.healthCheckOpts.Interval,
			Port:     port,
			Retries:  b.healthCheckOpts.Retries,
			Timeout:  b.healthCheckOpts.Timeout,
		}
		if b.healthCheckOpts.Protocol == hcloud.LoadBalancerServiceProtocolHTTP ||
			b.healthCheckOpts.Protocol == hcloud.LoadBalancerServiceProtocolHTTPS {
			opts.HealthCheck.HTTP = &hcloud.LoadBalancerAddServiceOptsHealthCheckHTTP{
				Domain:      b.healthCheckOpts.httpOpts.Domain,
				Path:        b.healthCheckOpts.httpOpts.Path,
				Response:    b.healthCheckOpts.httpOpts.Response,
				StatusCodes: b.healthCheckOpts.httpOpts.StatusCodes,
				TLS:         b.healthCheckOpts.httpOpts.TLS,
			}
		}
	} else {
		opts.HealthCheck = &hcloud.LoadBalancerAddServiceOptsHealthCheck{
			Protocol: hcloud.LoadBalancerServiceProtocolTCP,
			Port:     hcloud.Ptr(b.destinationPort),
		}
	}

	return opts, nil
}

func (b *hclbServiceOptsBuilder) buildUpdateServiceOpts() (hcloud.LoadBalancerUpdateServiceOpts, error) {
	const op = "hcops/hclbServiceOptsBuilder.buildUpdateServiceOpts"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if err := b.initialize(); err != nil {
		return hcloud.LoadBalancerUpdateServiceOpts{}, fmt.Errorf("%s: %w", op, err)
	}

	opts := hcloud.LoadBalancerUpdateServiceOpts{
		DestinationPort: hcloud.Ptr(b.destinationPort),
		Protocol:        b.protocol,
		Proxyprotocol:   b.proxyProtocol,
	}
	if b.addHTTP {
		opts.HTTP = &hcloud.LoadBalancerUpdateServiceOptsHTTP{
			CookieName:     b.httpOpts.CookieName,
			CookieLifetime: b.httpOpts.CookieLifetime,
			RedirectHTTP:   b.httpOpts.RedirectHTTP,
			Certificates:   b.httpOpts.Certificates,
			StickySessions: b.httpOpts.StickySessions,
		}
	}
	if b.addHealthCheck {
		port := b.healthCheckOpts.Port
		if port == nil {
			port = hcloud.Ptr(b.destinationPort)
		}
		opts.HealthCheck = &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
			Protocol: b.healthCheckOpts.Protocol,
			Interval: b.healthCheckOpts.Interval,
			Port:     port,
			Retries:  b.healthCheckOpts.Retries,
			Timeout:  b.healthCheckOpts.Timeout,
		}
		if b.healthCheckOpts.Protocol == hcloud.LoadBalancerServiceProtocolHTTP ||
			b.healthCheckOpts.Protocol == hcloud.LoadBalancerServiceProtocolHTTPS {
			opts.HealthCheck.HTTP = &hcloud.LoadBalancerUpdateServiceOptsHealthCheckHTTP{
				Domain:      b.healthCheckOpts.httpOpts.Domain,
				Path:        b.healthCheckOpts.httpOpts.Path,
				Response:    b.healthCheckOpts.httpOpts.Response,
				StatusCodes: b.healthCheckOpts.httpOpts.StatusCodes,
				TLS:         b.healthCheckOpts.httpOpts.TLS,
			}
		}
	} else {
		opts.HealthCheck = &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
			Protocol: hcloud.LoadBalancerServiceProtocolTCP,
			Port:     hcloud.Ptr(b.destinationPort),
		}
	}

	return opts, nil
}

func lbAttached(lb *hcloud.LoadBalancer, nwID int64) bool {
	for _, nw := range lb.PrivateNet {
		if nw.Network.ID == nwID {
			return true
		}
	}
	return false
}
