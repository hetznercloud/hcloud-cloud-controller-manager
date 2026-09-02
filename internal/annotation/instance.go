package annotation

const (
	// RobotExternalIPv6 configures the IPv6 address a Robot server is reachable at. We use it
	// as the IPv6 ExternalIP of the Node instead of deriving one.
	RobotExternalIPv6 IP = "instance.hetzner.cloud/robot-external-ipv6"
)
