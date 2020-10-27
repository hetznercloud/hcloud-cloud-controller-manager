package hcloud

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

type routes struct {
	client      *hcloud.Client
	network     *hcloud.Network
	serverCache *hcops.AllServersCache
}

func newRoutes(client *hcloud.Client, networkID int) (*routes, error) {
	const op = "hcloud/newRoutes"

	networkObj, _, err := client.Network.GetByID(context.Background(), networkID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if networkObj == nil {
		return nil, fmt.Errorf("network not found: %d", networkID)
	}

	return &routes{
		client:      client,
		network:     networkObj,
		serverCache: &hcops.AllServersCache{LoadFunc: client.Server.All},
	}, nil
}

func (r *routes) reloadNetwork(ctx context.Context) error {
	const op = "hcloud/reloadNetwork"

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

// ListRoutes lists all managed routes that belong to the specified clusterName
func (r *routes) ListRoutes(ctx context.Context, clusterName string) ([]*cloudprovider.Route, error) {
	const op = "hcloud/ListRoutes"

	if err := r.reloadNetwork(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	routes := make([]*cloudprovider.Route, len(r.network.Routes))
	for i, route := range r.network.Routes {
		r, err := r.hcloudRouteToRoute(route)
		if err != nil {
			return routes, fmt.Errorf("%s: %w", op, err)
		}
		routes[i] = r
	}
	return routes, nil
}

// CreateRoute creates the described managed route
// route.Name will be ignored, although the cloud-provider may use nameHint
// to create a more user-meaningful name.
func (r *routes) CreateRoute(ctx context.Context, clusterName string, nameHint string, route *cloudprovider.Route) error {
	const op = "hcloud/CreateRoute"

	srv, err := r.serverCache.ByName(string(route.TargetNode))
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	privNet, ok := findServerPrivateNetByID(srv, r.network.ID)
	if !ok {
		return fmt.Errorf("%s: server %v: no network with id: %d", op, route.TargetNode, r.network.ID)
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
// Route should be as returned by ListRoutes
func (r *routes) DeleteRoute(ctx context.Context, clusterName string, route *cloudprovider.Route) error {
	const op = "hcloud/DeleteRoute"

	srv, err := r.serverCache.ByName(string(route.TargetNode))
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	privNet, ok := findServerPrivateNetByID(srv, r.network.ID)
	if !ok {
		return fmt.Errorf("%s: server %v: no network with id: %d", op, route.TargetNode, r.network.ID)
	}
	ip := privNet.IP

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
		if hcloud.IsError(err, hcloud.ErrorCodeLocked) || hcloud.IsError(err, hcloud.ErrorCodeConflict) {
			retryDelay := time.Second * 5
			klog.InfoS("retry due to conflict or lock",
				"op", op, "delay", fmt.Sprintf("%v", retryDelay), "err", fmt.Sprintf("%v", err))
			time.Sleep(retryDelay)

			return r.DeleteRoute(ctx, clusterName, route)
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

	srv, err := r.serverCache.ByPrivateIP(route.Gateway)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", op, err)
	}
	nodeName := srv.Name

	return &cloudprovider.Route{
		DestinationCIDR: route.Destination.String(),
		Name:            fmt.Sprintf("%s-%s", route.Gateway.String(), route.Destination.String()),
		TargetNode:      types.NodeName(nodeName),
	}, nil
}

func (r *routes) checkIfRouteAlreadyExists(ctx context.Context, route *cloudprovider.Route) (bool, error) {
	const op = "hcloud/checkIfRouteAlreadyExists"

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

func findServerPrivateNetByID(srv *hcloud.Server, id int) (hcloud.ServerPrivateNet, bool) {
	for _, n := range srv.PrivateNet {
		if n.Network.ID == id {
			return n, true
		}
	}
	return hcloud.ServerPrivateNet{}, false
}
