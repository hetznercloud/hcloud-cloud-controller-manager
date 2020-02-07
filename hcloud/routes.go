package hcloud

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/cloudprovider"
)

type routes struct {
	client      *hcloud.Client
	network     *hcloud.Network
	serverNames map[string]string
	serverIPs   map[string]net.IP
}

func newRoutes(client *hcloud.Client, network string) (*routes, error) {
	networkObj, _, err := client.Network.Get(context.Background(), network)
	if err != nil {
		return nil, err
	}

	return &routes{client, networkObj, make(map[string]string), make(map[string]net.IP)}, nil
}

func (r *routes) reloadNetwork(ctx context.Context) error {
	networkObj, _, err := r.client.Network.GetByID(ctx, r.network.ID)
	if err != nil {
		return err
	}
	r.network = networkObj
	return nil
}

func (r *routes) loadServers(ctx context.Context) error {
	serversRaw, err := r.client.Server.All(ctx)
	if err != nil {
		return err
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
	err := r.reloadNetwork(ctx)
	if err != nil {
		return []*cloudprovider.Route{}, err
	}
	err = r.loadServers(ctx)
	if err != nil {
		return []*cloudprovider.Route{}, err
	}
	var routes []*cloudprovider.Route
	for _, route := range r.network.Routes {
		r, err := r.hcloudRouteToRoute(route)
		if err != nil {
			return nil, err
		}
		routes = append(routes, r)
	}
	return routes, nil
}

// CreateRoute creates the described managed route
// route.Name will be ignored, although the cloud-provider may use nameHint
// to create a more user-meaningful name.
func (r *routes) CreateRoute(ctx context.Context, clusterName string, nameHint string, route *cloudprovider.Route) error {
	err := r.loadServers(ctx)
	if err != nil {
		return err
	}

	ip, ok := r.serverIPs[string(route.TargetNode)]
	if !ok {
		return fmt.Errorf("server %v not found", route.TargetNode)
	}
	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return err
	}
	doesRouteAlreadyExist, err := r.checkIfRouteAlreadyExists(ctx, route)
	if err != nil {
		return err
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
				time.Sleep(time.Second * 5)
				return r.CreateRoute(ctx, clusterName, nameHint, route)
			}
			return err
		}
		err = r.watchAction(ctx, action)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteRoute deletes the specified managed route
// Route should be as returned by ListRoutes
func (r *routes) DeleteRoute(ctx context.Context, clusterName string, route *cloudprovider.Route) error {
	err := r.loadServers(ctx)
	if err != nil {
		return err
	}
	ip, ok := r.serverIPs[string(route.TargetNode)]
	if !ok {
		return fmt.Errorf("server %v not found", route.TargetNode)
	}
	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return err
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
			time.Sleep(time.Second * 5)
			return r.DeleteRoute(ctx, clusterName, route)
		}
		return err
	}
	err = r.watchAction(ctx, action)
	if err != nil {
		return err
	}
	return nil
}

func (r *routes) hcloudRouteToRoute(route hcloud.NetworkRoute) (*cloudprovider.Route, error) {
	nodeName, ok := r.serverNames[route.Gateway.String()]
	if !ok {
		return nil, fmt.Errorf("server with IP %v not found", route.Gateway)
	}
	return &cloudprovider.Route{
		DestinationCIDR: route.Destination.String(),
		Name:            fmt.Sprintf("%s-%s", route.Gateway.String(), route.Destination.String()),
		TargetNode:      types.NodeName(nodeName),
	}, nil
}

func (r *routes) watchAction(ctx context.Context, action *hcloud.Action) error {
	_, errCh := r.client.Action.WatchProgress(ctx, action)
	if err := <-errCh; err != nil {
		return err
	}
	return nil
}

func (r *routes) checkIfRouteAlreadyExists(ctx context.Context, route *cloudprovider.Route) (bool, error) {
	err := r.reloadNetwork(ctx)
	if err != nil {
		return false, err
	}

	for _, _route := range r.network.Routes {
		if _route.Destination.String() == route.DestinationCIDR {
			ip, ok := r.serverIPs[string(route.TargetNode)]
			if !ok {
				return false, fmt.Errorf("server with name %v not found", string(route.TargetNode))
			}
			if !_route.Gateway.Equal(ip) {
				action, _, err := r.client.Network.DeleteRoute(context.Background(), r.network, hcloud.NetworkDeleteRouteOpts{
					Route: _route,
				})
				if err != nil {
					return false, err
				}
				if r.watchAction(ctx, action) != nil {
					return false, err
				}
			}
			return true, nil
		}
	}
	return false, nil
}
