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
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
	"github.com/syself/hetzner-cloud-controller-manager/internal/hcops"
	"github.com/syself/hetzner-cloud-controller-manager/internal/metrics"
	robotclient "github.com/syself/hetzner-cloud-controller-manager/internal/robot/client"
	"github.com/syself/hetzner-cloud-controller-manager/internal/robot/client/cache"
	"github.com/syself/hetzner-cloud-controller-manager/internal/util"
	hrobot "github.com/syself/hrobot-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	hcloudTokenENVVar    = "HCLOUD_TOKEN"
	hcloudEndpointENVVar = "HCLOUD_ENDPOINT"
	hcloudNetworkENVVar  = "HCLOUD_NETWORK"
	hcloudDebugENVVar    = "HCLOUD_DEBUG"
	robotDebugENVVar     = "ROBOT_DEBUG"

	robotUserNameENVVar = "ROBOT_USER_NAME"
	robotPasswordENVVar = "ROBOT_PASSWORD"

	// Only as reference - is used in hcops package.
	// Default is 5 minutes.
	RateLimitWaitTimeRobot = "RATE_LIMIT_WAIT_TIME_ROBOT"

	// default is 5 minutes.
	CacheTimeout = "CACHE_TIMEOUT"

	// Disable the "master/server is attached to the network" check against the metadata service.
	hcloudNetworkDisableAttachedCheckENVVar  = "HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK"
	hcloudNetworkRoutesEnabledENVVar         = "HCLOUD_NETWORK_ROUTES_ENABLED"
	hcloudInstancesAddressFamily             = "HCLOUD_INSTANCES_ADDRESS_FAMILY"
	hcloudLoadBalancersEnabledENVVar         = "HCLOUD_LOAD_BALANCERS_ENABLED"
	hcloudLoadBalancersLocation              = "HCLOUD_LOAD_BALANCERS_LOCATION"
	hcloudLoadBalancersNetworkZone           = "HCLOUD_LOAD_BALANCERS_NETWORK_ZONE"
	hcloudLoadBalancersDisablePrivateIngress = "HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS"
	hcloudLoadBalancersUsePrivateIP          = "HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP"
	hcloudLoadBalancersDisableIPv6           = "HCLOUD_LOAD_BALANCERS_DISABLE_IPV6"
	hcloudMetricsEnabledENVVar               = "HCLOUD_METRICS_ENABLED"
	hcloudMetricsAddress                     = ":8233"
	nodeNameENVVar                           = "NODE_NAME"
	providerName                             = "hcloud"
	hostNamePrefixRobot                      = "bm-"
)

var errMissingRobotCredentials = errors.New("missing robot credentials - cannot connect to robot API")

// providerVersion is set by the build process using -ldflags -X.
var providerVersion = "unknown"

type cloud struct {
	client       *hcloud.Client
	robotClient  robotclient.Client
	instances    *instances
	routes       *routes
	loadBalancer *loadBalancers
	networkID    int64
}

type LoggingTransport struct {
	roundTripper http.RoundTripper
}

var replaceHex = regexp.MustCompile(`0x[0123456789abcdef]+`)

func (lt *LoggingTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	stack := replaceHex.ReplaceAllString(string(debug.Stack()), "0xX")
	stack = strings.ReplaceAll(stack, "\n", "\\n")

	resp, err = lt.roundTripper.RoundTrip(req)
	if err != nil {
		klog.InfoS("hetzner robot API. Error.", "err", err, "method", req.Method, "url", req.URL, "stack", stack)
		return resp, err
	}
	klog.V(1).InfoS("hetzner robot API called.", "statusCode", resp.StatusCode, "method", req.Method, "url", req.URL, "stack", stack)
	return resp, nil
}

