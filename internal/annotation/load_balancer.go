package annotation

const (
	// LBPublicIPv4 is the public IPv4 address assigned to the Load Balancer by
	// the backend.
	//
	// Type: string
	// Read-only: true
	LBPublicIPv4 Name = "load-balancer.hetzner.cloud/ipv4"

	// LBPublicIPv4RDNS is the reverse DNS record assigned to the IPv4 address of
	// the Load Balancer.
	//
	// Type: string
	// Read-only: true
	LBPublicIPv4RDNS Name = "load-balancer.hetzner.cloud/ipv4-rdns"

	// LBPublicIPv6 is the public IPv6 address assigned to the Load Balancer by
	// the backend.
	//
	// Type: string
	// Read-only: true
	LBPublicIPv6 Name = "load-balancer.hetzner.cloud/ipv6"

	// LBPublicIPv6RDNS is the reverse DNS record assigned to the IPv6 address of
	// the Load Balancer.
	//
	// Type: string
	// Read-only: true
	LBPublicIPv6RDNS Name = "load-balancer.hetzner.cloud/ipv6-rdns"

	// LBIPv6Disabled disables the use of IPv6 for the Load Balancer.
	// Set this annotation if you use external-dns.
	//
	// Type: bool
	// Default: false
	LBIPv6Disabled Name = "load-balancer.hetzner.cloud/ipv6-disabled"

	// LBName is the name of the Load Balancer. The name will be visible in
	// the Hetzner Cloud API console.
	//
	// Type: string
	LBName Name = "load-balancer.hetzner.cloud/name"

	// LBDisablePublicNetwork disables the public network of the Hetzner Cloud
	// Load Balancer. It will still have a public network assigned, but all
	// traffic is routed over the private network.
	//
	// Type: bool
	// Default: false
	LBDisablePublicNetwork Name = "load-balancer.hetzner.cloud/disable-public-network"

	// LBDisablePrivateIngress disables the use of the private network for
	// ingress.
	//
	// Type: bool
	// Default: false
	LBDisablePrivateIngress Name = "load-balancer.hetzner.cloud/disable-private-ingress"

	// LBUsePrivateIP configures the Load Balancer to use the private IP for
	// Load Balancer server targets.
	//
	// Type: bool
	// Default: false
	LBUsePrivateIP Name = "load-balancer.hetzner.cloud/use-private-ip"

	// LBPrivateIPv4 specifies the IPv4 address to assign to the load balancer in the
	// private network that it's attached to.
	//
	// Type: string
	LBPrivateIPv4 Name = "load-balancer.hetzner.cloud/private-ipv4"

	// PrivateSubnetIPRange specifies an existing subnet to which the load balancer will be attached.
	// The value must be in the CIDR notation. The subnet must belong to the network defined
	// in the CCM configuration and must already exist.
	// See: https://docs.hetzner.cloud/reference/cloud#network-actions-add-a-subnet-to-a-network
	//
	// Type: string
	PrivateSubnetIPRange Name = "load-balancer.hetzner.cloud/private-subnet-ip-range"

	// LBHostname specifies the hostname of the Load Balancer. This will be
	// used as ingress address instead of the Load Balancer IP addresses if
	// specified.
	//
	// Type: string
	LBHostname Name = "load-balancer.hetzner.cloud/hostname"

	// LBSvcProtocol specifies the protocol of the service.
	//
	// Type: tcp | http | https
	// Default: tcp
	LBSvcProtocol Name = "load-balancer.hetzner.cloud/protocol"

	// LBAlgorithmType specifies the algorithm type of the Load Balancer.
	//
	// Type: round_robin | least_connections
	// Default: round_robin
	LBAlgorithmType Name = "load-balancer.hetzner.cloud/algorithm-type"

	// LBType specifies the type of the Load Balancer.
	//
	// Type: string
	// Default: lb11
	LBType Name = "load-balancer.hetzner.cloud/type"

	// LBLocation specifies the location where the Load Balancer will be
	// created in.
	//
	// Changing the location to a different value after the load balancer was
	// created has no effect. In order to move a load balancer to a different
	// location it is necessary to delete and re-create it. Note, that this
	// will lead to the load balancer getting new public IPs assigned.
	//
	// Mutually exclusive with [LBNetworkZone].
	//
	// Type: string
	LBLocation Name = "load-balancer.hetzner.cloud/location"

	// LBNetworkZone specifies the network zone where the Load Balancer will be
	// created in.
	//
	// Changing the network zone to a different value after the load balancer
	// was created has no effect.  In order to move a load balancer to a
	// different network zone it is necessary to delete and re-create it. Note,
	// that this will lead to the load balancer getting new public IPs
	// assigned.
	//
	// Mutually exclusive with [LBLocation].
	//
	// Type: string
	LBNetworkZone Name = "load-balancer.hetzner.cloud/network-zone"

	// LBNodeSelector can be set to restrict which Nodes are added as targets to the
	// Load Balancer. It accepts a Kubernetes label selector string, using either the
	// set-based or equality-based formats.
	//
	// If the selector can not be parsed, the targets in the Load Balancer are not
	// updated and an Event is created with the error message.
	//
	// Format: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	//
	// Type: string
	LBNodeSelector Name = "load-balancer.hetzner.cloud/node-selector"

	// LBSvcProxyProtocol specifies if the Load Balancer services should
	// use the proxy protocol.
	//
	// Type: bool
	// Default: false
	LBSvcProxyProtocol Name = "load-balancer.hetzner.cloud/uses-proxyprotocol"

	// LBSvcHTTPCookieName specifies the cookie name when using  HTTP or HTTPS
	// as protocol.
	//
	// Type: string
	LBSvcHTTPCookieName Name = "load-balancer.hetzner.cloud/http-cookie-name"

	// LBSvcHTTPCookieLifetime specifies the lifetime of the HTTP cookie.
	//
	// Type: int
	LBSvcHTTPCookieLifetime Name = "load-balancer.hetzner.cloud/http-cookie-lifetime"

	// LBSvcHTTPCertificateType defines the type of certificate the Load
	// Balancer should use.
	//
	// Type: uploaded | managed
	// Default: uploaded
	LBSvcHTTPCertificateType Name = "load-balancer.hetzner.cloud/certificate-type"

	// LBSvcHTTPCertificates a comma separated list of IDs or Names of
	// Certificates assigned to the service.
	//
	// Type: string
	LBSvcHTTPCertificates Name = "load-balancer.hetzner.cloud/http-certificates"

	// LBSvcHTTPManagedCertificateName contains the name of the managed
	// certificate to create by the Cloud Controller manager. Ignored if
	// [LBSvcHTTPCertificateType] is missing or set to "uploaded".
	//
	// Type: string
	LBSvcHTTPManagedCertificateName Name = "load-balancer.hetzner.cloud/http-managed-certificate-name"

	// LBSvcHTTPManagedCertificateUseACMEStaging tells the cloud controller manager to create
	// the certificate using Let's Encrypt staging.
	//
	// This annotation is exclusively for Hetzner internal testing purposes.
	// Users should not use this annotation. There is no guarantee that it
	// remains or continues to function as it currently functions.
	//
	// Type: bool
	// Default: false
	// Internal: true
	LBSvcHTTPManagedCertificateUseACMEStaging Name = "load-balancer.hetzner.cloud/http-managed-certificate-acme-staging"

	// LBSvcHTTPManagedCertificateDomains contains a comma separated list of the
	// domain names of the managed certificate.
	//
	// All domains are used to create a single managed certificate.
	//
	// Type: string
	LBSvcHTTPManagedCertificateDomains Name = "load-balancer.hetzner.cloud/http-managed-certificate-domains"

	// LBSvcRedirectHTTP create a redirect from HTTP to HTTPS.
	//
	// Type: bool
	// Default: false
	LBSvcRedirectHTTP Name = "load-balancer.hetzner.cloud/http-redirect-http"

	// LBSvcHTTPStickySessions enables the sticky sessions feature of Hetzner
	// Cloud HTTP Load Balancers.
	//
	// Type: bool
	// Default: false
	LBSvcHTTPStickySessions Name = "load-balancer.hetzner.cloud/http-sticky-sessions"

	// LBSvcHealthCheckProtocol sets the protocol the health check should be
	// performed over.
	//
	// Type: tcp | http | https
	// Default: tcp
	LBSvcHealthCheckProtocol Name = "load-balancer.hetzner.cloud/health-check-protocol"

	// LBSvcHealthCheckPort specifies the port the health check is be performed
	// on.
	//
	// Type: int
	LBSvcHealthCheckPort Name = "load-balancer.hetzner.cloud/health-check-port"

	// LBSvcHealthCheckInterval specifies the interval in which time we perform
	// a health check in seconds.
	//
	// Type: int
	LBSvcHealthCheckInterval Name = "load-balancer.hetzner.cloud/health-check-interval"

	// LBSvcHealthCheckTimeout specifies the timeout of a single health check.
	//
	// Type: int
	LBSvcHealthCheckTimeout Name = "load-balancer.hetzner.cloud/health-check-timeout"

	// LBSvcHealthCheckRetries specifies the number of time a health check is
	// retried until a target is marked as unhealthy.
	//
	// Type: int
	LBSvcHealthCheckRetries Name = "load-balancer.hetzner.cloud/health-check-retries"

	// LBSvcHealthCheckHTTPDomain specifies the domain we try to access when
	// performing the health check.
	//
	// Type: string
	LBSvcHealthCheckHTTPDomain Name = "load-balancer.hetzner.cloud/health-check-http-domain"

	// LBSvcHealthCheckHTTPPath specifies the path we try to access when
	// performing the health check.
	//
	// Type: string
	LBSvcHealthCheckHTTPPath Name = "load-balancer.hetzner.cloud/health-check-http-path"

	// LBSvcHealthCheckHTTPValidateCertificate specifies whether the health
	// check should validate the SSL certificate that comes from the target
	// nodes.
	//
	// Type: bool
	LBSvcHealthCheckHTTPValidateCertificate Name = "load-balancer.hetzner.cloud/health-check-http-validate-certificate"

	// LBSvcHealthCheckHTTPStatusCodes is a comma separated list of HTTP status
	// codes which we expect.
	//
	// Type: string
	LBSvcHealthCheckHTTPStatusCodes Name = "load-balancer.hetzner.cloud/http-status-codes"

	// LBID is the ID assigned to the Hetzner Cloud Load Balancer by the
	// backend.
	//
	// Deprecated: This annotation is not used. It is reserved for possible future use.
	//
	// Type: string
	// Read-only: true
	LBID Name = "load-balancer.hetzner.cloud/id"
)
