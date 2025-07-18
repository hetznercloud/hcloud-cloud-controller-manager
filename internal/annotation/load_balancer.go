package annotation

const (
	// LBPublicIPv4 is the public IPv4 address assigned to the Load Balancer by
	// the backend. Read-only.
	LBPublicIPv4 Name = "load-balancer.hetzner.cloud/ipv4"

	// LBPublicIPv4RDNS is the reverse DNS record assigned to the IPv4 address of
	// the Load Balancer.
	LBPublicIPv4RDNS Name = "load-balancer.hetzner.cloud/ipv4-rdns"

	// LBPublicIPv6 is the public IPv6 address assigned to the Load Balancer by
	// the backend. Read-only.
	LBPublicIPv6 Name = "load-balancer.hetzner.cloud/ipv6"

	// LBPublicIPv6RDNS is the reverse DNS record assigned to the IPv6 address of
	// the Load Balancer.
	LBPublicIPv6RDNS Name = "load-balancer.hetzner.cloud/ipv6-rdns"

	// LBIPv6Disabled disables the use of IPv6 for the Load Balancer.
	//
	// Set this annotation if you use external-dns.
	//
	// Default: false.
	LBIPv6Disabled Name = "load-balancer.hetzner.cloud/ipv6-disabled"

	// LBName is the name of the Load Balancer. The name will be visible in
	// the Hetzner Cloud API console.
	LBName Name = "load-balancer.hetzner.cloud/name"

	// LBDisablePublicNetwork disables the public network of the Hetzner Cloud
	// Load Balancer. It will still have a public network assigned, but all
	// traffic is routed over the private network.
	LBDisablePublicNetwork Name = "load-balancer.hetzner.cloud/disable-public-network"

	// LBDisablePrivateIngress disables the use of the private network for
	// ingress.
	LBDisablePrivateIngress Name = "load-balancer.hetzner.cloud/disable-private-ingress"

	// LBUsePrivateIP configures the Load Balancer to use the private IP for
	// Load Balancer server targets.
	LBUsePrivateIP Name = "load-balancer.hetzner.cloud/use-private-ip"

	// LBPrivateIPv4 specifies the IPv4 address to assign to the load balancer in the
	// private network that it's attached to.
	LBPrivateIPv4 Name = "load-balancer.hetzner.cloud/private-ipv4"

	// LBHostname specifies the hostname of the Load Balancer. This will be
	// used as ingress address instead of the Load Balancer IP addresses if
	// specified.
	LBHostname Name = "load-balancer.hetzner.cloud/hostname"

	// LBSvcProtocol specifies the protocol of the service. Default: tcp, Possible
	// values: tcp, http, https
	LBSvcProtocol Name = "load-balancer.hetzner.cloud/protocol"

	// LBSvcProtocolPorts specifies the protocol per port for the service. This allows
	// different ports to use different protocols. Format: "80:http,443:https,9000:tcp"
	// If set, this takes precedence over LBSvcProtocol for the specified ports.
	LBSvcProtocolPorts Name = "load-balancer.hetzner.cloud/protocol-ports"

	// LBAlgorithmType specifies the algorithm type of the Load Balancer.
	//
	// Possible values: round_robin, least_connections
	//
	// Default: round_robin.
	LBAlgorithmType Name = "load-balancer.hetzner.cloud/algorithm-type"

	// LBType specifies the type of the Load Balancer.
	//
	// Default: lb11.
	LBType Name = "load-balancer.hetzner.cloud/type"

	// LBLocation specifies the location where the Load Balancer will be
	// created in.
	//
	// Changing the location to a different value after the load balancer was
	// created has no effect. In order to move a load balancer to a different
	// location it is necessary to delete and re-create it. Note, that this
	// will lead to the load balancer getting new public IPs assigned.
	//
	// Mutually exclusive with LBNetworkZone.
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
	// Mutually exclusive with LBLocation.
	LBNetworkZone Name = "load-balancer.hetzner.cloud/network-zone"

	// LBNodeSelector can be set to restrict which Nodes are added as targets to the
	// Load Balancer. It accepts a Kubernetes label selector string, using either the
	// set-based or equality-based formats.
	//
	// If the selector can not be parsed, the targets in the Load Balancer are not
	// updated and an Event is created with the error message.
	//
	// Format: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	LBNodeSelector Name = "load-balancer.hetzner.cloud/node-selector"

	// LBSvcProxyProtocol specifies if the Load Balancer services should
	// use the proxy protocol.
	//
	// Default: false.
	LBSvcProxyProtocol Name = "load-balancer.hetzner.cloud/uses-proxyprotocol"

	// LBSvcHTTPCookieName specifies the cookie name when using  HTTP or HTTPS
	// as protocol.
	LBSvcHTTPCookieName Name = "load-balancer.hetzner.cloud/http-cookie-name"

	// LBSvcHTTPCookieLifetime specifies the lifetime of the HTTP cookie.
	LBSvcHTTPCookieLifetime Name = "load-balancer.hetzner.cloud/http-cookie-lifetime"

	// LBSvcHTTPCertificateType defines the type of certificate the Load
	// Balancer should use.
	//
	// Possible values are "uploaded" and "managed".
	//
	// If not set LBSvcHTTPCertificateType defaults to "uploaded".
	// LBSvcHTTPManagedCertificateDomains is ignored in this case.
	//
	// HTTPS only.
	LBSvcHTTPCertificateType Name = "load-balancer.hetzner.cloud/certificate-type"

	// LBSvcHTTPCertificates a comma separated list of IDs or Names of
	// Certificates assigned to the service.
	//
	// HTTPS only.
	LBSvcHTTPCertificates Name = "load-balancer.hetzner.cloud/http-certificates"

	// LBSvcHTTPCertificatesPorts specifies certificates per port for HTTPS services.
	// Format: "443:cert1,cert2;8443:cert3,cert4"
	// If set, this takes precedence over LBSvcHTTPCertificates for the specified ports.
	// 
	// HTTPS only.
	LBSvcHTTPCertificatesPorts Name = "load-balancer.hetzner.cloud/http-certificates-ports"

	// LBSvcHTTPManagedCertificateName contains the names of the managed
	// certificate to create by the Cloud Controller manager. Ignored if
	// LBSvcHTTPCertificateType is missing or set to "uploaded". Optional.
	LBSvcHTTPManagedCertificateName Name = "load-balancer.hetzner.cloud/http-managed-certificate-name"

	// LBSvcHTTPManagedCertificateUseACMEStaging tells the cloud controller manager to create
	// the certificate using Let's Encrypt staging.
	//
	// This annotation is exclusively for Hetzner internal testing purposes.
	// Users should not use this annotation. There is no guarantee that it
	// remains or continues to function as it currently functions.
	LBSvcHTTPManagedCertificateUseACMEStaging Name = "load-balancer.hetzner.cloud/http-managed-certificate-acme-staging"

	// LBSvcHTTPManagedCertificateDomains contains a coma separated list of the
	// domain names of the managed certificate.
	//
	// All domains are used to create a single managed certificate.
	LBSvcHTTPManagedCertificateDomains Name = "load-balancer.hetzner.cloud/http-managed-certificate-domains"

	// LBSvcRedirectHTTP create a redirect from HTTP to HTTPS. HTTPS only.
	LBSvcRedirectHTTP Name = "load-balancer.hetzner.cloud/http-redirect-http"

	// LBSvcHTTPStickySessions enables the sticky sessions feature of Hetzner
	// Cloud HTTP Load Balancers.
	//
	// Default: false.
	LBSvcHTTPStickySessions Name = "load-balancer.hetzner.cloud/http-sticky-sessions"

	// LBSvcHealthCheckProtocol sets the protocol the health check should be
	// performed over.
	//
	// Possible values: tcp, http, https
	//
	// Default: tcp.
	LBSvcHealthCheckProtocol Name = "load-balancer.hetzner.cloud/health-check-protocol"

	// LBSvcHealthCheckPort specifies the port the health check is be performed
	// on.
	LBSvcHealthCheckPort Name = "load-balancer.hetzner.cloud/health-check-port"

	// LBSvcHealthCheckInterval specifies the interval in which time we perform
	// a health check in seconds.
	LBSvcHealthCheckInterval Name = "load-balancer.hetzner.cloud/health-check-interval"

	// LBSvcHealthCheckTimeout specifies the timeout of a single health check.
	LBSvcHealthCheckTimeout Name = "load-balancer.hetzner.cloud/health-check-timeout"

	// LBSvcHealthCheckRetries specifies the number of time a health check is
	// retried until a target is marked as unhealthy.
	LBSvcHealthCheckRetries Name = "load-balancer.hetzner.cloud/health-check-retries"

	// LBSvcHealthCheckHTTPDomain specifies the domain we try to access when
	// performing the health check.
	LBSvcHealthCheckHTTPDomain Name = "load-balancer.hetzner.cloud/health-check-http-domain"

	// LBSvcHealthCheckHTTPPath specifies the path we try to access when
	// performing the health check.
	LBSvcHealthCheckHTTPPath Name = "load-balancer.hetzner.cloud/health-check-http-path"

	// LBSvcHealthCheckHTTPValidateCertificate specifies whether the health
	// check should validate the SSL certificate that comes from the target
	// nodes.
	LBSvcHealthCheckHTTPValidateCertificate Name = "load-balancer.hetzner.cloud/health-check-http-validate-certificate"

	// LBSvcHealthCheckHTTPStatusCodes is a comma separated list of HTTP status
	// codes which we expect.
	LBSvcHealthCheckHTTPStatusCodes Name = "load-balancer.hetzner.cloud/http-status-codes"

	// LBID is the ID assigned to the Hetzner Cloud Load Balancer by the
	// backend. Read-only.
	//
	// Deprecated: This annotation is not used. It is reserved for possible future use.
	LBID Name = "load-balancer.hetzner.cloud/id"
)
