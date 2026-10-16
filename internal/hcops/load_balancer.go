package hcops

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
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
	const op = "hcops/LoadBalancerOps.GetByID"
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
// svc is only used to report events about the created Load Balancer.
func (l *LoadBalancerOps) Create(
	ctx context.Context, svc *corev1.Service, spec lbspec.Spec,
) (*hcloud.LoadBalancer, error) {
	const op = "hcops/LoadBalancerOps.Create"
	metrics.OperationCalled.WithLabelValues(op).Inc()

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

	result, _, err := l.LBClient.DeleteWithResult(ctx, lb)
	if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = l.ActionClient.WaitFor(ctx, result.Action)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ReconcileHCLB configures the Hetzner Cloud Load Balancer to match spec.
//
// svc is only used to report events about the Load Balancer.
func (l *LoadBalancerOps) ReconcileHCLB(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service, spec lbspec.Spec,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLB"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var changed bool

	l.reconcilePrimaryIPs(lb, svc, spec)

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

func (l *LoadBalancerOps) reconcilePrimaryIPs(
	lb *hcloud.LoadBalancer,
	svc *corev1.Service,
	spec lbspec.Spec,
) {
	if spec.PrimaryIPv4 != 0 && spec.PrimaryIPv4 != lb.PublicNet.IPv4.ID {
		utils.WarnEventLogf(
			l.Recorder,
			svc,
			"PrimaryIPChangeUnsupported",
			"Load Balancer has Primary IPv4 %d, changing it to %d is not supported",
			lb.PublicNet.IPv4.ID,
			spec.PrimaryIPv4,
		)
	}

	if spec.PrimaryIPv6 != 0 && spec.PrimaryIPv6 != lb.PublicNet.IPv6.ID {
		utils.WarnEventLogf(
			l.Recorder,
			svc,
			"PrimaryIPChangeUnsupported",
			"Load Balancer has Primary IPv6 %d, changing it to %d is not supported",
			lb.PublicNet.IPv6.ID,
			spec.PrimaryIPv6,
		)
	}
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
		if l.NetworkID == lbpn.Network.ID && (!spec.PrivateIPv4.IsValid() || sameIP(spec.PrivateIPv4, lbpn.IP)) {
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

	if spec.PrivateIPv4.IsValid() {
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

		select {
		case <-ctx.Done():
			return false, fmt.Errorf("%s: %w", op, ctx.Err())
		case <-time.After(retryDelay):
		}
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
//
// Cloud Servers are added as server targets, Robot Servers as IP targets.
func (l *LoadBalancerOps) ReconcileHCLBTargets(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service, spec lbspec.Spec, nodes []*corev1.Node,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLBTargets"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if spec.UsePrivateIP && l.NetworkID == 0 {
		return false, fmt.Errorf("%s: use private ip: missing network id", op)
	}

	serverTargets, ipTargets, err := l.desiredTargets(svc, spec.UsePrivateIP, nodes)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	var changed bool
	numberOfTargets := len(lb.Targets)
	existingServerTargets := make(map[int64]bool)
	existingIPTargets := make(map[string]bool)

	for _, target := range lb.Targets {
		var (
			targetName string
			action     *hcloud.Action
			err        error
		)

		switch target.Type {
		case hcloud.LoadBalancerTargetTypeServer:
			id := target.Server.Server.ID
			node, desired := serverTargets[id]
			if desired && target.UsePrivateIP == spec.UsePrivateIP {
				existingServerTargets[id] = true
				continue
			}

			targetName = strconv.FormatInt(id, 10)
			if node != nil {
				targetName = node.Name
			}

			klog.InfoS("remove target", "op", op, "service", svc.Name, "targetName", targetName)
			action, _, err = l.LBClient.RemoveServerTarget(ctx, lb, target.Server.Server)
		case hcloud.LoadBalancerTargetTypeIP:
			ip := target.IP.IP
			if _, desired := ipTargets[ip]; desired {
				existingIPTargets[ip] = true
				continue
			}

			targetName = ip

			klog.InfoS("remove target", "op", op, "service", svc.Name, "targetIP", ip)
			action, _, err = l.LBClient.RemoveIPTarget(ctx, lb, net.ParseIP(ip))
		default:
			continue
		}

		if err == nil {
			err = l.ActionClient.WaitFor(ctx, action)
		}
		if err != nil {
			return changed, fmt.Errorf("%s: target: %s: %w", op, targetName, err)
		}
		changed = true
		numberOfTargets--
	}

	var pending []pendingTarget
	for id, node := range serverTargets {
		if existingServerTargets[id] {
			continue
		}
		pending = append(pending, pendingTarget{
			node: node,
			add: func() (*hcloud.Action, *hcloud.Response, error) {
				klog.InfoS("add target", "op", op, "service", svc.Name, "targetName", node.Name)
				return l.LBClient.AddServerTarget(ctx, lb, spec.AddServerTargetOpts(id))
			},
		})
	}
	for ip, node := range ipTargets {
		if existingIPTargets[ip] {
			continue
		}
		pending = append(pending, pendingTarget{
			node: node,
			add: func() (*hcloud.Action, *hcloud.Response, error) {
				klog.InfoS("add target", "op", op, "service", svc.Name, "targetName", node.Name, "ip", ip)
				return l.LBClient.AddIPTarget(ctx, lb, hcloud.LoadBalancerAddIPTargetOpts{IP: net.ParseIP(ip)})
			},
		})
	}

	for _, target := range pending {
		added, err := l.addTarget(ctx, op, lb, svc, target, numberOfTargets)
		if err != nil {
			return changed, fmt.Errorf("%s: target %s: %w", op, target.node.Name, err)
		}
		if added {
			changed = true
			numberOfTargets++
		}
	}

	return changed, nil
}

type pendingTarget struct {
	node *corev1.Node
	add  func() (*hcloud.Action, *hcloud.Response, error)
}

// addTarget returns false without an error if the target was not added because
// the Load Balancer has reached its maximum number of targets.
func (l *LoadBalancerOps) addTarget(
	ctx context.Context,
	op string,
	lb *hcloud.LoadBalancer,
	svc *corev1.Service,
	target pendingTarget,
	numberOfTargets int,
) (bool, error) {
	if numberOfTargets >= lb.LoadBalancerType.MaxTargets {
		l.emitMaxTargetsReachedError(target.node, svc, op)
		return false, nil
	}

	action, _, err := target.add()
	if hcloud.IsError(err, hcloud.ErrorCodeResourceLimitExceeded) {
		l.emitMaxTargetsReachedError(target.node, svc, op)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := l.ActionClient.WaitFor(ctx, action); err != nil {
		return false, err
	}
	return true, nil
}

// desiredTargets returns the server targets by Cloud Server ID and the IP targets by IP
// that the Load Balancer should have for the given nodes.
func (l *LoadBalancerOps) desiredTargets(
	svc *corev1.Service,
	usePrivateIP bool,
	nodes []*corev1.Node,
) (map[int64]*corev1.Node, map[string]*corev1.Node, error) {
	serverTargets := make(map[int64]*corev1.Node)
	robotNodes := make(map[int64]*corev1.Node)

	for _, node := range nodes {
		id, provider, err := providerid.ToServerID(node.Spec.ProviderID)
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
		if err != nil {
			return nil, nil, err
		}

		switch provider {
		case providerid.ProviderCloud:
			serverTargets[id] = node
		case providerid.ProviderRobot:
			if l.Cfg.Robot.Enabled {
				robotNodes[id] = node
			}
		}
	}

	robotIPs, err := l.robotExternalIPs(robotNodes)
	if err != nil {
		return nil, nil, err
	}

	ipTargets := make(map[string]*corev1.Node)
	for id, node := range robotNodes {
		externalIP, found := robotIPs[id]
		if l.RobotClient != nil && !found {
			utils.WarnEventLogf(l.Recorder, node, "ServerNotFound", "No server with id %d was found in Robot", id)
			continue
		}

		if ip := l.targetIP(svc, node, externalIP, usePrivateIP); ip != "" {
			ipTargets[ip] = node
			continue
		}
		utils.WarnEventLogf(
			l.Recorder,
			svc,
			"InternalIPNotConfigured",
			"no InternalIP found for Robot node %s (id=%d), cannot add as LB target without Robot credentials; skipping",
			node.Name,
			id,
		)
	}

	return serverTargets, ipTargets, nil
}

func (l *LoadBalancerOps) robotExternalIPs(nodes map[int64]*corev1.Node) (map[int64]string, error) {
	ips := make(map[int64]string)
	if len(nodes) == 0 || l.RobotClient == nil {
		return ips, nil
	}

	servers, err := l.RobotClient.ServerGetList()
	if err != nil {
		return nil, fmt.Errorf("failed to get list of dedicated servers: %w", err)
	}
	for _, s := range servers {
		ips[int64(s.ServerNumber)] = s.ServerIP
	}
	return ips, nil
}

// targetIP prefers the InternalIP of the Node if private IPs are enabled and falls back to externalIP.
func (l *LoadBalancerOps) targetIP(svc *corev1.Service, node *corev1.Node, externalIP string, usePrivateIP bool) string {
	if !usePrivateIP {
		return externalIP
	}

	if internalIP := getNodeInternalIP(node); internalIP != "" {
		return internalIP
	}

	if externalIP != "" {
		utils.WarnEventLogf(
			l.Recorder,
			svc,
			"InternalIPNotConfigured",
			"load balancer %s has set `use-private-ip: true`, but no InternalIP found for node %s. Continuing with ExternalIP.",
			svc.Name,
			node.Name,
		)
	}
	return externalIP
}

func (l *LoadBalancerOps) emitMaxTargetsReachedError(node *corev1.Node, svc *corev1.Service, op string) {
	l.Recorder.Eventf(node, corev1.EventTypeWarning, "MaxTargetsReached",
		"Node could not be added to Load Balancer for service %s because the max number of targets has been reached",
		svc.ObjectMeta.Name)
	klog.InfoS("cannot add server target because max number of targets have been reached", "op", op, "service", svc.ObjectMeta.Name, "targetName", node.Name)
}

// ReconcileHCLBServices synchronizes services exposed by the Hetzner Cloud
// Load Balancer with the kubernetes cluster.
func (l *LoadBalancerOps) ReconcileHCLBServices(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service, spec lbspec.Spec,
) (bool, error) {
	const op = "hcops/LoadBalancerOps.ReconcileHCLBServices"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var changed bool

	if err := l.reconcileOwnedCertificate(ctx, spec); err != nil {
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

func (l *LoadBalancerOps) reconcileOwnedCertificate(
	ctx context.Context,
	spec lbspec.Spec,
) error {
	const op = "hcops/LoadBalancerOps.reconcileOwnedCertificate"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if spec.OwnedCertificate == nil {
		return nil
	}

	err := l.CertOps.CreateManagedCertificate(ctx, spec.OwnedCertificate.CreateOpts())
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
// name. A certificate we own ourselves is looked up by the label it was created
// with.
//
// It returns nil when the Service has no certificates configured.
func (l *LoadBalancerOps) resolveCertificates(
	ctx context.Context,
	svc *corev1.Service,
	spec lbspec.Spec,
) ([]*hcloud.Certificate, error) {
	const op = "hcops/LoadBalancerOps.resolveCertificates"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if spec.OwnedCertificate != nil {
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

func lbAttached(lb *hcloud.LoadBalancer, nwID int64, privateIPv4 netip.Addr) bool {
	for _, nw := range lb.PrivateNet {
		if nw.Network.ID == nwID && (!privateIPv4.IsValid() || sameIP(privateIPv4, nw.IP)) {
			return true
		}
	}
	return false
}

// sameIP reports whether addr and ip are the same address. Both sides are
// unmapped first, so that an address written as ::ffff:10.0.0.1 equals the
// same address written as 10.0.0.1, as net.IP.Equal did.
func sameIP(addr netip.Addr, ip net.IP) bool {
	other, ok := netip.AddrFromSlice(ip)
	return ok && addr.Unmap() == other.Unmap()
}

func getNodeInternalIP(node *corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}
