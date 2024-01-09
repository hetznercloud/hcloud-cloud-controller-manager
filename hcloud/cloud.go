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
	"strings"

	hrobot "github.com/syself/hrobot-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/robot"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
)

const (
	providerName = "hcloud"
)

// providerVersion is set by the build process using -ldflags -X.
var providerVersion = "unknown"

type cloud struct {
	client      *hcloud.Client
	robotClient robot.Client
	cfg         config.HCCMConfiguration
	recorder    record.EventRecorder
	networkID   int64
}

func newCloud(_ io.Reader) (cloudprovider.Interface, error) {
	const op = "hcloud/newCloud"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	cfg, err := config.Read()
	if err != nil {
		return nil, err
	}
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}

	opts := []hcloud.ClientOption{
		hcloud.WithToken(cfg.HCloudClient.Token),
		hcloud.WithApplication("hcloud-cloud-controller", providerVersion),
	}

	// start metrics server if enabled (enabled by default)
	if cfg.Metrics.Enabled {
		go metrics.Serve(cfg.Metrics.Address)
		opts = append(opts, hcloud.WithInstrumentation(metrics.GetRegistry()))
	}

	if cfg.HCloudClient.Debug {
		opts = append(opts, hcloud.WithDebugWriter(os.Stderr))
	}
	if cfg.HCloudClient.Endpoint != "" {
		opts = append(opts, hcloud.WithEndpoint(cfg.HCloudClient.Endpoint))
	}
	client := hcloud.NewClient(opts...)
	metadataClient := metadata.NewClient()

	var robotClient robot.Client
	if cfg.Robot.Enabled {
		c := hrobot.NewBasicAuthClient(cfg.Robot.User, cfg.Robot.Password)

		robotClient = robot.NewRateLimitedClient(
			cfg.Robot.RateLimitWaitTime,
			robot.NewCachedClient(cfg.Robot.CacheTimeout, c),
		)
	}

	var networkID int64
	if cfg.Network.NameOrID != "" {
		n, _, err := client.Network.Get(context.Background(), cfg.Network.NameOrID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if n == nil {
			return nil, fmt.Errorf("%s: Network %s not found", op, cfg.Network.NameOrID)
		}
		networkID = n.ID

		if !cfg.Network.DisableAttachedCheck {
			attached, err := serverIsAttachedToNetwork(metadataClient, networkID)
			if err != nil {
				return nil, fmt.Errorf("%s: checking if server is in Network not possible: %w", op, err)
			}
			if !attached {
				return nil, fmt.Errorf("%s: This node is not attached to Network %s", op, cfg.Network.NameOrID)
			}
		}
	}

	// Validate that the provided token works, and we have network connectivity to the Hetzner Cloud API
	_, _, err = client.Server.List(context.Background(), hcloud.ServerListOpts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	klog.Infof("Hetzner Cloud k8s cloud controller %s started\n", providerVersion)

	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "hcloud-cloud-controller-manager"})

	return &cloud{
		client:      client,
		robotClient: robotClient,
		cfg:         cfg,
		networkID:   networkID,
		recorder:    recorder,
	}, nil
}

func (c *cloud) Initialize(_ cloudprovider.ControllerClientBuilder, _ <-chan struct{}) {
}

func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	// Replaced by InstancesV2
	return nil, false
}

func (c *cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return newInstances(c.client, c.robotClient, c.recorder, c.cfg.Instance.AddressFamily, c.networkID), true
}

func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	// Replaced by InstancesV2
	return nil, false
}

func (c *cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	if !c.cfg.LoadBalancer.Enabled {
		return nil, false
	}

	lbOps := &hcops.LoadBalancerOps{
		LBClient:      &c.client.LoadBalancer,
		RobotClient:   c.robotClient,
		CertOps:       &hcops.CertificateOps{CertClient: &c.client.Certificate},
		ActionClient:  &c.client.Action,
		NetworkClient: &c.client.Network,
		NetworkID:     c.networkID,
		Cfg:           c.cfg,
		Recorder:      c.recorder,
	}

	return newLoadBalancers(lbOps, c.cfg.LoadBalancer.DisablePrivateIngress, c.cfg.LoadBalancer.DisableIPv6), true
}

func (c *cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

func (c *cloud) Routes() (cloudprovider.Routes, bool) {
	if !c.cfg.Route.Enabled {
		// If no network is configured, disable the routes controller
		return nil, false
	}

	r, err := newRoutes(c.client, c.networkID)
	if err != nil {
		klog.ErrorS(err, "create routes provider", "networkID", c.networkID)
		return nil, false
	}
	return r, true
}

func (c *cloud) ProviderName() string {
	return providerName
}

func (c *cloud) HasClusterID() bool {
	return false
}

// serverIsAttachedToNetwork checks if the server where the master is running on is attached to the configured private network
// We use this measurement to protect users against some parts of misconfiguration, like configuring a master in a not attached
// network.
func serverIsAttachedToNetwork(metadataClient *metadata.Client, networkID int64) (bool, error) {
	const op = "serverIsAttachedToNetwork"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	serverPrivateNetworks, err := metadataClient.PrivateNetworks()
	if err != nil {
		return false, fmt.Errorf("%s: %s", op, err)
	}
	return strings.Contains(serverPrivateNetworks, fmt.Sprintf("network_id: %d\n", networkID)), nil
}

func init() {
	cloudprovider.RegisterCloudProvider(providerName, newCloud)
}
