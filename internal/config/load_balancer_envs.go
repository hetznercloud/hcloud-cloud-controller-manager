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
	// Mutually exclusive with hcloudLoadBalancersNetworkZone.
	//
	// Type: string
	hcloudLoadBalancersLocation = "HCLOUD_LOAD_BALANCERS_LOCATION"

	// hcloudLoadBalancersNetworkZone specifies the default network zone where the Load Balancer will be
	// created in.
	//
	// Mutually exclusive with hcloudLoadBalancersLocation.
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
)
