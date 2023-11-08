package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const (
	hcloudToken    = "HCLOUD_TOKEN"
	hcloudEndpoint = "HCLOUD_ENDPOINT"
	hcloudNetwork  = "HCLOUD_NETWORK"
	hcloudDebug    = "HCLOUD_DEBUG"

	hcloudInstancesAddressFamily = "HCLOUD_INSTANCES_ADDRESS_FAMILY"

	// Disable the "master/server is attached to the network" check against the metadata service.
	hcloudNetworkDisableAttachedCheck = "HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK"
	hcloudNetworkRoutesEnabled        = "HCLOUD_NETWORK_ROUTES_ENABLED"

	hcloudLoadBalancersEnabled               = "HCLOUD_LOAD_BALANCERS_ENABLED"
	hcloudLoadBalancersLocation              = "HCLOUD_LOAD_BALANCERS_LOCATION"
	hcloudLoadBalancersNetworkZone           = "HCLOUD_LOAD_BALANCERS_NETWORK_ZONE"
	hcloudLoadBalancersDisablePrivateIngress = "HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS"
	hcloudLoadBalancersUsePrivateIP          = "HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP"
	hcloudLoadBalancersDisableIPv6           = "HCLOUD_LOAD_BALANCERS_DISABLE_IPV6"

	hcloudMetricsEnabled = "HCLOUD_METRICS_ENABLED"
	hcloudMetricsAddress = ":8233"
)

type HCloudClientConfiguration struct {
	Token    string
	Endpoint string
	Debug    bool
}

type MetricsConfiguration struct {
	Enabled bool
	Address string
}

type AddressFamily string

const (
	AddressFamilyDualStack AddressFamily = "dualstack"
	AddressFamilyIPv6      AddressFamily = "ipv6"
	AddressFamilyIPv4      AddressFamily = "ipv4"
)

type InstanceConfiguration struct {
	AddressFamily AddressFamily
}

type LoadBalancerConfiguration struct {
	Enabled               bool
	Location              string
	NetworkZone           string
	DisablePrivateIngress bool
	UsePrivateIP          bool
	DisableIPv6           bool
}

type NetworkConfiguration struct {
	NameOrID             string
	DisableAttachedCheck bool
}

type RouteConfiguration struct {
	Enabled bool
}

type HCCMConfiguration struct {
	HCloudClient HCloudClientConfiguration
	Metrics      MetricsConfiguration
	Instance     InstanceConfiguration
	LoadBalancer LoadBalancerConfiguration
	Network      NetworkConfiguration
	Route        RouteConfiguration
}

// Read evaluates all environment variables and returns a [HCCMConfiguration]. It only validates as far as
// it needs to parse the values. For business logic validation, check out [HCCMConfiguration.Validate].
func Read() (HCCMConfiguration, error) {
	var err error
	// Collect all errors and return them as one.
	// This helps users because they will see all the errors at once
	// instead of having to fix them one by one.
	var errs []error
	var cfg HCCMConfiguration

	cfg.HCloudClient.Token = os.Getenv(hcloudToken)
	cfg.HCloudClient.Endpoint = os.Getenv(hcloudEndpoint)
	cfg.HCloudClient.Debug, err = getEnvBool(hcloudDebug, false)
	if err != nil {
		errs = append(errs, err)
	}

	cfg.Metrics.Enabled, err = getEnvBool(hcloudMetricsEnabled, true)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.Metrics.Address = hcloudMetricsAddress

	// Validation happens in [HCCMConfiguration.Validate]
	cfg.Instance.AddressFamily = AddressFamily(os.Getenv(hcloudInstancesAddressFamily))
	if cfg.Instance.AddressFamily == "" {
		cfg.Instance.AddressFamily = AddressFamilyIPv4
	}

	cfg.LoadBalancer.Enabled, err = getEnvBool(hcloudLoadBalancersEnabled, true)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.LoadBalancer.Location = os.Getenv(hcloudLoadBalancersLocation)
	cfg.LoadBalancer.NetworkZone = os.Getenv(hcloudLoadBalancersNetworkZone)
	cfg.LoadBalancer.DisablePrivateIngress, err = getEnvBool(hcloudLoadBalancersDisablePrivateIngress, false)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.LoadBalancer.UsePrivateIP, err = getEnvBool(hcloudLoadBalancersUsePrivateIP, false)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.LoadBalancer.DisableIPv6, err = getEnvBool(hcloudLoadBalancersDisableIPv6, false)
	if err != nil {
		errs = append(errs, err)
	}

	cfg.Network.NameOrID = os.Getenv(hcloudNetwork)
	cfg.Network.DisableAttachedCheck, err = getEnvBool(hcloudNetworkDisableAttachedCheck, false)
	if err != nil {
		errs = append(errs, err)
	}

	// Enabling Routes only makes sense when a Network is configured, otherwise there is no network to add the routes to.
	if cfg.Network.NameOrID != "" {
		cfg.Route.Enabled, err = getEnvBool(hcloudNetworkRoutesEnabled, true)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return HCCMConfiguration{}, errors.Join(errs...)
	}
	return cfg, nil
}

func (c HCCMConfiguration) Validate() (err error) {
	// Collect all errors and return them as one.
	// This helps users because they will see all the errors at once
	// instead of having to fix them one by one.
	var errs []error

	if c.HCloudClient.Token == "" {
		errs = append(errs, fmt.Errorf("environment variable %q is required", hcloudToken))
	} else if len(c.HCloudClient.Token) != 64 {
		errs = append(errs, fmt.Errorf("entered token is invalid (must be exactly 64 characters long)"))
	}

	if c.Instance.AddressFamily != AddressFamilyDualStack && c.Instance.AddressFamily != AddressFamilyIPv4 && c.Instance.AddressFamily != AddressFamilyIPv6 {
		errs = append(errs, fmt.Errorf("invalid value for %q, expect one of: %s,%s,%s", hcloudInstancesAddressFamily, AddressFamilyIPv4, AddressFamilyIPv6, AddressFamilyDualStack))
	}

	if c.LoadBalancer.Location != "" && c.LoadBalancer.NetworkZone != "" {
		errs = append(errs, fmt.Errorf("invalid value for %q/%q, only one of them can be set", hcloudLoadBalancersLocation, hcloudLoadBalancersNetworkZone))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// getEnvBool returns the boolean parsed from the environment variable with the given key and a potential error
// parsing the var. Returns the default value if the env var is unset.
func getEnvBool(key string, defaultValue bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s: %v", key, err)
	}

	return b, nil
}
