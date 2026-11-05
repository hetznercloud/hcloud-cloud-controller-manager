package events

type Reason string

// Instances controller event reasons.
const (
	PossibleNodeDeletion    Reason = "PossibleNodeDeletion"
	InstanceLookupFailed    Reason = "InstanceLookupFailed"
	IgnoredExternalIPv6     Reason = "IgnoredExternalIPv6"
	InvalidExternalIPv6     Reason = "InvalidExternalIPv6"
	MisconfiguredInternalIP Reason = "MisconfiguredInternalIP"
)

// Route controller event reasons.
const (
	ClusterCIDRMisconfigured Reason = "ClusterCIDRMisconfigured"
)

// Load balancer controller event reasons.
const (
	PrimaryIPChangeUnsupported    Reason = "PrimaryIPChangeUnsupported"
	UnknownProviderIDPrefix       Reason = "UnknownProviderIDPrefix"
	InternalIPNotConfigured       Reason = "InternalIPNotConfigured"
	ExternalIPTargetReplaced      Reason = "ExternalIPTargetReplaced"
	ServerNotFound                Reason = "ServerNotFound"
	MaxTargetsReached             Reason = "MaxTargetsReached"
	UnsupportedProtocolConfigured Reason = "UnsupportedProtocolConfigured"
	LoadBalancerTypeUnconfigured  Reason = "LoadBalancerTypeUnconfigured"
	LoadBalancerTypeDeprecated    Reason = "LoadBalancerTypeDeprecated"
	InvalidLoadBalancerAnnotation Reason = "InvalidLoadBalancerAnnotation"
)
