package lbspec

import (
	"maps"
	"net"
	"net/netip"

	corev1 "k8s.io/api/core/v1"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// labelUseStagingCA tells the backend to issue managed certificates through the
// Let's Encrypt staging environment.
const labelUseStagingCA = "HC-Use-Staging-CA"

func (s Spec) CreateOpts(lbType *hcloud.LoadBalancerType) hcloud.LoadBalancerCreateOpts {
	opts := hcloud.LoadBalancerCreateOpts{
		Name:             s.Name,
		LoadBalancerType: lbType,
		Labels:           s.Labels,
		NetworkZone:      s.NetworkZone,
		PublicInterface:  s.PublicInterface,
	}

	if s.Location != "" {
		opts.Location = &hcloud.Location{Name: s.Location}
	}
	if s.Algorithm != "" {
		opts.Algorithm = &hcloud.LoadBalancerAlgorithm{Type: s.Algorithm}
	}

	return opts
}

func (s Spec) UpdateOpts(lb *hcloud.LoadBalancer) (hcloud.LoadBalancerUpdateOpts, bool) {
	var (
		opts   hcloud.LoadBalancerUpdateOpts
		update bool
	)

	if !hasLabels(lb.Labels, s.Labels) {
		labels := make(map[string]string, len(lb.Labels)+len(s.Labels))
		maps.Copy(labels, lb.Labels)
		maps.Copy(labels, s.Labels)

		opts.Labels = labels
		update = true
	}

	if !s.NameUnset && s.Name != lb.Name {
		opts.Name = s.Name
		update = true
	}

	return opts, update
}

func (s Spec) ChangeAlgorithmOpts() hcloud.LoadBalancerChangeAlgorithmOpts {
	return hcloud.LoadBalancerChangeAlgorithmOpts{Type: s.Algorithm}
}

func (s Spec) AttachToNetworkOpts(network *hcloud.Network) hcloud.LoadBalancerAttachToNetworkOpts {
	return hcloud.LoadBalancerAttachToNetworkOpts{
		Network: network,
		IP:      toIP(s.PrivateIPv4),
		IPRange: toIPNet(s.PrivateSubnetIPRange),
	}
}

func (s Spec) AddServerTargetOpts(serverID int64) hcloud.LoadBalancerAddServerTargetOpts {
	return hcloud.LoadBalancerAddServerTargetOpts{
		Server:       &hcloud.Server{ID: serverID},
		UsePrivateIP: new(s.UsePrivateIP),
	}
}

func (c OwnedCertificate) CreateOpts() hcloud.CertificateCreateOpts {
	labels := c.Labels
	if c.UseACMEStaging {
		// A copy, so that requesting staging does not leave the label behind
		// in the spec.
		labels = make(map[string]string, len(c.Labels)+1)
		maps.Copy(labels, c.Labels)
		labels[labelUseStagingCA] = "true"
	}

	return hcloud.CertificateCreateOpts{
		Name:        c.Name,
		Type:        hcloud.CertificateTypeManaged,
		DomainNames: c.Domains,
		Labels:      labels,
	}
}

func toIP(addr netip.Addr) net.IP {
	if !addr.IsValid() {
		return nil
	}
	return addr.AsSlice()
}

func toIPNet(prefix netip.Prefix) *net.IPNet {
	if !prefix.IsValid() {
		return nil
	}
	return &net.IPNet{
		IP:   prefix.Addr().AsSlice(),
		Mask: net.CIDRMask(prefix.Bits(), prefix.Addr().BitLen()),
	}
}

// hasLabels reports whether actual carries every label in required.
func hasLabels(actual, required map[string]string) bool {
	for k, v := range required {
		if actual[k] != v {
			return false
		}
	}
	return true
}

func (s ServiceSpec) AddServiceOpts(
	port corev1.ServicePort, certificates []*hcloud.Certificate,
) hcloud.LoadBalancerAddServiceOpts {
	destinationPort := int(port.NodePort)

	opts := hcloud.LoadBalancerAddServiceOpts{
		ListenPort:      new(int(port.Port)),
		DestinationPort: new(destinationPort),
		Protocol:        s.Protocol,
		Proxyprotocol:   s.ProxyProtocol,
		HealthCheck:     s.addHealthCheckOpts(destinationPort),
	}

	if s.HTTP != nil {
		opts.HTTP = &hcloud.LoadBalancerAddServiceOptsHTTP{
			CookieName:     s.HTTP.CookieName,
			CookieLifetime: s.HTTP.CookieLifetime,
			Certificates:   certificates,
			RedirectHTTP:   s.HTTP.RedirectHTTP,
			StickySessions: s.HTTP.StickySessions,
			TimeoutIdle:    s.HTTP.TimeoutIdle,
		}
	}

	return opts
}

func (s ServiceSpec) addHealthCheckOpts(destinationPort int) *hcloud.LoadBalancerAddServiceOptsHealthCheck {
	if s.HealthCheck == nil {
		return &hcloud.LoadBalancerAddServiceOptsHealthCheck{
			Protocol: hcloud.LoadBalancerServiceProtocolTCP,
			Port:     new(destinationPort),
		}
	}

	port := s.HealthCheck.Port
	if port == nil {
		port = new(destinationPort)
	}

	opts := &hcloud.LoadBalancerAddServiceOptsHealthCheck{
		Protocol: s.HealthCheck.Protocol,
		Interval: s.HealthCheck.Interval,
		Port:     port,
		Retries:  s.HealthCheck.Retries,
		Timeout:  s.HealthCheck.Timeout,
	}

	if s.HealthCheck.Protocol == hcloud.LoadBalancerServiceProtocolHTTP ||
		s.HealthCheck.Protocol == hcloud.LoadBalancerServiceProtocolHTTPS {
		opts.HTTP = &hcloud.LoadBalancerAddServiceOptsHealthCheckHTTP{
			Domain:      s.HealthCheck.HTTP.Domain,
			Path:        s.HealthCheck.HTTP.Path,
			StatusCodes: s.HealthCheck.HTTP.StatusCodes,
			TLS:         s.HealthCheck.HTTP.TLS,
		}
	}

	return opts
}

func (s ServiceSpec) UpdateServiceOpts(
	port corev1.ServicePort, certificates []*hcloud.Certificate,
) hcloud.LoadBalancerUpdateServiceOpts {
	add := s.AddServiceOpts(port, certificates)

	opts := hcloud.LoadBalancerUpdateServiceOpts{
		DestinationPort: add.DestinationPort,
		Protocol:        add.Protocol,
		Proxyprotocol:   add.Proxyprotocol,
	}

	if add.HTTP != nil {
		opts.HTTP = &hcloud.LoadBalancerUpdateServiceOptsHTTP{
			CookieName:     add.HTTP.CookieName,
			CookieLifetime: add.HTTP.CookieLifetime,
			Certificates:   add.HTTP.Certificates,
			RedirectHTTP:   add.HTTP.RedirectHTTP,
			StickySessions: add.HTTP.StickySessions,
			TimeoutIdle:    add.HTTP.TimeoutIdle,
		}
	}

	if add.HealthCheck != nil {
		opts.HealthCheck = &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
			Protocol: add.HealthCheck.Protocol,
			Interval: add.HealthCheck.Interval,
			Port:     add.HealthCheck.Port,
			Retries:  add.HealthCheck.Retries,
			Timeout:  add.HealthCheck.Timeout,
		}
		if add.HealthCheck.HTTP != nil {
			opts.HealthCheck.HTTP = &hcloud.LoadBalancerUpdateServiceOptsHealthCheckHTTP{
				Domain:      add.HealthCheck.HTTP.Domain,
				Path:        add.HealthCheck.HTTP.Path,
				StatusCodes: add.HealthCheck.HTTP.StatusCodes,
				TLS:         add.HealthCheck.HTTP.TLS,
			}
		}
	}

	return opts
}
