package annotation

const (
	// ExternalIPv6 configures the IPv6 ExternalIP of the Node, instead of deriving it from the
	// IPv6 subnet the server reports. This annotations can be set for Cloud and Robot servers.
	ExternalIPv6 IP = "instance.hetzner.cloud/external-ipv6"
)
