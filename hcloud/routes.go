package hcloud

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type routes struct {
	client      *hcloud.Client
	network     *hcloud.Network
	serverCache *hcops.AllServersCache
}

func newRoutes(client *hcloud.Client, networkID int64) (*routes, error) {
	const op = "hcloud/newRoutes"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	networkObj, _, err := client.Network.GetByID(context.Background(), networkID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if networkObj == nil {
		return nil, fmt.Errorf("network not found: %d", networkID)
	}

	return &routes{
		client:  client,
		network: networkObj,
		serverCache: &hcops.AllServersCache{
			// client.Server.All will load ALL the servers in the project, even those
			// that are not part of the Kubernetes cluster.
			LoadFunc: client.Server.All,
			Network:  networkObj,
		},
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
		ro, err := r.hcloudRouteToRoute(route)
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
func (r *routes) CreateRoute(ctx context.Context, clusterName string, nameHint string, route *cloudprovider.Route) error {
	const op = "hcloud/CreateRoute"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	srv, err := r.serverCache.ByName(string(route.TargetNode))
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	privNet, ok := findServerPrivateNetByID(srv, r.network.ID)
	if !ok {
		r.serverCache.InvalidateCache()
		srv, err = r.serverCache.ByName(string(route.TargetNode))
		if err != nil {
			return fmt.Errorf("%s: %v", op, err)
		}

		privNet, ok = findServerPrivateNetByID(srv, r.network.ID)
		if !ok {
			return fmt.Errorf("%s: server %v: network with id %d not attached to this server", op, route.TargetNode, r.network.ID)
		}
	}
	ip := privNet.IP

	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	doesRouteAlreadyExist, err := r.checkIfRouteAlreadyExists(ctx, route)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if !doesRouteAlreadyExist {
		opts := hcloud.NetworkAddRouteOpts{
			Route: hcloud.NetworkRoute{
				Destination: cidr,
				Gateway:     ip,
			},
		}
		action, _, err := r.client.Network.AddRoute(ctx, r.network, opts)
		if err != nil {
			if hcloud.IsError(err, hcloud.ErrorCodeLocked) || hcloud.IsError(err, hcloud.ErrorCodeConflict) {
				retryDelay := time.Second * 5
				klog.InfoS("retry due to conflict or lock",
					"op", op, "delay", fmt.Sprintf("%v", retryDelay), "err", fmt.Sprintf("%v", err))
				time.Sleep(retryDelay)

				return r.CreateRoute(ctx, clusterName, nameHint, route)
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		if err := hcops.WatchAction(ctx, &r.client.Action, action); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
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

	err = r.deleteRouteFromHcloud(ctx, cidr, ip)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *routes) deleteRouteFromHcloud(ctx context.Context, cidr *net.IPNet, ip net.IP) error {
	const op = "hcloud/deleteRouteFromHcloud"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	opts := hcloud.NetworkDeleteRouteOpts{
		Route: hcloud.NetworkRoute{
			Destination: cidr,
			Gateway:     ip,
		},
	}

	action, _, err := r.client.Network.DeleteRoute(ctx, r.network, opts)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeLocked) || hcloud.IsError(err, hcloud.ErrorCodeConflict) {
			retryDelay := time.Second * 5
			klog.InfoS("retry due to conflict or lock",
				"op", op, "delay", fmt.Sprintf("%v", retryDelay), "err", fmt.Sprintf("%v", err))
			time.Sleep(retryDelay)

			return r.deleteRouteFromHcloud(ctx, cidr, ip)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	if err := hcops.WatchAction(ctx, &r.client.Action, action); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *routes) hcloudRouteToRoute(route hcloud.NetworkRoute) (*cloudprovider.Route, error) {
	const op = "hcloud/hcloudRouteToRoute"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	cpRoute := &cloudprovider.Route{
		DestinationCIDR: route.Destination.String(),
		Name:            fmt.Sprintf("%s-%s", route.Gateway.String(), route.Destination.String()),
	}

	srv, err := r.serverCache.ByPrivateIP(route.Gateway)
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

func (r *routes) checkIfRouteAlreadyExists(ctx context.Context, route *cloudprovider.Route) (bool, error) {
	const op = "hcloud/checkIfRouteAlreadyExists"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if err := r.reloadNetwork(ctx); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	for _, _route := range r.network.Routes {
		if _route.Destination.String() == route.DestinationCIDR {
			srv, err := r.serverCache.ByName(string(route.TargetNode))
			if err != nil {
				return false, fmt.Errorf("%s: %v", op, err)
			}
			privNet, ok := findServerPrivateNetByID(srv, r.network.ID)
			if !ok {
				return false, fmt.Errorf("%s: server %v: no network with id: %d", op, route.TargetNode, r.network.ID)
			}
			ip := privNet.IP

			if !_route.Gateway.Equal(ip) {
				action, _, err := r.client.Network.DeleteRoute(context.Background(), r.network, hcloud.NetworkDeleteRouteOpts{
					Route: _route,
				})
				if err != nil {
					return false, fmt.Errorf("%s: %w", op, err)
				}

				if err := hcops.WatchAction(ctx, &r.client.Action, action); err != nil {
					return false, fmt.Errorf("%s: %w", op, err)
				}
			}

			return true, nil
		}
	}
	return false, nil
}

func findServerPrivateNetByID(srv *hcloud.Server, id int64) (hcloud.ServerPrivateNet, bool) {
	for _, n := range srv.PrivateNet {
		if n.Network.ID == id {
			return n, true
		}
	}
	return hcloud.ServerPrivateNet{}, false
}
