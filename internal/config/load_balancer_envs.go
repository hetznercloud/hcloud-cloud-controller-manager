package config

const (
	// hcloudLoadBalancersEnabled controls whether the load balancer controller of HCCM should run.
	//
	// Type: bool
	// Default: true
	hcloudLoadBalancersEnabled = "HCLOUD_LOAD_BALANCERS_ENABLED"

	// hcloudLoadBalancersLocation specifies the default location where the Load Balancer will be
	// created in.
	//
	// Mutually exclusive with [hcloudLoadBalancersNetworkZone].
	//
	// Type: string
	hcloudLoadBalancersLocation = "HCLOUD_LOAD_BALANCERS_LOCATION"

	// hcloudLoadBalancersNetworkZone specifies the default network zone where the Load Balancer will be
	// created in.
	//
	// Mutually exclusive with [hcloudLoadBalancersLocation].
	//
	// Type: string
	hcloudLoadBalancersNetworkZone = "HCLOUD_LOAD_BALANCERS_NETWORK_ZONE"

	// hcloudLoadBalancersDisablePrivateIngress disables the use of the private network for ingress by default.
	//
	// Type: bool
	// Default: false
	hcloudLoadBalancersDisablePrivateIngress = "HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS"

	// hcloudLoadBalancersUsePrivateIP configures the Load Balancer to use the private IP for
	// Load Balancer server targets by default.
	//
	// Type: bool
	// Default: false
	hcloudLoadBalancersUsePrivateIP = "HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP"

	// hcloudLoadBalancersDisableIPv6 disables the use of IPv6 for the Load Balancer by default.
	//
	// Type: bool
	// Default: false
	hcloudLoadBalancersDisableIPv6 = "HCLOUD_LOAD_BALANCERS_DISABLE_IPV6"

	// hcloudLoadBalancersAlgorithmType configures the default Load Balancer algorithm.
	//
	// Type: round_robin | least_connections
	// Default: round_robin
	hcloudLoadBalancersAlgorithmType = "HCLOUD_LOAD_BALANCERS_ALGORITHM_TYPE"

	// hcloudLoadBalancersDisablePublicNetwork disables the public interface of the Load Balancer by default.
	//
	// Type: bool
	// Default: false
	hcloudLoadBalancersDisablePublicNetwork = "HCLOUD_LOAD_BALANCERS_DISABLE_PUBLIC_NETWORK"

	// hcloudLoadBalancersHealthCheckInterval configures the default time interval in seconds
	// in which health checks are performed.
	//
	// Type: int
	// Default: 10
	hcloudLoadBalancersHealthCheckInterval = "HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_INTERVAL"

	// hcloudLoadBalancersHealthCheckRetries configures the default amount of unsuccessful retries
	// needed until a target is considered unhealthy.
	//
	// Type: int
	// Default: 3
	hcloudLoadBalancersHealthCheckRetries = "HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_RETRIES"

	// hcloudLoadBalancersHealthCheckTimeout configures the default time in seconds after an attempt is
	// considered a timeout.
	//
	// Type: int
	// Default: 15
	hcloudLoadBalancersHealthCheckTimeout = "HCLOUD_LOAD_BALANCERS_HEALTH_CHECK_TIMEOUT"

	// hcloudLoadBalancersPrivateSubnetIPRange configures the default IP range in CIDR block notation of
	// the subnet to attach to.
	//
	// Type: string
	hcloudLoadBalancersPrivateSubnetIPRange = "HCLOUD_LOAD_BALANCERS_PRIVATE_SUBNET_IP_RANGE"

	// hcloudLoadBalancersType configures the default Load Balancer type this Load Balancer should be created with.
	//
	// Type: string
	// Default: lb11
	hcloudLoadBalancersType = "HCLOUD_LOAD_BALANCERS_TYPE"

	// hcloudLoadBalancersUsesProxyProtocol enables the proxyprotocol for a Load Balancer service by default.
	//
	// Type: bool
	// Default: false
	hcloudLoadBalancersUsesProxyProtocol = "HCLOUD_LOAD_BALANCERS_USES_PROXYPROTOCOL"
)
