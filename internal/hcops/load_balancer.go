package hcops

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	hrobot "github.com/syself/hrobot-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/cache"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/lbspec"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/providerid"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/utils"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/deprecationutil"
)

const loadBalancerSubsystem = "load_balancer"

// LoadBalancerOps implements all operations regarding Hetzner Cloud Load Balancers.
type LoadBalancerOps struct {
	LBClient      hcloud.ILoadBalancerClient
	ActionClient  hcloud.IActionClient
	NetworkClient hcloud.INetworkClient
	RobotClient   hrobot.RobotClient
	LBTypeCache   *cache.Cache[hcloud.LoadBalancerType]
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
			LabelSelector: fmt.Sprintf("%s=%s", lbspec.LabelServiceUID, svc.ObjectMeta.UID),
		},
	}
	lbs, err := l.LBClient.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: api error: %w", op, err)
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
func (l *LoadBalancerOps) Create(ctx context.Context, svc *corev1.Service) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.Create"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	spec, err := lbspec.Resolve(l.Recorder, svc, l.Cfg.LoadBalancer)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if spec.Location == "" && spec.NetworkZone == "" {
		return nil, fmt.Errorf("%s: neither %s nor %s set", op, annotation.LBLocation, annotation.LBNetworkZone)
	}

	lbType, err := l.verifyType(ctx, svc, spec)
	if err != nil {
		return nil, fmt.Errorf("error getting load balancer type: %w", err)
	}

	result, _, err := l.LBClient.Create(ctx, spec.CreateOpts(lbType))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
	}
	if err := l.ActionClient.WaitFor(ctx, result.Action); err != nil {
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
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// ReconcileHCLB configures the Hetzner Cloud Load Balancer to match what is
// defined for the K8S Load Balancer svc.
func (l *LoadBalancerOps) ReconcileHCLB(ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLB"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var changed bool

	spec, err := lbspec.Resolve(l.Recorder, svc, l.Cfg.LoadBalancer)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}

	labelSet, err := l.changeHCLBInfo(ctx, lb, spec)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || labelSet

	ipv4RDNSChanged, err := l.changeIPv4RDNS(ctx, lb, spec)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || ipv4RDNSChanged

	ipv6RDNSChanged, err := l.changeIPv6RDNS(ctx, lb, spec)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || ipv6RDNSChanged

	algorithmChanged, err := l.changeAlgorithm(ctx, lb, spec)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || algorithmChanged

	typeChanged, err := l.changeType(ctx, lb, svc, spec)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || typeChanged

	networkDetached, err := l.detachFromNetwork(ctx, lb, spec)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || networkDetached

	networkAttached, err := l.attachToNetwork(ctx, lb, spec)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}
	changed = changed || networkAttached

	pubIfaceToggled, err := l.togglePublicInterface(ctx, lb, spec)
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
func (l *LoadBalancerOps) changeHCLBInfo(
	ctx context.Context,
	lb *hcloud.LoadBalancer,
	spec lbspec.Spec,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeHCLBInfo"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	opts, update := spec.UpdateOpts(lb)
	if !update {
		return false, nil
	}

	updated, _, err := l.LBClient.Update(ctx, lb, opts)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
	}
	lb.Name = updated.Name
	lb.Labels = updated.Labels

	return true, nil
}

