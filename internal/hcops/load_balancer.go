package hcops

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-go/hcloud"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// LabelServiceUID is a label added to the Hetzner Cloud backend to uniquely
// identify a load balancer managed by Hetzner Cloud Cloud Controller Manager.
const LabelServiceUID = "hcloud-ccm/service-uid"

// HCloudLoadBalancerClient defines the hcloud-go functions required by the
// Load Balancer operations type.
type HCloudLoadBalancerClient interface {
	GetByID(ctx context.Context, id int) (*hcloud.LoadBalancer, *hcloud.Response, error)
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

	AddServerTarget(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServerTargetOpts) (*hcloud.Action, *hcloud.Response, error)
	RemoveServerTarget(ctx context.Context, lb *hcloud.LoadBalancer, server *hcloud.Server) (*hcloud.Action, *hcloud.Response, error)

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
	CertClient    HCloudCertificateClient
	RetryDelay    time.Duration
	NetworkID     int
	Defaults      LoadBalancerDefaults
}

// LoadBalancerDefaults stores cluster-wide default values for load balancers.
type LoadBalancerDefaults struct {
	Location              string
	NetworkZone           string
	DisablePrivateIngress bool
	UsePrivateIP          bool
}

// GetByK8SServiceUID tries to find a Load Balancer by its Kubernetes service
// UID.
//
// If no Load Balancer could be found ErrNotFound is returned. Likewise,
// ErrNonUniqueResult is returned if more than one matching Load Balancer is
// found.
func (l *LoadBalancerOps) GetByK8SServiceUID(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.GetByK8SServiceUID"

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
func (l *LoadBalancerOps) GetByID(ctx context.Context, id int) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.GetByName"

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
	ctx context.Context, lbName string, svc *v1.Service,
) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.Create"

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
	if v, ok := annotation.LBLocation.StringFromService(svc); ok {
		opts.Location = &hcloud.Location{Name: v}
	}
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
		opts.PublicInterface = hcloud.Bool(false)
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
func (l *LoadBalancerOps) ReconcileHCLB(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLB"
	var changed bool

	labelSet, err := l.changeHCLBInfo(ctx, lb, svc)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || labelSet

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
func (l *LoadBalancerOps) changeHCLBInfo(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeHCLBInfo"
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

func (l *LoadBalancerOps) changeAlgorithm(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeAlgorithm"

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

func (l *LoadBalancerOps) changeType(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeType"

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

func (l *LoadBalancerOps) togglePublicInterface(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.togglePublicInterface"
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
	ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service, nodes []*v1.Node,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLBTargets"
	var (
		// Set of all K8S server IDs currently assigned as nodes to this
		// cluster.
		k8sNodeIDs   = make(map[int]bool)
		k8sNodeNames = make(map[int]string)

		// Set of server IDs assigned as targets to the HC Load Balancer. Some
		// of the entries may get deleted during reconcilement. In this case
		// the hclbTargetIDs[id] is always false. If hclbTargetIDs[id] is true,
		// the node with this server id is assigned to the K8S cluster.
		hclbTargetIDs = make(map[int]bool)

		changed bool
	)

	usePrivateIP, err := annotation.LBUsePrivateIP.BoolFromService(svc)
	if err != nil && !errors.Is(err, annotation.ErrNotSet) {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	if usePrivateIP && l.NetworkID == 0 {
		return changed, fmt.Errorf("%s: use private ip: missing network id", op)
	}

	// Extract HC server IDs of all K8S nodes assigned to the K8S cluster.
	for _, node := range nodes {
		id, err := providerIDToServerID(node.Spec.ProviderID)
		if err != nil {
			return changed, fmt.Errorf("%s: %w", op, err)
		}
		k8sNodeIDs[id] = true
		k8sNodeNames[id] = node.Name
	}

	// Extract IDs of the hc Load Balancer's server targets. Along the way,
	// Remove all server targets from the HC Load Balancer which are currently
	// not assigned as nodes to the K8S Load Balancer.
	for _, target := range lb.Targets {
		if target.Type != hcloud.LoadBalancerTargetTypeServer {
			continue
		}

		id := target.Server.Server.ID
		recreate := target.UsePrivateIP != usePrivateIP
		hclbTargetIDs[id] = k8sNodeIDs[id] && !recreate
		if hclbTargetIDs[id] {
			continue
		}

		klog.InfoS("remove target", "op", op, "service", svc.ObjectMeta.Name, "targetName", k8sNodeNames[id])
		// Target needs to be re-created or node currently not in use by k8s
		// Load Balancer. Remove it from the HC Load Balancer
		a, _, err := l.LBClient.RemoveServerTarget(ctx, lb, target.Server.Server)
		if err != nil {
			return changed, fmt.Errorf("%s: target: %s: %w", op, k8sNodeNames[id], err)
		}
		if err := WatchAction(ctx, l.ActionClient, a); err != nil {
			return changed, fmt.Errorf("%s: target: %s: %w", op, k8sNodeNames[id], err)
		}
		changed = true
	}

	// Assign the servers which are currently assigned as nodes
	// to the K8S Load Balancer as server targets to the HC Load Balancer.
	for id := range k8sNodeIDs {
		// Don't assign the node again if it is already assigned to the HC load
		// balancer.
		if hclbTargetIDs[id] {
			continue
		}

		klog.InfoS("add target", "op", op, "service", svc.ObjectMeta.Name, "targetName", k8sNodeNames[id])
		opts := hcloud.LoadBalancerAddServerTargetOpts{
			Server:       &hcloud.Server{ID: id},
			UsePrivateIP: &usePrivateIP,
		}
		a, _, err := l.LBClient.AddServerTarget(ctx, lb, opts)
		if err != nil {
			return changed, fmt.Errorf("%s: target %s: %w", op, k8sNodeNames[id], err)
		}
		if err := WatchAction(ctx, l.ActionClient, a); err != nil {
			return changed, fmt.Errorf("%s: target %s: %w", op, k8sNodeNames[id], err)
		}
		changed = true
	}

	return changed, nil
}

// ReconcileHCLBServices synchronizes services exposed by the Hetzner Cloud
// Load Balancer with the kubernetes cluster.
func (l *LoadBalancerOps) ReconcileHCLBServices(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLBServices"
	var changed bool

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

		b := &hclbServiceOptsBuilder{Port: port, Service: svc, CertClient: l.CertClient}
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

type hclbServiceOptsBuilder struct {
	Port       v1.ServicePort
	Service    *v1.Service
	CertClient HCloudCertificateClient

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
		b.proxyProtocol = hcloud.Bool(pp)
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
		certs, err := annotation.LBSvcHTTPCertificates.CertificatesFromService(b.Service)
		if errors.Is(err, annotation.ErrNotSet) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		certs, err = b.resolveCertNames(ctx, certs)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		b.httpOpts.Certificates = certs
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

func (b *hclbServiceOptsBuilder) resolveCertNames(ctx context.Context, cs []*hcloud.Certificate) ([]*hcloud.Certificate, error) {
	const op = "hcops/hclbServiceOptsBuilder.resolveCertNames"

	resolved := make([]*hcloud.Certificate, len(cs))
	for i, c := range cs {
		if c.ID != 0 {
			resolved[i] = c
			continue
		}

		c, _, err := b.CertClient.Get(ctx, c.Name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		resolved[i] = &hcloud.Certificate{ID: c.ID}
	}
	return resolved, nil
}

func (b *hclbServiceOptsBuilder) extractHealthCheck() {
	const op = "hcops/hclbServiceOptsBuilder.extractHealthCheck"

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
		b.healthCheckOpts.Port = hcloud.Int(hcPort)
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
		b.healthCheckOpts.Interval = hcloud.Duration(hcInterval)
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
		b.healthCheckOpts.Timeout = hcloud.Duration(t)
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
		b.healthCheckOpts.Retries = hcloud.Int(v)
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

	if err := b.initialize(); err != nil {
		return hcloud.LoadBalancerAddServiceOpts{}, fmt.Errorf("%s: %w", op, err)
	}

	opts := hcloud.LoadBalancerAddServiceOpts{
		ListenPort:      hcloud.Int(b.listenPort),
		DestinationPort: hcloud.Int(b.destinationPort),
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
		opts.HealthCheck = &hcloud.LoadBalancerAddServiceOptsHealthCheck{
			Protocol: b.healthCheckOpts.Protocol,
			Interval: b.healthCheckOpts.Interval,
			Port:     b.healthCheckOpts.Port,
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
			Port:     hcloud.Int(b.destinationPort),
		}
	}

	return opts, nil
}

func (b *hclbServiceOptsBuilder) buildUpdateServiceOpts() (hcloud.LoadBalancerUpdateServiceOpts, error) {
	const op = "hcops/hclbServiceOptsBuilder.buildUpdateServiceOpts"

	if err := b.initialize(); err != nil {
		return hcloud.LoadBalancerUpdateServiceOpts{}, fmt.Errorf("%s: %w", op, err)
	}

	opts := hcloud.LoadBalancerUpdateServiceOpts{
		DestinationPort: hcloud.Int(b.destinationPort),
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
		opts.HealthCheck = &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
			Protocol: b.healthCheckOpts.Protocol,
			Interval: b.healthCheckOpts.Interval,
			Port:     b.healthCheckOpts.Port,
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
			Port:     hcloud.Int(b.destinationPort),
		}
	}

	return opts, nil
}

// TODO this is a copy of the function in hcloud/utils.go => refactor
func providerIDToServerID(providerID string) (int, error) {
	const op = "hcops/providerIDToServerID"

	providerPrefix := "hcloud://"
	if !strings.HasPrefix(providerID, providerPrefix) {
		return 0, fmt.Errorf("%s: missing prefix hcloud://: %s", op, providerID)
	}

	idString := strings.ReplaceAll(providerID, providerPrefix, "")
	if idString == "" {
		return 0, fmt.Errorf("%s: missing serverID: %s", op, providerID)
	}

	id, err := strconv.Atoi(idString)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid serverID: %s", op, providerID)
	}
	return id, nil
}

func lbAttached(lb *hcloud.LoadBalancer, nwID int) bool {
	for _, nw := range lb.PrivateNet {
		if nw.Network.ID == nwID {
			return true
		}
	}
	return false
}
