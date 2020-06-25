/*
Copyright 2018 Hetzner Cloud GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hcloud

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-go/hcloud"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

const (
	hcloudTokenENVVar    = "HCLOUD_TOKEN"
	hcloudEndpointENVVar = "HCLOUD_ENDPOINT"
	hcloudNetworkENVVar  = "HCLOUD_NETWORK"
	hcloudDebugENVVar    = "HCLOUD_DEBUG"
	nodeNameENVVar       = "NODE_NAME"
	providerName         = "hcloud"
	providerVersion      = "v1.6.1"
)

type cloud struct {
	client       *hcloud.Client
	instances    cloudprovider.Instances
	zones        cloudprovider.Zones
	routes       cloudprovider.Routes
	loadBalancer cloudprovider.LoadBalancer
	networkID    int
}

func newCloud(config io.Reader) (cloudprovider.Interface, error) {
	const op = "hcloud/newCloud"

	token := os.Getenv(hcloudTokenENVVar)
	if token == "" {
		return nil, fmt.Errorf("environment variable %q is required", hcloudTokenENVVar)
	}
	if len(token) != 64 {
		return nil, fmt.Errorf("entered token is invalid (must be exactly 64 characters long)")
	}
	nodeName := os.Getenv(nodeNameENVVar)
	if nodeName == "" {
		return nil, fmt.Errorf("environment variable %q is required", nodeNameENVVar)
	}

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-cloud-controller", providerVersion),
	}
	if os.Getenv(hcloudDebugENVVar) == "true" {
		opts = append(opts, hcloud.WithDebugWriter(os.Stderr))
	}
	if endpoint := os.Getenv(hcloudEndpointENVVar); endpoint != "" {
		opts = append(opts, hcloud.WithEndpoint(endpoint))
	}
	client := hcloud.NewClient(opts...)

	var networkID int
	if v, ok := os.LookupEnv(hcloudNetworkENVVar); ok {
		n, _, err := client.Network.Get(context.Background(), v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		networkID = n.ID
	}
	if networkID == 0 {
		klog.InfoS("%s: %s empty", op, hcloudNetworkENVVar)
	}

	_, _, err := client.Server.List(context.Background(), hcloud.ServerListOpts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	fmt.Printf("Hetzner Cloud k8s cloud controller %s started\n", providerVersion)
	lbOps := &hcops.LoadBalancerOps{
		LBClient:      &client.LoadBalancer,
		ActionClient:  &client.Action,
		NetworkClient: &client.Network,
		NetworkID:     networkID,
	}
	return &cloud{
		client:       client,
		zones:        newZones(client, nodeName),
		instances:    newInstances(client),
		loadBalancer: newLoadBalancers(lbOps, &client.LoadBalancer, &client.Action),
		routes:       nil,
		networkID:    networkID,
	}, nil
}

func (c *cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
}

func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	return c.instances, true
}

func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	return c.zones, true
}

func (c *cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return c.loadBalancer, true
}

func (c *cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

func (c *cloud) Routes() (cloudprovider.Routes, bool) {
	if c.networkID > 0 {
		r, err := newRoutes(c.client, c.networkID)
		if err != nil {
			klog.ErrorS(err, "create routes provider", "networkID", c.networkID)
			return nil, false
		}
		return r, true
	}
	return nil, false // If no network is configured, disable the routes part
}

func (c *cloud) ProviderName() string {
	return providerName
}

func (c *cloud) ScrubDNS(nameservers, searches []string) (nsOut, srchOut []string) {
	return nil, nil
}

func (c *cloud) HasClusterID() bool {
	return false
}

func init() {
	cloudprovider.RegisterCloudProvider(providerName, func(config io.Reader) (cloudprovider.Interface, error) {
		return newCloud(config)
	})
}
