package hcloud

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"time"

	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/providerid"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var serversCacheMissRefreshRate = rate.Every(30 * time.Second)

type routes struct {
	client      *hcloud.Client
	network     *hcloud.Network
	serverCache *hcops.AllServersCache
	clusterCIDR *net.IPNet
	recorder    record.EventRecorder
	nodeLister  corelisters.NodeLister
}

func newRoutes(client *hcloud.Client, networkID int64, clusterCIDR string, recorder record.EventRecorder, nodeLister corelisters.NodeLister) (*routes, error) {
	const op = "hcloud/newRoutes"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	networkObj, _, err := client.Network.GetByID(context.Background(), networkID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if networkObj == nil {
		return nil, fmt.Errorf("network not found: %d", networkID)
	}

	_, cidr, err := net.ParseCIDR(clusterCIDR)
	if err != nil {
		return nil, err
	}

	return &routes{
		client:  client,
		network: networkObj,
		serverCache: &hcops.AllServersCache{
			// client.Server.All will load ALL the servers in the project, even those
			// that are not part of the Kubernetes cluster.
			LoadFunc:                client.Server.All,
			Network:                 networkObj,
			CacheMissRefreshLimiter: rate.NewLimiter(serversCacheMissRefreshRate, 1),
		},
		clusterCIDR: cidr,
		recorder:    recorder,
		nodeLister:  nodeLister,
	}, nil
}

func (r *routes) reloadNetwork(ctx context.Context) error {
	const op = "hcloud/reloadNetwork"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	networkObj, _, err := r.client.Network.GetByID(ctx, r.network.ID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if networkObj == nil {
		return fmt.Errorf("network not found: %s", r.network.Name)
	}
	r.network = networkObj
	return nil
}

// ListRoutes lists all managed routes that belong to the specified clusterName.
func (r *routes) ListRoutes(ctx context.Context, _ string) ([]*cloudprovider.Route, error) {
	const op = "hcloud/ListRoutes"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if err := r.reloadNetwork(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	routes := make([]*cloudprovider.Route, 0, len(r.network.Routes))
	for _, route := range r.network.Routes {
		ro, err := r.hcloudRouteToRoute(ctx, route)
		if err != nil {
			return routes, fmt.Errorf("%s: %w", op, err)
		}
		routes = append(routes, ro)
	}
	return routes, nil
}

// CreateRoute creates the described managed route
// route.Name will be ignored, although the cloud-provider may use nameHint
// to create a more user-meaningful name.
func (r *routes) CreateRoute(ctx context.Context, _ string, _ string, route *cloudprovider.Route) error {
	const op = "hcloud/CreateRoute"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	node, gateway, err := r.resolveRouteTarget(ctx, string(route.TargetNode))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if !slices.ContainsFunc(route.TargetNodeAddresses, func(target corev1.NodeAddress) bool {
		return target.Type == corev1.NodeInternalIP && target.Address == gateway.String()
	}) {
		return fmt.Errorf("%s: IP %s not part of routes target addresses", op, gateway.String())
	}

	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	r.warnCIDRMismatch(cidr, node)

	return r.upsertRoute(ctx, gateway, cidr, string(route.TargetNode))
}

// resolveRouteTarget returns the k8s node and the hcloud server's private IP on the routes
// network — everything needed to create a route for this node (gateway IP) and record events
// against it (node).
// Looks up the server by ProviderID to survive k8s node-name drift, with a ByName fallback for
// ID changes (e.g. server recreated). Refreshes the cache once if the private-net attachment
// isn't yet reflected.
func (r *routes) resolveRouteTarget(ctx context.Context, nodeName string) (*corev1.Node, net.IP, error) {
	const op = "hcloud/resolveRouteTarget"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	node, err := r.nodeLister.Get(nodeName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	if node.Spec.ProviderID == "" {
		return nil, nil, fmt.Errorf("%s: node %q not yet initialized", op, node.Name)
	}

	id, isCloudServer, err := providerid.ToServerID(node.Spec.ProviderID)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}
	if !isCloudServer {
		return nil, nil, fmt.Errorf("%s: node %q is not a Cloud server, routes are only supported for Cloud servers", op, node.Name)
	}

	server, err := r.serverCache.ByID(ctx, id)
	if errors.Is(err, hcops.ErrNotFound) {
		server, err = r.serverCache.ByName(ctx, node.Name)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	privNet, ok := findServerPrivateNetByID(server, r.network.ID)
	if !ok {
		r.serverCache.InvalidateCache()
		server, err = r.serverCache.ByID(ctx, server.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}
		privNet, ok = findServerPrivateNetByID(server, r.network.ID)
		if !ok {
			return nil, nil, fmt.Errorf("%s: server %q: network with id %d not attached to this server", op, node.Name, r.network.ID)
		}
	}

	return node, privNet.IP, nil
}

// upsertRoute ensures the hcloud network has a route for cidr pointing at gateway. A matching
// route is a no-op; a stale route with a different gateway is replaced in place. nodeName is
// used only for logging and for surfacing API conflicts against the right k8s object.
func (r *routes) upsertRoute(ctx context.Context, gateway net.IP, cidr *net.IPNet, nodeName string) error {
	const op = "hcloud/upsertRoute"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if err := r.reloadNetwork(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	destination := cidr.String()
	existingIdx := slices.IndexFunc(r.network.Routes, func(nr hcloud.NetworkRoute) bool {
		return nr.Destination.String() == destination
	})
	if existingIdx >= 0 {
		existing := r.network.Routes[existingIdx]
		if existing.Gateway.Equal(gateway) {
			klog.InfoS(
				"route already exists: skipping creation",
				"target-node", nodeName,
				"destination-cidr", destination,
			)
			return nil
		}

		action, _, err := r.client.Network.DeleteRoute(ctx, r.network, hcloud.NetworkDeleteRouteOpts{
			Route: existing,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		if err := r.client.Action.WaitFor(ctx, action); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		klog.InfoS(
			"deleted stale route with wrong gateway; recreating",
			"target-node", nodeName,
			"destination-cidr", destination,
		)
	}

	opts := hcloud.NetworkAddRouteOpts{
		Route: hcloud.NetworkRoute{
			Destination: cidr,
			Gateway:     gateway,
		},
	}
	action, _, err := r.client.Network.AddRoute(ctx, r.network, opts)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeLocked, hcloud.ErrorCodeConflict) {
			return apierrors.NewConflict(
				corev1.Resource("nodes"),
				nodeName,
				err,
			)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := r.client.Action.WaitFor(ctx, action); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// DeleteRoute deletes the specified managed route
// Route should be as returned by ListRoutes.
func (r *routes) DeleteRoute(ctx context.Context, _ string, route *cloudprovider.Route) error {
	const op = "hcloud/DeleteRoute"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	// Get target IP from current list of routes, routes can be uniquely identified by their destination cidr.
	var ip net.IP
	for _, cloudRoute := range r.network.Routes {
		if cloudRoute.Destination.String() == route.DestinationCIDR {
			ip = cloudRoute.Gateway
			break
		}
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("%s: route %s not found in cloud network routes", op, route.Name)
	}

	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	opts := hcloud.NetworkDeleteRouteOpts{
		Route: hcloud.NetworkRoute{
			Destination: cidr,
			Gateway:     ip,
		},
	}

	action, _, err := r.client.Network.DeleteRoute(ctx, r.network, opts)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if err := r.client.Action.WaitFor(ctx, action); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *routes) hcloudRouteToRoute(ctx context.Context, route hcloud.NetworkRoute) (*cloudprovider.Route, error) {
	const op = "hcloud/hcloudRouteToRoute"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	cpRoute := &cloudprovider.Route{
		DestinationCIDR: route.Destination.String(),
		Name:            fmt.Sprintf("%s-%s", route.Gateway.String(), route.Destination.String()),
	}

	srv, err := r.serverCache.ByPrivateIP(ctx, route.Gateway)
	if err != nil {
		if errors.Is(err, hcops.ErrNotFound) {
			// Route belongs to non-existing target
			cpRoute.Blackhole = true
			return cpRoute, nil
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	cpRoute.TargetNode = types.NodeName(srv.Name)
	return cpRoute, nil
}

func (r *routes) warnCIDRMismatch(cidr *net.IPNet, node *corev1.Node) {
	clusterPrefixLen, _ := r.clusterCIDR.Mask.Size()
	destPrefixLen, _ := cidr.Mask.Size()

	if !r.clusterCIDR.Contains(cidr.IP) || destPrefixLen < clusterPrefixLen {
		warnMsg := fmt.Sprintf(
			"route CIDR %s is not contained within cluster CIDR %s",
			cidr.String(),
			r.clusterCIDR.String(),
		)
		klog.Warning(warnMsg)
		r.recorder.Event(node, corev1.EventTypeWarning, "ClusterCIDRMisconfigured", warnMsg)
	}
}

func findServerPrivateNetByID(srv *hcloud.Server, id int64) (hcloud.ServerPrivateNet, bool) {
	for _, n := range srv.PrivateNet {
		if n.Network.ID == id {
			return n, true
		}
	}
	return hcloud.ServerPrivateNet{}, false
}
