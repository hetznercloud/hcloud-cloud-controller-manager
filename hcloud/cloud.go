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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

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
	// Disable the "master/server is attached to the network" check against the metadata service.
	hcloudNetworkDisableAttachedCheckENVVar  = "HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK"
	hcloudLoadBalancersEnabledENVVar         = "HCLOUD_LOAD_BALANCERS_ENABLED"
	hcloudLoadBalancersLocation              = "HCLOUD_LOAD_BALANCERS_LOCATION"
	hcloudLoadBalancersNetworkZone           = "HCLOUD_LOAD_BALANCERS_NETWORK_ZONE"
	hcloudLoadBalancersDisablePrivateIngress = "HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS"
	hcloudLoadBalancersUsePrivateIP          = "HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP"
	nodeNameENVVar                           = "NODE_NAME"
	providerName                             = "hcloud"
	providerVersion                          = "v1.9.1"
)

type cloud struct {
	client       *hcloud.Client
	instances    *instances
	zones        *zones
	routes       *routes
	loadBalancer *loadBalancers
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
		if n == nil {
			return nil, fmt.Errorf("%s: Network %s not found", op, v)
		}
		networkID = n.ID
		if networkDisableAttachedCheck := os.Getenv(hcloudNetworkDisableAttachedCheckENVVar); networkDisableAttachedCheck != "true" {
			e, err := serverIsAttachedToNetwork(networkID)
			if err != nil {
				return nil, fmt.Errorf("%s: checking if server is in Network not possible: %w", op, err)
			}
			if !e {
				return nil, fmt.Errorf("%s: This node is not attached to Network %s", op, v)
			}
		}
	}
	if networkID == 0 {
		klog.Infof("%s: %s empty", op, hcloudNetworkENVVar)
	}

	_, _, err := client.Server.List(context.Background(), hcloud.ServerListOpts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	lbOpsDefaults, lbDisablePrivateIngress, err := loadBalancerDefaultsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	fmt.Printf("Hetzner Cloud k8s cloud controller %s started\n", providerVersion)

	lbOps := &hcops.LoadBalancerOps{
		LBClient:      &client.LoadBalancer,
		CertOps:       &hcops.CertificateOps{CertClient: &client.Certificate},
		ActionClient:  &client.Action,
		NetworkClient: &client.Network,
		NetworkID:     networkID,
		Defaults:      lbOpsDefaults,
	}

	loadBalancers := newLoadBalancers(lbOps, &client.Action, lbDisablePrivateIngress)
	if os.Getenv(hcloudLoadBalancersEnabledENVVar) == "false" {
		loadBalancers = nil
	}
	return &cloud{
		client:       client,
		zones:        newZones(client, nodeName),
		instances:    newInstances(client),
		loadBalancer: loadBalancers,
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
	if c.loadBalancer == nil {
		return nil, false
	}
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

func loadBalancerDefaultsFromEnv() (hcops.LoadBalancerDefaults, bool, error) {
	defaults := hcops.LoadBalancerDefaults{
		Location:    os.Getenv(hcloudLoadBalancersLocation),
		NetworkZone: os.Getenv(hcloudLoadBalancersNetworkZone),
	}

	if defaults.Location != "" && defaults.NetworkZone != "" {
		return defaults, false, errors.New(
			"HCLOUD_LOAD_BALANCERS_LOCATION/HCLOUD_LOAD_BALANCERS_NETWORK_ZONE: Only one of these can be set")
	}

	disablePrivateIngress, err := getEnvBool(hcloudLoadBalancersDisablePrivateIngress)
	if err != nil {
		return defaults, false, err
	}

	defaults.UsePrivateIP, err = getEnvBool(hcloudLoadBalancersUsePrivateIP)
	if err != nil {
		return defaults, false, err
	}

	return defaults, disablePrivateIngress, nil
}

// serverIsAttachedToNetwork checks if the server where the master is running on is attached to the configured private network
// We use this measurement to protect users against some parts of misconfiguration, like configuring a master in a not attached
// network.
func serverIsAttachedToNetwork(networkId int) (bool, error) {
	const op = "serverIsAttachedToNetwork"
	resp, err := http.Get("http://169.254.169.254/hetzner/v1/metadata/private-networks")
	if err != nil {
		return false, fmt.Errorf("%s: %s", op, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("%s: %s", op, err)
	}
	return strings.Contains(string(body), fmt.Sprintf("network_id: %d", networkId)), nil
}

// getEnvBool returns the boolean parsed from the environment variable with the given key and a potential error
// parsing the var. Returns false if the env var is unset.
func getEnvBool(key string) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return false, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %v", key, err)
	}

	return b, nil
}

func init() {
	cloudprovider.RegisterCloudProvider(providerName, func(config io.Reader) (cloudprovider.Interface, error) {
		return newCloud(config)
	})
}