func newCloud(_ io.Reader) (cloudprovider.Interface, error) {
	const op = "hcloud/newCloud"
	metrics.OperationCalled.WithLabelValues(op).Inc()

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

	// start metrics server if enabled (enabled by default)
	if os.Getenv(hcloudMetricsEnabledENVVar) != "false" {
		go metrics.Serve(hcloudMetricsAddress)

		opts = append(opts, hcloud.WithInstrumentation(metrics.GetRegistry()))
	}

	if os.Getenv(hcloudDebugENVVar) == "true" {
		opts = append(opts, hcloud.WithDebugWriter(os.Stderr))
	}
	if endpoint := os.Getenv(hcloudEndpointENVVar); endpoint != "" {
		opts = append(opts, hcloud.WithEndpoint(endpoint))
	}
	client := hcloud.NewClient(opts...)
	metadataClient := metadata.NewClient()

	robotUserName := os.Getenv(robotUserNameENVVar)
	robotPassword := os.Getenv(robotPasswordENVVar)

	cacheTimeout, err := util.GetEnvDuration(CacheTimeout)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if cacheTimeout == 0 {
		cacheTimeout = 5 * time.Minute
	}

	var robotClient robotclient.Client
	if robotUserName != "" && robotPassword != "" {
		var c hrobot.RobotClient
		if os.Getenv(robotDebugENVVar) == "true" {
			client := &http.Client{
				Transport: &LoggingTransport{
					roundTripper: http.DefaultTransport,
				},
			}
			c = hrobot.NewBasicAuthClientWithCustomHttpClient(robotUserName, robotPassword, client)
			klog.Info("Enabled robot API debugging")
		} else {
			c = hrobot.NewBasicAuthClient(robotUserName, robotPassword)
			klog.Infof("Not enabling robot API debugging. Set env var %s=true to enable it.", robotDebugENVVar)
		}
		robotClient = cache.NewClient(c, cacheTimeout)
	} else {
		klog.Infof("Hetzner robot is not support because of insufficient credentials. Robot user name specified: %v. Robot password specified: %v", robotUserName != "", robotPassword != "")
	}

	var networkID int64
	if v, ok := os.LookupEnv(hcloudNetworkENVVar); ok {
		n, _, err := client.Network.Get(context.Background(), v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if n == nil {
			return nil, fmt.Errorf("%s: Network %s not found", op, v)
		}
		networkID = n.ID

		networkDisableAttachedCheck, err := getEnvBool(hcloudNetworkDisableAttachedCheckENVVar)
		if err != nil {
			return nil, fmt.Errorf("%s: checking if server is in Network not possible: %w", op, err)
		}
		if !networkDisableAttachedCheck {
			e, err := serverIsAttachedToNetwork(metadataClient, networkID)
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

	// Validate that the provided token works, and we have network connectivity to the Hetzner Cloud API
	_, _, err = client.Server.List(context.Background(), hcloud.ServerListOpts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	lbOpsDefaults, lbDisablePrivateIngress, lbDisableIPv6, err := loadBalancerDefaultsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	klog.Infof("Hetzner Cloud k8s cloud controller %s started\n", providerVersion)

	lbOpsDefaults.DisableIPv6 = lbDisableIPv6

	eventBroadcaster := record.NewBroadcaster()
	lbRecorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "hetzner-ccm-loadbalancer"})

	lbOps := &hcops.LoadBalancerOps{
		LBClient:      &client.LoadBalancer,
		CertOps:       &hcops.CertificateOps{CertClient: &client.Certificate},
		ActionClient:  &client.Action,
		NetworkClient: &client.Network,
		RobotClient:   robotClient,
		NetworkID:     networkID,
		Recorder:      lbRecorder,
		Defaults:      lbOpsDefaults,
	}

	loadBalancers := newLoadBalancers(lbOps, &client.Action, lbDisablePrivateIngress, lbDisableIPv6)
	if os.Getenv(hcloudLoadBalancersEnabledENVVar) == "false" {
		loadBalancers = nil
	}

	instancesAddressFamily, err := addressFamilyFromEnv()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &cloud{
		client:       client,
		robotClient:  robotClient,
		instances:    newInstances(client, robotClient, instancesAddressFamily, networkID),
		loadBalancer: loadBalancers,
		routes:       nil,
		networkID:    networkID,
	}, nil
}

func (c *cloud) Initialize(_ cloudprovider.ControllerClientBuilder, _ <-chan struct{}) {
}

func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	// Replaced by InstancesV2
	return nil, false
}

func (c *cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return c.instances, true
}

func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	// Replaced by InstancesV2
	return nil, false
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
	if c.networkID > 0 && os.Getenv(hcloudNetworkRoutesEnabledENVVar) != "false" {
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

func (c *cloud) HasClusterID() bool {
	return false
}

func loadBalancerDefaultsFromEnv() (hcops.LoadBalancerDefaults, bool, bool, error) {
	defaults := hcops.LoadBalancerDefaults{
		Location:    os.Getenv(hcloudLoadBalancersLocation),
		NetworkZone: os.Getenv(hcloudLoadBalancersNetworkZone),
	}

	if defaults.Location != "" && defaults.NetworkZone != "" {
		return defaults, false, false, errors.New(
			"HCLOUD_LOAD_BALANCERS_LOCATION/HCLOUD_LOAD_BALANCERS_NETWORK_ZONE: Only one of these can be set")
	}

	disablePrivateIngress, err := getEnvBool(hcloudLoadBalancersDisablePrivateIngress)
	if err != nil {
		return defaults, false, false, err
	}

	disableIPv6, err := getEnvBool(hcloudLoadBalancersDisableIPv6)
	if err != nil {
		return defaults, false, false, err
	}

	defaults.UsePrivateIP, err = getEnvBool(hcloudLoadBalancersUsePrivateIP)
	if err != nil {
		return defaults, false, false, err
	}

	return defaults, disablePrivateIngress, disableIPv6, nil
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

// addressFamilyFromEnv returns the address family for the instance address from the environment
// variable. Returns AddressFamilyIPv4 if unset.
func addressFamilyFromEnv() (addressFamily, error) {
	family, ok := os.LookupEnv(hcloudInstancesAddressFamily)
	if !ok {
		return AddressFamilyIPv4, nil
	}

	switch strings.ToLower(family) {
	case "ipv6":
		return AddressFamilyIPv6, nil
	case "ipv4":
		return AddressFamilyIPv4, nil
	case "dualstack":
		return AddressFamilyDualStack, nil
	default:
		return -1, fmt.Errorf(
			"%v: Invalid value, expected one of: ipv4,ipv6,dualstack", hcloudInstancesAddressFamily)
	}
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
	cloudprovider.RegisterCloudProvider(providerName, newCloud)
}
