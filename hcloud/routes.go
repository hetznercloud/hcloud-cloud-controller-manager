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
	serverNames map[string]string
	serverIPs   map[string]net.IP
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

	return &routes{client, networkObj, make(map[string]string), make(map[string]net.IP)}, nil
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

func (r *routes) loadServers(ctx context.Context) error {
	const op = "hcloud/loadServers"

	serversRaw, err := r.client.Server.All(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	for _, server := range serversRaw {
		for _, privateNet := range server.PrivateNet {
			if privateNet.Network.ID == r.network.ID {
				r.serverNames[privateNet.IP.String()] = server.Name
				r.serverIPs[server.Name] = privateNet.IP
				break
			}
		}
	}
	return nil
}

// ListRoutes lists all managed routes that belong to the specified clusterName
func (r *routes) ListRoutes(ctx context.Context, clusterName string) ([]*cloudprovider.Route, error) {
	const op = "hcloud/ListRoutes"
	var routes []*cloudprovider.Route

	if err := r.reloadNetwork(ctx); err != nil {
		return routes, fmt.Errorf("%s: %w", op, err)
	}

	if err := r.loadServers(ctx); err != nil {
		return routes, fmt.Errorf("%s: %w", op, err)
	}

	for _, route := range r.network.Routes {
		r, err := r.hcloudRouteToRoute(route)
		if err != nil {
			return routes, fmt.Errorf("%s: %w", op, err)
		}
		routes = append(routes, r)
	}
	return routes, nil
}

// CreateRoute creates the described managed route
// route.Name will be ignored, although the cloud-provider may use nameHint
// to create a more user-meaningful name.
func (r *routes) CreateRoute(ctx context.Context, clusterName string, nameHint string, route *cloudprovider.Route) error {
	const op = "hcloud/CreateRoute"

	if err := r.loadServers(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	ip, ok := r.serverIPs[string(route.TargetNode)]
	if !ok {
		return fmt.Errorf("%s: server %v: not found", op, route.TargetNode)
	}

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

	if err := r.loadServers(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	ip, ok := r.serverIPs[string(route.TargetNode)]
	if !ok {
		return fmt.Errorf("%s: server %v: not found", op, route.TargetNode)
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

	nodeName, ok := r.serverNames[route.Gateway.String()]
	if !ok {
		return nil, fmt.Errorf("%s: server with IP %v: not found", op, route.Gateway)
	}

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
			ip, ok := r.serverIPs[string(route.TargetNode)]
			if !ok {
				return false, fmt.Errorf("%s: server %v: not found", op, string(route.TargetNode))
			}

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
