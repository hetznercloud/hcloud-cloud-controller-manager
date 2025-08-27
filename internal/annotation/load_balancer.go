package annotation

const (
	// Type: string.
	LBPublicIPv4 Name = "load-balancer.hetzner.cloud/ipv4"

	// Type: string.
	LBPublicIPv4RDNS Name = "load-balancer.hetzner.cloud/ipv4-rdns"

	// Type: string.
	LBPublicIPv6 Name = "load-balancer.hetzner.cloud/ipv6"

	// Type: string.
	LBPublicIPv6RDNS Name = "load-balancer.hetzner.cloud/ipv6-rdns"

	// Default: false.
	LBIPv6Disabled Name = "load-balancer.hetzner.cloud/ipv6-disabled"

	// Type: string.
	LBName Name = "load-balancer.hetzner.cloud/name"

	// Default: false.
	LBDisablePublicNetwork Name = "load-balancer.hetzner.cloud/disable-public-network"

	// Default: false.
	LBDisablePrivateIngress Name = "load-balancer.hetzner.cloud/disable-private-ingress"

	// Default: false.
	LBUsePrivateIP Name = "load-balancer.hetzner.cloud/use-private-ip"

	// Type: string.
	LBPrivateIPv4 Name = "load-balancer.hetzner.cloud/private-ipv4"

	// Type: string.
	LBHostname Name = "load-balancer.hetzner.cloud/hostname"

	// Default: tcp.
	LBSvcProtocol Name = "load-balancer.hetzner.cloud/protocol"

	// Default: round_robin.
	LBAlgorithmType Name = "load-balancer.hetzner.cloud/algorithm-type"

	// Default: lb11.
	LBType Name = "load-balancer.hetzner.cloud/type"

	// Type: string.
	LBLocation Name = "load-balancer.hetzner.cloud/location"

	// Type: string.
	LBNetworkZone Name = "load-balancer.hetzner.cloud/network-zone"

	// Type: string.
	LBNodeSelector Name = "load-balancer.hetzner.cloud/node-selector"

	// Default: false.
	LBSvcProxyProtocol Name = "load-balancer.hetzner.cloud/uses-proxyprotocol"

	// Type: string.
	LBSvcHTTPCookieName Name = "load-balancer.hetzner.cloud/http-cookie-name"

	// Type: int.
	LBSvcHTTPCookieLifetime Name = "load-balancer.hetzner.cloud/http-cookie-lifetime"

	// Default: uploaded.
	LBSvcHTTPCertificateType Name = "load-balancer.hetzner.cloud/certificate-type"

	// Type: string.
	LBSvcHTTPCertificates Name = "load-balancer.hetzner.cloud/http-certificates"

	// Type: string.
	LBSvcHTTPManagedCertificateName Name = "load-balancer.hetzner.cloud/http-managed-certificate-name"

	// Default: false.
	LBSvcHTTPManagedCertificateUseACMEStaging Name = "load-balancer.hetzner.cloud/http-managed-certificate-acme-staging"

	// Type: string.
	LBSvcHTTPManagedCertificateDomains Name = "load-balancer.hetzner.cloud/http-managed-certificate-domains"

	// Default: false.
	LBSvcRedirectHTTP Name = "load-balancer.hetzner.cloud/http-redirect-http"

	// Default: false.
	LBSvcHTTPStickySessions Name = "load-balancer.hetzner.cloud/http-sticky-sessions"

	// Default: tcp.
	LBSvcHealthCheckProtocol Name = "load-balancer.hetzner.cloud/health-check-protocol"

	// Type: int.
	LBSvcHealthCheckPort Name = "load-balancer.hetzner.cloud/health-check-port"

	// Type: int.
	LBSvcHealthCheckInterval Name = "load-balancer.hetzner.cloud/health-check-interval"

	// Type: int.
	LBSvcHealthCheckTimeout Name = "load-balancer.hetzner.cloud/health-check-timeout"

	// Type: int.
	LBSvcHealthCheckRetries Name = "load-balancer.hetzner.cloud/health-check-retries"

	// Type: string.
	LBSvcHealthCheckHTTPDomain Name = "load-balancer.hetzner.cloud/health-check-http-domain"

	// Type: string.
	LBSvcHealthCheckHTTPPath Name = "load-balancer.hetzner.cloud/health-check-http-path"

	// Type: bool.
	LBSvcHealthCheckHTTPValidateCertificate Name = "load-balancer.hetzner.cloud/health-check-http-validate-certificate"

	// Type: string.
	LBSvcHealthCheckHTTPStatusCodes Name = "load-balancer.hetzner.cloud/http-status-codes"

	// Read-only: true.
	LBID Name = "load-balancer.hetzner.cloud/id"
)
