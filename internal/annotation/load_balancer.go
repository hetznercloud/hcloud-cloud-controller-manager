package annotation

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/hcloud"
	v1 "k8s.io/api/core/v1"
)

const (
	// LBID is the ID assigned to the Hetzner Cloud Load Balancer by the
	// backend. Read-only.
	LBID Name = "load-balancer.hetzner.cloud/id"

	// LBPublicIPv4 is the public IPv4 address assigned to the Load Balancer by
	// the backend. Read-only.
	LBPublicIPv4 Name = "load-balancer.hetzner.cloud/ipv4"

	// LBPublicIPv4RDNS is the reverse DNS record assigned to the IPv4 address of
	// the Load Balancer
	LBPublicIPv4RDNS Name = "load-balancer.hetzner.cloud/ipv4-rdns"

	// LBPublicIPv6 is the public IPv6 address assigned to the Load Balancer by
	// the backend. Read-only.
	LBPublicIPv6 Name = "load-balancer.hetzner.cloud/ipv6"

	// LBPublicIPv6RDNS is the reverse DNS record assigned to the IPv6 address of
	// the Load Balancer
	LBPublicIPv6RDNS Name = "load-balancer.hetzner.cloud/ipv6-rdns"

	// LBIPv6Disabled disables the use of IPv6 for the Load Balancer.
	//
	// Set this annotation if you use external-dns.
	//
	// Default: false
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

	// LBHostname specifies the hostname of the Load Balancer. This will be
	// used as ingress address instead of the Load Balancer IP addresses if
	// specified.
	LBHostname Name = "load-balancer.hetzner.cloud/hostname"

	// LBSvcProtocol specifies the protocol of the service. Default: tcp, Possible
	// values: tcp, http, https
	LBSvcProtocol Name = "load-balancer.hetzner.cloud/protocol"

	// LBAlgorithmType specifies the algorithm type of the Load Balancer.
	//
	// Possible values: round_robin, least_connections
	//
	// Default: round_robin.
	LBAlgorithmType Name = "load-balancer.hetzner.cloud/algorithm-type"

	// LBType specifies the type of the Load Balancer.
	//
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

	// LBSvcProxyProtocol specifies if the Load Balancer services should
	// use the proxy protocol.
	//
	// Default: false
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
	// Default: false
	LBSvcHTTPStickySessions Name = "load-balancer.hetzner.cloud/http-sticky-sessions"

	// LBSvcHealthCheckProtocol sets the protocol the health check should be
	// performed over.
	//
	// Possible values: tcp, http, https
	//
	// Default: tcp
	LBSvcHealthCheckProtocol Name = "load-balancer.hetzner.cloud/health-check-protocol"

	// LBSvcHealthCheckPort specifies the port the health check is be performed
	// on.
	LBSvcHealthCheckPort Name = "load-balancer.hetzner.cloud/health-check-port"

	// LBSvcHealthCheckInterval specifies the interval in which time we perform
	// a health check in seconds
	LBSvcHealthCheckInterval Name = "load-balancer.hetzner.cloud/health-check-interval"

	// LBSvcHealthCheckTimeout specifies the timeout of a single health check
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
)

// LBToService sets the relevant annotations on svc to their respective values
// from lb.
func LBToService(svc *v1.Service, lb *hcloud.LoadBalancer) error {
	const op = "annotation/LBToService"

	sa := &serviceAnnotator{Svc: svc}

	sa.Annotate(LBID, lb.ID)
	sa.Annotate(LBName, lb.Name)
	sa.Annotate(LBType, lb.LoadBalancerType.Name)
	sa.Annotate(LBAlgorithmType, lb.Algorithm.Type)
	sa.Annotate(LBLocation, lb.Location.Name)
	sa.Annotate(LBNetworkZone, lb.Location.NetworkZone)
	sa.Annotate(LBPublicIPv4, lb.PublicNet.IPv4.IP)
	sa.Annotate(LBPublicIPv6, lb.PublicNet.IPv6.IP)

	for _, hclbService := range lb.Services {
		var found bool

		// Find the HC Load Balancer service that matches our K8S service by
		// comparing the port numbers.
		for _, p := range svc.Spec.Ports {
			if hclbService.ListenPort == int(p.Port) {
				found = true
				break
			}
		}

		// This hclbService does not match our K8S service. Continue with the
		// next one.
		if !found {
			continue
		}

		// Once we found a matching service we copy its annotations.
		sa.Annotate(LBSvcProtocol, hclbService.Protocol)
		sa.Annotate(LBSvcProxyProtocol, hclbService.Proxyprotocol)

		if isHTTP(hclbService) || isHTTPS(hclbService) {
			sa.Annotate(LBSvcHTTPCookieName, hclbService.HTTP.CookieName)
			sa.Annotate(LBSvcHTTPCookieLifetime, hclbService.HTTP.CookieLifetime)
		}

		if isHTTPS(hclbService) {
			sa.Annotate(LBSvcRedirectHTTP, hclbService.HTTP.RedirectHTTP)
			sa.Annotate(LBSvcHTTPCertificates, hclbService.HTTP.Certificates)
		}

		sa.Annotate(LBSvcHealthCheckProtocol, hclbService.HealthCheck.Protocol)
		sa.Annotate(LBSvcHealthCheckPort, hclbService.HealthCheck.Port)
		sa.Annotate(LBSvcHealthCheckInterval, hclbService.HealthCheck.Interval)
		sa.Annotate(LBSvcHealthCheckTimeout, hclbService.HealthCheck.Timeout)
		sa.Annotate(LBSvcHealthCheckRetries, hclbService.HealthCheck.Retries)

		if isHTTPHealthCheck(hclbService) || isHTTPSHealthCheck(hclbService) {
			sa.Annotate(LBSvcHealthCheckHTTPDomain, hclbService.HealthCheck.HTTP.Domain)
			sa.Annotate(LBSvcHealthCheckHTTPPath, hclbService.HealthCheck.HTTP.Path)
			sa.Annotate(LBSvcHealthCheckHTTPStatusCodes, hclbService.HealthCheck.HTTP.StatusCodes)
		}
		if isHTTPSHealthCheck(hclbService) {
			sa.Annotate(LBSvcHealthCheckHTTPValidateCertificate, hclbService.HealthCheck.HTTP.TLS)
		}

		// At most one service matches and we've already found it. There
		// is no need to bother with the remaining services.
		break
	}

	if sa.Err != nil {
		return fmt.Errorf("%s: %w", op, sa.Err)
	}
	return nil
}

func isHTTP(s hcloud.LoadBalancerService) bool {
	return s.Protocol == hcloud.LoadBalancerServiceProtocolHTTP
}

func isHTTPHealthCheck(s hcloud.LoadBalancerService) bool {
	return s.HealthCheck.HTTP != nil && s.HealthCheck.Protocol == hcloud.LoadBalancerServiceProtocolHTTP
}

func isHTTPS(s hcloud.LoadBalancerService) bool {
	return s.Protocol == hcloud.LoadBalancerServiceProtocolHTTPS
}

func isHTTPSHealthCheck(s hcloud.LoadBalancerService) bool {
	return s.HealthCheck.HTTP != nil && s.HealthCheck.Protocol == hcloud.LoadBalancerServiceProtocolHTTPS
}
