package hcloud

import (
	"context"
	"fmt"
	"net"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/cache"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/providerid"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// routeTargetCacheMaxAge overrides the shared server cache's short default max age
// for route reconciliation. The routes controller only reads the server's
// slow-changing private-net attachment, so it can tolerate staler entries and skip
// an extra API call. ListRoutes refreshes the cache first, so CreateRoute has less
// need for an additional refresh.
const routeTargetCacheMaxAge = 1 * time.Minute

type routes struct {
	client      *hcloud.Client
	network     *hcloud.Network
	serverCache *cache.Cache[hcloud.Server]
	clusterCIDR *net.IPNet
	recorder    record.EventRecorder
	nodeLister  corelisters.NodeLister
}

func newRoutes(client *hcloud.Client, networkID int64, clusterCIDR string, recorder record.EventRecorder, nodeLister corelisters.NodeLister, serverCache *cache.Cache[hcloud.Server]) (*routes, error) {
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
		client:      client,
		network:     networkObj,
		serverCache: serverCache,
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
	ctx = cache.SetSubsystem(ctx, "routes")

	if err := r.reloadNetwork(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	servers, err := r.serverCache.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: error fetching servers: %w", op, err)
	}

	serversByPrivateIP := make(map[string]*hcloud.Server, len(servers))
	for _, server := range servers {
		if privateNet := server.PrivateNetFor(r.network); privateNet != nil {
			serversByPrivateIP[privateNet.IP.String()] = server
		}
	}

	routes := make([]*cloudprovider.Route, 0, len(r.network.Routes))
	for _, route := range r.network.Routes {
		cpRoute := &cloudprovider.Route{
			DestinationCIDR: route.Destination.String(),
			Name:            fmt.Sprintf("%s-%s", route.Gateway.String(), route.Destination.String()),
		}

		server, ok := serversByPrivateIP[route.Gateway.String()]
		if ok {
			cpRoute.TargetNode = types.NodeName(server.Name)
		} else {
			// Route belongs to non-existing target
			cpRoute.Blackhole = true
		}
		routes = append(routes, cpRoute)
	}

	return routes, nil
}

// CreateRoute creates the described managed route
// route.Name will be ignored, although the cloud-provider may use nameHint
// to create a more user-meaningful name.
func (r *routes) CreateRoute(ctx context.Context, _ string, _ string, route *cloudprovider.Route) error {
	const op = "hcloud/CreateRoute"
	metrics.OperationCalled.WithLabelValues(op).Inc()
	ctx = cache.SetSubsystem(ctx, "routes")

	// Parse and return early if we detect IPv6 routes.
	// Private Networks don't support IPv6, so we can save an API
	// request by validating beforehand.
	ip, ipNet, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if ip.To4() == nil {
		return fmt.Errorf(
			"%s: can't create route %q via node %q: private networks do not support IPv6",
			op,
			ipNet.String(),
			route.TargetNode,
		)
	}

	node, gateway, err := r.resolveRouteTarget(ctx, string(route.TargetNode))
	if err != nil {
		return fmt.Errorf("%s: error resolving route target: %w", op, err)
	}

	if !slices.ContainsFunc(route.TargetNodeAddresses, func(target corev1.NodeAddress) bool {
		return target.Type == corev1.NodeInternalIP && target.Address == gateway.String()
	}) {
		return fmt.Errorf("%s: IP %s not part of routes target addresses", op, gateway.String())
	}

	r.warnCIDRMismatch(ipNet, node)

	if err := r.upsertRoute(ctx, gateway, ipNet, string(route.TargetNode)); err != nil {
		return fmt.Errorf("error upserting route %q via %q: %w", ipNet.String(), gateway.String(), err)
	}

	return nil
}

// resolveRouteTarget returns the k8s node and the hcloud server's private IP on the routes
// network — everything needed to create a route for this node (gateway IP) and record events
// against it (node).
//
// The hcloud server is resolved by ProviderID. Nodes without a ProviderID yet are
// looked up by name as a fallback. Refreshes the cache once if the
// private-net attachment isn't yet reflected.
func (r *routes) resolveRouteTarget(ctx context.Context, nodeName string) (*corev1.Node, net.IP, error) {
	node, err := r.nodeLister.Get(nodeName)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching node %s by name: %w", nodeName, err)
	}

	var server *hcloud.Server
	if node.Spec.ProviderID != "" {
		id, isCloudServer, err := providerid.ToServerID(node.Spec.ProviderID)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing provider id %q for node %s: %w", node.Spec.ProviderID, nodeName, err)
		}
		if !isCloudServer {
			return nil, nil, fmt.Errorf("node %s is not a cloud server, routes are only supported for cloud servers", node.Name)
		}
		server, err = r.serverCache.ByID(ctx, id, cache.WithMaxAge(routeTargetCacheMaxAge))
		if err != nil {
			return nil, nil, fmt.Errorf("error looking up hcloud server by id %d for node %s: %w", id, nodeName, err)
		}
	} else {
		server, err = r.serverCache.ByName(ctx, node.Name, cache.WithMaxAge(routeTargetCacheMaxAge))
		if err != nil {
			return nil, nil, fmt.Errorf("error looking up hcloud server by name for node %s: %w", nodeName, err)
		}
	}

	// The cache returns (nil, nil) when the server does not exist (e.g. it was deleted).
	if server == nil {
		return nil, nil, fmt.Errorf("hcloud server for node %s not found", nodeName)
	}

	// CreateRoute may fail if the Server is not yet attached to the
	// Private Network. In that case it returns an error and is retried;
	// ListRoutes runs first and refreshes the cache.
	privNet := server.PrivateNetFor(r.network)
	if privNet == nil {
		return nil, nil, fmt.Errorf("server %s (%d): network with id %d not attached to this server", server.Name, server.ID, r.network.ID)
	}

	return node, privNet.IP, nil
}

// upsertRoute ensures the hcloud network has a route for cidr pointing at gateway. A matching
// route is a no-op; a stale route with a different gateway is replaced in place. nodeName is
// used only for logging and for surfacing API conflicts against the right k8s object.
func (r *routes) upsertRoute(ctx context.Context, gateway net.IP, cidr *net.IPNet, nodeName string) error {
	if err := r.reloadNetwork(ctx); err != nil {
		return fmt.Errorf("error reloading network: %w", err)
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
			return fmt.Errorf("error deleting route for %q via %q: %w", cidr.String(), gateway.String(), err)
		}
		if err := r.client.Action.WaitFor(ctx, action); err != nil {
			return fmt.Errorf("error deleting route for %q via %q: %w", cidr.String(), gateway.String(), err)
		}
		klog.InfoS(
			"deleted stale route with wrong gateway; recreating",
			"node", nodeName,
			"gateway", gateway,
			"cidr", destination,
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
		return fmt.Errorf("error adding route for %q via %q: %w", cidr.String(), gateway.String(), err)
	}

	if err := r.client.Action.WaitFor(ctx, action); err != nil {
		return fmt.Errorf("error adding route for %q via %q: %w", cidr.String(), gateway.String(), err)
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
