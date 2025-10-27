package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/envutil"
)

const (
	hcloudToken    = "HCLOUD_TOKEN"
	hcloudEndpoint = "HCLOUD_ENDPOINT"
	hcloudNetwork  = "HCLOUD_NETWORK"
	hcloudDebug    = "HCLOUD_DEBUG"

	robotEnabled            = "ROBOT_ENABLED"
	robotUser               = "ROBOT_USER"
	robotPassword           = "ROBOT_PASSWORD"
	robotCacheTimeout       = "ROBOT_CACHE_TIMEOUT"
	robotRateLimitWaitTime  = "ROBOT_RATE_LIMIT_WAIT_TIME"
	robotForwardInternalIPs = "ROBOT_FORWARD_INTERNAL_IPS"

	hcloudInstancesAddressFamily = "HCLOUD_INSTANCES_ADDRESS_FAMILY"

	// Disable the "master/server is attached to the network" check against the metadata service.
	hcloudNetworkDisableAttachedCheck = "HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK"
	hcloudNetworkRoutesEnabled        = "HCLOUD_NETWORK_ROUTES_ENABLED"

	hcloudLoadBalancersAlgorithmType         = "HCLOUD_LOAD_BALANCERS_ALGORITHM_TYPE"
	hcloudLoadBalancersDisableIPv6           = "HCLOUD_LOAD_BALANCERS_DISABLE_IPV6"
	hcloudLoadBalancersDisablePrivateIngress = "HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS"
	hcloudLoadBalancersDisablePublicNetwork  = "HCLOUD_LOAD_BALANCERS_DISABLE_PUBLIC_NETWORK"
	hcloudLoadBalancersEnabled               = "HCLOUD_LOAD_BALANCERS_ENABLED"
	hcloudLoadBalancersHealthCheckInterval   = "HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_INTERVAL"
	hcloudLoadBalancersHealthCheckRetries    = "HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_RETRIES"
	hcloudLoadBalancersHealthCheckTimeout    = "HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_TIMEOUT"
	hcloudLoadBalancersLocation              = "HCLOUD_LOAD_BALANCERS_LOCATION"
	hcloudLoadBalancersNetworkZone           = "HCLOUD_LOAD_BALANCERS_NETWORK_ZONE"
	hcloudLoadBalancersPrivateSubnetIPRange  = "HCLOUD_LOAD_BALANCERS_PRIVATE_SUBNET_IP_RANGE"
	hcloudLoadBalancersType                  = "HCLOUD_LOAD_BALANCERS_TYPE"
	hcloudLoadBalancersUsePrivateIP          = "HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP"
	hcloudLoadBalancersUsesProxyProtocol     = "HCLOUD_LOAD_BALANCERS_USES_PROXYPROTOCOL"

	hcloudMetricsEnabled = "HCLOUD_METRICS_ENABLED"
	hcloudMetricsAddress = "HCLOUD_METRICS_ADDRESS"
)

type HCloudClientConfiguration struct {
	Token    string
	Endpoint string
	Debug    bool
}

type RobotConfiguration struct {
	Enabled           bool
	User              string
	Password          string
	CacheTimeout      time.Duration
	RateLimitWaitTime time.Duration
	// ForwardInternalIPs is enabled by default.
	ForwardInternalIPs bool
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
	AlgorithmType         hcloud.LoadBalancerAlgorithmType
	DisablePublicNetwork  *bool
	Enabled               bool
	HealthCheckInterval   time.Duration
	HealthCheckRetries    int
	HealthCheckTimeout    time.Duration
	IPv6Enabled           bool
	Location              string
	NetworkZone           string
	PrivateIngressEnabled bool
	PrivateIPEnabled      bool
	PrivateSubnetIPRange  string
	ProxyProtocolEnabled  *bool
	Type                  string
}

type NetworkConfiguration struct {
	NameOrID             string
	AttachedCheckEnabled bool
}

type RouteConfiguration struct {
	Enabled bool
}

