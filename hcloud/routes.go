package hcloud

import (
	"context"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/cloudprovider"
	"net"
	"time"
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
	serversRaw, err := client.Server.All(context.Background())
	if err != nil {
		return nil, err
	}
	serverNamesMap := make(map[string]string)
	serverIPsMap := make(map[string]net.IP)
	for _, server := range serversRaw {
		for _, privateNet := range server.PrivateNet {
			if privateNet.Network.ID == networkObj.ID {
				serverNamesMap[privateNet.IP.String()] = server.Name
				serverIPsMap[server.Name] = privateNet.IP
				break
			}
		}

	}
	return &routes{client, networkObj, serverNamesMap, serverIPsMap}, nil
}

// ListRoutes lists all managed routes that belong to the specified clusterName

// CreateRoute creates the described managed route
// route.Name will be ignored, although the cloud-provider may use nameHint
// to create a more user-meaningful name.

// DeleteRoute deletes the specified managed route
// Route should be as returned by ListRoutes

func (r *routes) ListRoutes(ctx context.Context, clusterName string) ([]*cloudprovider.Route, error) {
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

func (r *routes) CreateRoute(ctx context.Context, clusterName string, nameHint string, route *cloudprovider.Route) error {
	ip, ok := r.serverIPs[string(route.TargetNode)]
	if !ok {
		return fmt.Errorf("server %v not found", route.TargetNode)
	}
	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return err
	}
	opts := hcloud.NetworkAddRouteOpts{
		Route: hcloud.NetworkRoute{
			Destination: cidr,
			Gateway:     ip,
		},
	}
	action, _, err := r.client.Network.AddRoute(ctx, r.network, opts)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeLocked) || hcloud.IsError(err, hcloud.ErrorCodeConflict) {
			time.Sleep(time.Second * 2)
			return r.CreateRoute(ctx, clusterName, nameHint, route)
		}
		return err
	}
	err = watchAction(ctx, r.client, action)
	return err
}

func (r *routes) DeleteRoute(ctx context.Context, clusterName string, route *cloudprovider.Route) error {
	ip, ok := r.serverIPs[string(route.TargetNode)]
	if !ok {
		return fmt.Errorf("server %v not found", route.TargetNode)
	}
	_, cidr, err := net.ParseCIDR(route.DestinationCIDR)
	if err != nil {
		return err
	}
	opts := hcloud.NetworkRemoveRouteOpts{
		Route: hcloud.NetworkRoute{
			Destination: cidr,
			Gateway:     ip,
		},
	}
	action, _, err := r.client.Network.RemoveRoute(ctx, r.network, opts)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeLocked) || hcloud.IsError(err, hcloud.ErrorCodeConflict) {
			time.Sleep(time.Second * 2)
			return r.DeleteRoute(ctx, clusterName, route)
		}
		return err
	}
	err = watchAction(ctx, r.client, action)
	return err
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
		Blackhole:       false,
	}, nil
}

func watchAction(ctx context.Context, client *hcloud.Client, action *hcloud.Action) error {
	_, errCh := client.Action.WatchProgress(ctx, action)
	if err := <-errCh; err != nil {
		return err
	}
	return nil
}