func (l *LoadBalancerOps) changeIPv4RDNS(ctx context.Context, lb *hcloud.LoadBalancer, spec lbspec.Spec) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeIPv4RDNS"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	// If the annotation is not set, no changes are needed
	if spec.IPv4RDNS == nil {
		return false, nil
	}
	// If the annotation and the actual value match, no changes are needed
	if *spec.IPv4RDNS == lb.PublicNet.IPv4.DNSPtr {
		return false, nil
	}

	action, _, err := l.LBClient.ChangeDNSPtr(ctx, lb, lb.PublicNet.IPv4.IP.String(), spec.IPv4RDNS)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
	}
	err = l.ActionClient.WaitFor(ctx, action)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (l *LoadBalancerOps) changeIPv6RDNS(ctx context.Context, lb *hcloud.LoadBalancer, spec lbspec.Spec) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeIPv6RDNS"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	// If the annotation is not set, no changes are needed
	if spec.IPv6RDNS == nil {
		return false, nil
	}
	// If the annotation and the actual value match, no changes are needed
	if *spec.IPv6RDNS == lb.PublicNet.IPv6.DNSPtr {
		return false, nil
	}

	action, _, err := l.LBClient.ChangeDNSPtr(ctx, lb, lb.PublicNet.IPv6.IP.String(), spec.IPv6RDNS)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
	}
	err = l.ActionClient.WaitFor(ctx, action)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (l *LoadBalancerOps) changeAlgorithm(ctx context.Context, lb *hcloud.LoadBalancer, spec lbspec.Spec) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeAlgorithm"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	// An unconfigured algorithm is left alone.
	if spec.Algorithm == "" || spec.Algorithm == lb.Algorithm.Type {
		return false, nil
	}

	opts := spec.ChangeAlgorithmOpts()
	action, _, err := l.LBClient.ChangeAlgorithm(ctx, lb, opts)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
	}
	err = l.ActionClient.WaitFor(ctx, action)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (l *LoadBalancerOps) changeType(
	ctx context.Context,
	lb *hcloud.LoadBalancer,
	svc *corev1.Service,
	spec lbspec.Spec,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.changeType"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	opts := hcloud.LoadBalancerChangeTypeOpts{}

	lbType, err := l.verifyType(ctx, svc, spec)
	if err != nil {
		return false, fmt.Errorf("error getting load balancer type: %w", err)
	}

	// If the user removes the annotation, we do not downgrade the Load Balancer
	// back to its default value. This could be changed in a next major release.
	if spec.TypeUnset {
		return false, nil
	}

	opts.LoadBalancerType = lbType

	if lb.LoadBalancerType.Name == lbType.Name {
		return false, nil
	}

	action, _, err := l.LBClient.ChangeType(ctx, lb, opts)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
	}
	err = l.ActionClient.WaitFor(ctx, action)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (l *LoadBalancerOps) detachFromNetwork(ctx context.Context, lb *hcloud.LoadBalancer, spec lbspec.Spec) (bool, error) {
	const op = "hcops/LoadBalancerOps.detachFromNetwork"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var changed bool

	for _, lbpn := range lb.PrivateNet {
		// Don't detach the Load Balancer from the network it is supposed to
		// be attached to and the current private IP of the load balancer matches
		// the one configured by the user, if one is configured.
		if l.NetworkID == lbpn.Network.ID && (spec.PrivateIPv4 == nil || spec.PrivateIPv4.Equal(lbpn.IP)) {
			continue
		}
		klog.InfoS("detach from network", "op", op, "loadBalancerID", lb.ID, "networkID", lbpn.Network.ID, "privateIPv4", lbpn.IP.String())

		opts := hcloud.LoadBalancerDetachFromNetworkOpts{Network: lbpn.Network}
		a, _, err := l.LBClient.DetachFromNetwork(ctx, lb, opts)
		if err != nil {
			return changed, fmt.Errorf("%s: %w", op, err)
		}
		if err := l.ActionClient.WaitFor(ctx, a); err != nil {
			return changed, fmt.Errorf("%s: %w", op, err)
		}
		changed = true
	}
	return changed, nil
}