type HCCMConfiguration struct {
	HCloudClient HCloudClientConfiguration
	Robot        RobotConfiguration
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

	cfg.HCloudClient.Token, err = envutil.LookupEnvWithFile(hcloudToken)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.HCloudClient.Endpoint = os.Getenv(hcloudEndpoint)
	cfg.HCloudClient.Debug, err = getEnvBool(hcloudDebug, false)
	if err != nil {
		errs = append(errs, err)
	}

	cfg.Robot.Enabled, err = getEnvBool(robotEnabled, false)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.Robot.User, err = envutil.LookupEnvWithFile(robotUser)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.Robot.Password, err = envutil.LookupEnvWithFile(robotPassword)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.Robot.CacheTimeout, err = getEnvDuration(robotCacheTimeout)
	if err != nil {
		errs = append(errs, err)
	}
	if cfg.Robot.CacheTimeout == 0 {
		cfg.Robot.CacheTimeout = 5 * time.Minute
	}
	cfg.Robot.RateLimitWaitTime, err = getEnvDuration(robotRateLimitWaitTime)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.Robot.ForwardInternalIPs, err = getEnvBool(robotForwardInternalIPs, true)
	if err != nil {
		errs = append(errs, err)
	}
	// Robot needs to be enabled
	cfg.Robot.ForwardInternalIPs = cfg.Robot.ForwardInternalIPs && cfg.Robot.Enabled

	cfg.Metrics.Enabled, err = getEnvBool(hcloudMetricsEnabled, true)
	if err != nil {
		errs = append(errs, err)
	}

	if addr, ok := os.LookupEnv(hcloudMetricsAddress); ok {
		cfg.Metrics.Address = addr
	} else {
		cfg.Metrics.Address = ":8233"
	}

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

	disablePrivateIngress, err := getEnvBool(hcloudLoadBalancersDisablePrivateIngress, false)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.LoadBalancer.PrivateIngressEnabled = !disablePrivateIngress // Invert the logic, as the env var is prefixed with DISABLE_.

	cfg.LoadBalancer.PrivateIPEnabled, err = getEnvBool(hcloudLoadBalancersUsePrivateIP, false)
	if err != nil {
		errs = append(errs, err)
	}

	cfg.LoadBalancer.ProxyProtocolEnabled, err = getEnvBoolPtr(hcloudLoadBalancersUsesProxyProtocol)
	if err != nil {
		errs = append(errs, err)
	}

	disableIPv6, err := getEnvBool(hcloudLoadBalancersDisableIPv6, false)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.LoadBalancer.IPv6Enabled = !disableIPv6 // Invert the logic, as the env var is prefixed with DISABLE_.

	if subnetRange, ok := os.LookupEnv(hcloudLoadBalancersPrivateSubnetIPRange); ok {
		cfg.LoadBalancer.PrivateSubnetIPRange = subnetRange
	}

	if algorithmType, ok := os.LookupEnv(hcloudLoadBalancersAlgorithmType); ok {
		alg, parseErr := parseLoadBalancerAlgorithmType(algorithmType)
		if parseErr != nil {
			errs = append(errs, fmt.Errorf("failed to parse %s: %w", hcloudLoadBalancersAlgorithmType, parseErr))
		} else {
			cfg.LoadBalancer.AlgorithmType = alg
		}
	}

	if interval, ok := os.LookupEnv(hcloudLoadBalancersHealthCheckInterval); ok {
		d, parseErr := time.ParseDuration(interval)
		if parseErr != nil {
			errs = append(errs, fmt.Errorf("failed to parse %s: %w", hcloudLoadBalancersHealthCheckInterval, parseErr))
		} else {
			cfg.LoadBalancer.HealthCheckInterval = d
		}
	}

	if timeout, ok := os.LookupEnv(hcloudLoadBalancersHealthCheckTimeout); ok {
		d, parseErr := time.ParseDuration(timeout)
		if parseErr != nil {
			errs = append(errs, fmt.Errorf("failed to parse %s: %w", hcloudLoadBalancersHealthCheckTimeout, parseErr))
		} else {
			cfg.LoadBalancer.HealthCheckTimeout = d
		}
	}

	if retries, ok := os.LookupEnv(hcloudLoadBalancersHealthCheckRetries); ok {
		v, parseErr := strconv.Atoi(retries)
		if parseErr != nil {
			errs = append(errs, fmt.Errorf("failed to parse %s: %w", hcloudLoadBalancersHealthCheckRetries, parseErr))
		} else {
			cfg.LoadBalancer.HealthCheckRetries = v
		}
	}

	cfg.LoadBalancer.DisablePublicNetwork, err = getEnvBoolPtr(hcloudLoadBalancersDisablePublicNetwork)
	if err != nil {
		errs = append(errs, err)
	}

	if lbType, ok := os.LookupEnv(hcloudLoadBalancersType); ok {
		cfg.LoadBalancer.Type = lbType
	}

	cfg.Network.NameOrID = os.Getenv(hcloudNetwork)
	disableAttachedCheck, err := getEnvBool(hcloudNetworkDisableAttachedCheck, false)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.Network.AttachedCheckEnabled = !disableAttachedCheck // Invert the logic, as the env var is prefixed with DISABLE_.

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
		klog.Warningf("unrecognized token format, expected 64 characters, got %d, proceeding anyway", len(c.HCloudClient.Token))
	}

	if c.Instance.AddressFamily != AddressFamilyDualStack && c.Instance.AddressFamily != AddressFamilyIPv4 && c.Instance.AddressFamily != AddressFamilyIPv6 {
		errs = append(errs, fmt.Errorf("invalid value for %q, expect one of: %s,%s,%s", hcloudInstancesAddressFamily, AddressFamilyIPv4, AddressFamilyIPv6, AddressFamilyDualStack))
	}

	if c.LoadBalancer.Location != "" && c.LoadBalancer.NetworkZone != "" {
		errs = append(errs, fmt.Errorf("invalid value for %q/%q, only one of them can be set", hcloudLoadBalancersLocation, hcloudLoadBalancersNetworkZone))
	}

	if c.Robot.Enabled {
		if c.Robot.User == "" {
			errs = append(errs, fmt.Errorf("environment variable %q is required if Robot support is enabled", robotUser))
		}
		if c.Robot.Password == "" {
			errs = append(errs, fmt.Errorf("environment variable %q is required if Robot support is enabled", robotPassword))
		}

		if c.Route.Enabled {
			errs = append(errs, fmt.Errorf("using Routes with Robot is not supported"))
		}
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
		return false, fmt.Errorf("failed to parse %s: %w", key, err)
	}

	return b, nil
}

// getEnvBoolPtr returns a pointer to the boolean parsed from the environment variable with the given key.
// Returns nil if the env var is unset.
func getEnvBoolPtr(key string) (*bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return nil, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", key, err)
	}

	return &b, nil
}

// getEnvDuration returns the duration parsed from the environment variable with the given key and a potential error
// parsing the var. Returns false if the env var is unset.
func getEnvDuration(key string) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return 0, nil
	}

	b, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s: %w", key, err)
	}

	return b, nil
}

func parseLoadBalancerAlgorithmType(value string) (hcloud.LoadBalancerAlgorithmType, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	alg := hcloud.LoadBalancerAlgorithmType(v)
	if alg == hcloud.LoadBalancerAlgorithmTypeRoundRobin || alg == hcloud.LoadBalancerAlgorithmTypeLeastConnections {
		return alg, nil
	}
	return "", fmt.Errorf("unsupported value %q", value)
}