func (l *LoadBalancerOps) attachToNetwork(ctx context.Context, lb *hcloud.LoadBalancer, spec lbspec.Spec) (bool, error) {
	const op = "hcops/LoadBalancerOps.attachToNetwork"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	// Don't attach the Load Balancer if network is not set, or the load
	// balancer is already attached.
	if l.NetworkID == 0 || lbAttached(lb, l.NetworkID, spec.PrivateIPv4) {
		return false, nil
	}

	if spec.PrivateIPv4 != nil {
		klog.InfoS("attach to network", "op", op, "loadBalancerID", lb.ID, "networkID", l.NetworkID, "privateIP", spec.PrivateIPv4)
	} else {
		klog.InfoS("attach to network", "op", op, "loadBalancerID", lb.ID, "networkID", l.NetworkID)
	}

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
	opts := spec.AttachToNetworkOpts(nw)
	a, _, err := l.LBClient.AttachToNetwork(ctx, lb, opts)
	if hcloud.IsError(err, hcloud.ErrorCodeConflict, hcloud.ErrorCodeLocked) {
		klog.InfoS("retry due to conflict or lock",
			"op", op, "delay", fmt.Sprintf("%v", retryDelay), "err", fmt.Sprintf("%v", err))

		time.Sleep(retryDelay)
		a, _, err = l.LBClient.AttachToNetwork(ctx, lb, opts)
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
	}

	if err := l.ActionClient.WaitFor(ctx, a); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (l *LoadBalancerOps) togglePublicInterface(ctx context.Context, lb *hcloud.LoadBalancer, spec lbspec.Spec) (bool, error) {
	const op = "hcops/LoadBalancerOps.togglePublicInterface"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var a *hcloud.Action
	var err error

	// An unconfigured public interface is left alone.
	if spec.PublicInterface == nil || *spec.PublicInterface == lb.PublicNet.Enabled {
		return false, nil
	}

	if *spec.PublicInterface {
		a, _, err = l.LBClient.EnablePublicInterface(ctx, lb)
	} else {
		a, _, err = l.LBClient.DisablePublicInterface(ctx, lb)
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if err := l.ActionClient.WaitFor(ctx, a); err != nil {
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

	spec, err := lbspec.Resolve(l.Recorder, svc, l.Cfg.LoadBalancer)
	if err != nil {
		return changed, fmt.Errorf("%s: %w", op, err)
	}

	privateIPEnabled := spec.UsePrivateIP
	if privateIPEnabled && l.NetworkID == 0 {
		return changed, fmt.Errorf("%s: use private ip: missing network id", op)
	}

	// Extract HC server IDs of all K8S nodes assigned to the K8S cluster.
	for _, node := range nodes {
		id, isCloudServer, err := providerid.ToServerID(node.Spec.ProviderID)
		if err != nil {
			if errors.As(err, new(*providerid.UnkownPrefixError)) {
				// ProviderID has unknown prefix, cluster might have non-hccm nodes that can not be added to the
				// Load Balancer. Emitting an event and ignoring that Node in this reconciliation loop.
				utils.WarnEventLogf(
					l.Recorder,
					node,
					"UnknownProviderIDPrefix",
					"Node could not be added to Load Balancer for service %s because the provider ID does not match any known format",
					svc.Name,
				)
				continue
			}
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

	useRobotAPI := l.Cfg.Robot.Enabled && l.RobotClient != nil
	useRobotInternalIPs := l.Cfg.Robot.Enabled && l.RobotClient == nil && privateIPEnabled

	// Use Robot API to either fetch ExternalIP or use InternalIP from Node objects
	if useRobotAPI {
		dedicatedServers, err := l.RobotClient.ServerGetList()
		if err != nil {
			return changed, fmt.Errorf("%s: failed to get list of dedicated servers: %w", op, err)
		}

		for _, s := range dedicatedServers {
			// Set ExternalIP as Load Balancer target
			robotIPsToIDs[s.ServerIP] = s.ServerNumber
			robotIDToIPv4[s.ServerNumber] = s.ServerIP

			// If user does not want private IPs we can skip this part
			if !privateIPEnabled {
				continue
			}

			node, ok := k8sNodes[int64(s.ServerNumber)]
			if !ok {
				continue
			}

			// Check if InternalIP is set at Node object
			internalIP := getNodeInternalIP(node)
			if internalIP == "" {
				utils.WarnEventLogf(
					l.Recorder,
					svc,
					"InternalIPNotConfigured",
					"%s: load balancer %s has set `use-private-ip: true`, but no InternalIP found for node %s. Continuing with ExternalIP.",
					op,
					svc.Name,
					node.Name,
				)
				continue
			}

			// Overwrite ExternalIP with InternalIP
			robotIPsToIDs[internalIP] = s.ServerNumber
			robotIDToIPv4[s.ServerNumber] = internalIP
		}
	}

	// Use InternalIPs for Robot servers without querying the API
	if useRobotInternalIPs {
		// No Robot client: derive IP mapping directly from Kubernetes Node
		// objects. This works when the node's InternalIP is the correct
		// target (e.g. vSwitch private IP).
		for id := range k8sNodeIDsRobot {
			node, ok := k8sNodes[int64(id)]
			if !ok {
				continue
			}

			internalIP := getNodeInternalIP(node)
			if internalIP == "" {
				utils.WarnEventLogf(
					l.Recorder,
					svc,
					"InternalIPNotConfigured",
					"no InternalIP found for Robot node %s (id=%d), cannot add as LB target without Robot credentials; skipping",
					node.Name,
					id,
				)
				continue
			}

			robotIPsToIDs[internalIP] = id
			robotIDToIPv4[id] = internalIP
		}
	}

	numberOfTargets := len(lb.Targets)

	// Extract IDs of the hc Load Balancer's server targets. Along the way,
	// Remove all server targets from the HC Load Balancer which are currently
	// not assigned as nodes to the K8S Load Balancer.
	for _, target := range lb.Targets {
		if target.Type == hcloud.LoadBalancerTargetTypeServer {
			id := target.Server.Server.ID
			recreate := target.UsePrivateIP != privateIPEnabled
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
			if err := l.ActionClient.WaitFor(ctx, a); err != nil {
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
			if err := l.ActionClient.WaitFor(ctx, a); err != nil {
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
		opts := spec.AddServerTargetOpts(id)
		a, _, err := l.LBClient.AddServerTarget(ctx, lb, opts)
		if err != nil {
			if hcloud.IsError(err, hcloud.ErrorCodeResourceLimitExceeded) {
				l.emitMaxTargetsReachedError(node, svc, op)
				// Continue loop so that error is emitted for each node
				continue
			}
			return changed, fmt.Errorf("%s: target %s: %w", op, node.Name, err)
		}
		if err := l.ActionClient.WaitFor(ctx, a); err != nil {
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

			klog.InfoS("add target (robot node)", "op", op, "service", svc.ObjectMeta.Name, "targetName", node.Name, "ip", ip)
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
			if err := l.ActionClient.WaitFor(ctx, a); err != nil {
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

// ReconcileHCLBServices synchronizes services exposed by the Hetzner Cloud
// Load Balancer with the kubernetes cluster.
func (l *LoadBalancerOps) ReconcileHCLBServices(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLBServices"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var changed bool

	spec, err := lbspec.Resolve(l.Recorder, svc, l.Cfg.LoadBalancer)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if err := l.reconcileManagedCertificate(ctx, spec); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	// Resolved once for the whole Service, as every port uses the same
	// certificates.
	certificates, err := l.resolveCertificates(ctx, svc, spec)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	hclbListenPorts := make(map[int]bool, len(lb.Services))
	for _, hclbService := range lb.Services {
		hclbListenPorts[hclbService.ListenPort] = true
	}

	// Add all ports exposed by the K8S Load Balancer service to the HC load
	// balancer. Remove the ports from the set of HC Load Balancer listen
	// ports.
	for _, port := range svc.Spec.Ports {
		var action *hcloud.Action

		if port.Protocol != "" && port.Protocol != corev1.ProtocolTCP {
			utils.WarnEventLogf(
				l.Recorder,
				svc,
				"UnsupportedProtocolConfigured",
				"configured unsupported Hetzner Cloud load balancer protocol %s for service with name %s",
				port.Protocol,
				svc.Name,
			)
			continue
		}

		portNo := int(port.Port)
		portExists := hclbListenPorts[portNo]
		delete(hclbListenPorts, portNo)

		if portExists {
			klog.InfoS("update service", "op", op, "port", portNo, "loadBalancerID", lb.ID)

			opts := spec.Service.UpdateServiceOpts(port, certificates)
			action, _, err = l.LBClient.UpdateService(ctx, lb, portNo, opts)
			if err != nil {
				return changed, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
			}
		} else {
			klog.InfoS("add service", "op", op, "port", portNo, "loadBalancerID", lb.ID)

			opts := spec.Service.AddServiceOpts(port, certificates)
			action, _, err = l.LBClient.AddService(ctx, lb, opts)
			if err != nil {
				return changed, fmt.Errorf("%s: %w", op, withInvalidInputFields(err))
			}
		}

		if err = l.ActionClient.WaitFor(ctx, action); err != nil {
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
		err = l.ActionClient.WaitFor(ctx, a)
		if err != nil {
			return changed, fmt.Errorf("%s: port: %d: %w", op, p, err)
		}
		changed = true
	}

	return changed, nil
}

func (l *LoadBalancerOps) reconcileManagedCertificate(
	ctx context.Context,
	spec lbspec.Spec,
) error {
	const op = "hcops/LoadBalancerOps.reconcileManagedCertificate"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if spec.ManagedCertificate == nil {
		return nil
	}

	err := l.CertOps.CreateManagedCertificate(ctx, spec.ManagedCertificate.CreateOpts())
	if errors.Is(err, ErrAlreadyExists) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// resolveCertificates turns the certificate references of the Service into
// references the API accepts, which means looking up certificates referenced by
// name. A managed certificate is looked up by the label it was created with.
//
// It returns nil when the Service has no certificates configured.
func (l *LoadBalancerOps) resolveCertificates(
	ctx context.Context,
	svc *corev1.Service,
	spec lbspec.Spec,
) ([]*hcloud.Certificate, error) {
	const op = "hcops/LoadBalancerOps.resolveCertificates"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if spec.ManagedCertificate != nil {
		cert, err := l.CertOps.GetCertificateByLabel(ctx, fmt.Sprintf("%s=%s", lbspec.LabelServiceUID, svc.ObjectMeta.UID))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		return []*hcloud.Certificate{{ID: cert.ID}}, nil
	}

	if spec.Service.HTTP == nil || len(spec.Service.HTTP.Certificates) == 0 {
		return nil, nil
	}

	resolved := make([]*hcloud.Certificate, len(spec.Service.HTTP.Certificates))
	for i, c := range spec.Service.HTTP.Certificates {
		if c.ID != 0 {
			resolved[i] = c
			continue
		}

		cert, err := l.CertOps.GetCertificateByNameOrID(ctx, c.Name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		resolved[i] = &hcloud.Certificate{ID: cert.ID}
	}
	return resolved, nil
}

// verifyType looks up the Load Balancer type requested by spec and warns about a
// type that is unconfigured, deprecated or unavailable.
func (l *LoadBalancerOps) verifyType(
	ctx context.Context,
	svc *corev1.Service,
	spec lbspec.Spec,
) (*hcloud.LoadBalancerType, error) {
	ctx = cache.SetSubsystem(ctx, loadBalancerSubsystem)

	if spec.TypeUnset {
		utils.WarnEventLogf(
			l.Recorder,
			svc,
			"LoadBalancerTypeUnconfigured",
			"Load Balancer Type unconfigured: this will be required in the future, set it with the annotation %q or cluster-wide with the environment variable %q",
			annotation.LBType,
			config.HcloudLoadBalancersType,
		)
	}

	lbType, err := l.LBTypeCache.ByName(ctx, spec.Type)
	if err != nil {
		return nil, err
	}

	if lbType == nil {
		return nil, fmt.Errorf("load balancer type not found: %s", spec.Type)
	}

	msg, unavailable := deprecationutil.LoadBalancerTypeMessage(lbType)
	if unavailable {
		return nil, errors.New(msg)
	}
	if msg != "" {
		utils.WarnEventLogf(
			l.Recorder,
			svc,
			"LoadBalancerTypeDeprecated",
			"%s", msg,
		)
	}

	return lbType, nil
}

func lbAttached(lb *hcloud.LoadBalancer, nwID int64, privateIPv4 net.IP) bool {
	for _, nw := range lb.PrivateNet {
		if nw.Network.ID == nwID && (privateIPv4 == nil || privateIPv4.Equal(nw.IP)) {
			return true
		}
	}
	return false
}

func getNodeInternalIP(node *corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}
