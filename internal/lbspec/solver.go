package lbspec

import (
	"errors"
	"fmt"
	"net"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// Name returns the effective Load Balancer name for svc: the requested name if
// the annotation is set, otherwise the name derived from the Service UID.
func Name(svc *corev1.Service) string {
	if v, err := annotation.LBName.FromService(svc); err == nil && v != "" {
		return v
	}
	return cloudprovider.DefaultLoadBalancerName(svc)
}

// Resolve builds the desired state from the Service annotations and the
// cluster-wide Load Balancer configuration. Annotations take precedence over
// the configuration, which takes precedence over the built-in defaults.
//
// All invalid annotations are reported together in the returned error. The
// returned Spec is still fully populated, with the offending settings left at
// their fallback, so callers must not use it when err is non-nil.
func Resolve(recorder record.EventRecorder, svc *corev1.Service, cfg config.LoadBalancerConfiguration) (Spec, error) {
	var errs []error
	var spec Spec

	spec.Name = resolve(&errs, svc, annotation.LBName, "")
	if spec.Name == "" {
		spec.Name = cloudprovider.DefaultLoadBalancerName(svc)
		spec.NameUnset = true
	}

	spec.Hostname = resolve(&errs, svc, annotation.LBHostname, "")
	spec.NodeSelector = resolveNodeSelector(&errs, svc)
	spec.Labels = map[string]string{
		LabelServiceUID: string(svc.ObjectMeta.UID),
	}

	spec.Type = resolve(&errs, svc, annotation.LBType, cfg.Type)
	if spec.Type == "" {
		spec.Type = DefaultType
		spec.TypeUnset = true
	}

	spec.Location = resolve(&errs, svc, annotation.LBLocation, cfg.Location)
	spec.NetworkZone = hcloud.NetworkZone(resolve(&errs, svc, annotation.LBNetworkZone, cfg.NetworkZone))
	if spec.Location != "" {
		spec.NetworkZone = ""
	}

	spec.Algorithm = resolve(&errs, svc, annotation.LBAlgorithmType, cfg.AlgorithmType)
	spec.PublicInterface = negate(resolvePtr(&errs, svc, annotation.LBDisablePublicNetwork, cfg.DisablePublicNetwork))
	spec.IPv4RDNS = resolvePtr(&errs, svc, annotation.LBPublicIPv4RDNS, nil)
	spec.IPv6RDNS = resolvePtr(&errs, svc, annotation.LBPublicIPv6RDNS, nil)
	spec.PrivateIPv4 = resolve(&errs, svc, annotation.LBPrivateIPv4, nil)
	spec.PrivateSubnetIPRange = resolvePrivateSubnetIPRange(&errs, svc, cfg)
	spec.UsePrivateIP = resolve(&errs, svc, annotation.LBUsePrivateIP, cfg.PrivateIPEnabled)
	spec.PrivateIngress = !resolve(&errs, svc, annotation.LBDisablePrivateIngress, !cfg.PrivateIngressEnabled)
	spec.IPv6 = !resolve(&errs, svc, annotation.LBIPv6Disabled, !cfg.IPv6Enabled)
	spec.ManagedCertificate = resolveManagedCertificate(&errs, svc)
	spec.Service = resolveService(&errs, svc, cfg, spec.ManagedCertificate != nil)

	if len(errs) == 0 {
		return spec, nil
	}

	for _, err := range errs {
		klog.ErrorS(
			err,
			"invalid Load Balancer annotation",
			"service",
			klog.KObj(svc),
		)
		recorder.Eventf(
			svc,
			corev1.EventTypeWarning,
			"InvalidLoadBalancerAnnotation",
			"invalid Load Balancer annotation %v",
			err,
		)
	}

	return spec, fmt.Errorf("%d Load Balancer annotation(s) are invalid", len(errs))
}

func resolveNodeSelector(errs *[]error, svc *corev1.Service) labels.Selector {
	v := resolvePtr(errs, svc, annotation.LBNodeSelector, nil)
	if v == nil {
		return labels.Everything()
	}

	selector, err := labels.Parse(*v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: unable to parse the node-selector annotation: %w",
			annotation.LBNodeSelector, err))
		return labels.Everything()
	}

	return selector
}

func resolvePrivateSubnetIPRange(errs *[]error, svc *corev1.Service, cfg config.LoadBalancerConfiguration) *net.IPNet {
	value := resolvePtr(errs, svc, annotation.PrivateSubnetIPRange, nil)
	if value == nil {
		if cfg.PrivateSubnetIPRange == "" {
			return nil
		}
		value = &cfg.PrivateSubnetIPRange
	}

	_, subnet, err := net.ParseCIDR(*value)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("invalid private subnet IP range %q: %w", *value, err))
		return nil
	}

	return subnet
}

func resolveManagedCertificate(errs *[]error, svc *corev1.Service) *ManagedCertificate {
	// Compared as a raw string: only the exact value selects managed
	// certificates.
	if typ, err := annotation.LBSvcHTTPCertificateType.FromService(svc); err != nil || typ != string(hcloud.CertificateTypeManaged) {
		return nil
	}

	cert := ManagedCertificate{
		Name:   fmt.Sprintf("ccm-managed-certificate-%s", svc.ObjectMeta.UID),
		Labels: map[string]string{LabelServiceUID: string(svc.ObjectMeta.UID)},
	}
	if v, err := annotation.LBSvcHTTPManagedCertificateName.FromService(svc); err == nil && v != "" {
		cert.Name = v
	}

	domains, err := annotation.LBSvcHTTPManagedCertificateDomains.FromService(svc)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: no domains for managed certificate",
			annotation.LBSvcHTTPManagedCertificateDomains))
		return nil
	}
	cert.Domains = domains

	// The error is ignored on purpose: we are only interested in whether the
	// annotation is set and parses as a truthy boolean. Anything else tells us
	// not to use ACME staging.
	cert.UseACMEStaging, _ = annotation.LBSvcHTTPManagedCertificateUseACMEStaging.FromService(svc)

	return &cert
}

func resolveService(
	errs *[]error, svc *corev1.Service, cfg config.LoadBalancerConfiguration, hasManagedCertificate bool,
) ServiceSpec {
	var spec ServiceSpec

	spec.Protocol = resolve(errs, svc, annotation.LBSvcProtocol, hcloud.LoadBalancerServiceProtocolTCP)
	spec.ProxyProtocol = resolvePtr(errs, svc, annotation.LBSvcProxyProtocol, cfg.ProxyProtocolEnabled)

	var http HTTPSpec
	var httpConfigured bool

	if v := resolvePtr(errs, svc, annotation.LBSvcHTTPCookieName, nil); v != nil {
		http.CookieName = v
		httpConfigured = true
	}
	if v := resolvePtr(errs, svc, annotation.LBSvcHTTPCookieLifetime, nil); v != nil {
		http.CookieLifetime = v
		httpConfigured = true
	}
	if v := resolvePtr(errs, svc, annotation.LBSvcHTTPTimeoutIdle, nil); v != nil {
		http.TimeoutIdle = v
		httpConfigured = true
	}
	if v := resolvePtr(errs, svc, annotation.LBSvcRedirectHTTP, nil); v != nil {
		http.RedirectHTTP = v
		httpConfigured = true
	}
	if v := resolvePtr(errs, svc, annotation.LBSvcHTTPStickySessions, nil); v != nil {
		http.StickySessions = v
		httpConfigured = true
	}

	if hasManagedCertificate {
		// The managed certificate replaces the uploaded certificate list, so
		// that annotation is not read at all. The caller fills in the
		// certificate it looked up by label.
		httpConfigured = true
	} else {
		certs := resolve(errs, svc, annotation.LBSvcHTTPCertificates, nil)
		if len(certs) > 0 {
			http.Certificates = certs
			httpConfigured = true
		}
	}

	if httpConfigured {
		spec.HTTP = &http
	}
	spec.HealthCheck = resolveHealthCheck(errs, svc, cfg, spec.Protocol)

	return spec
}

func resolveHealthCheck(
	errs *[]error,
	svc *corev1.Service,
	cfg config.LoadBalancerConfiguration,
	serviceProtocol hcloud.LoadBalancerServiceProtocol,
) *HealthCheckSpec {
	// Without an explicit health check protocol the service protocol is used,
	// but that alone does not configure a health check.
	check := HealthCheckSpec{Protocol: serviceProtocol}
	var configured bool

	if v := resolvePtr(errs, svc, annotation.LBSvcHealthCheckProtocol, nil); v != nil {
		check.Protocol = *v
		configured = true
	}
	if v := resolvePtr(errs, svc, annotation.LBSvcHealthCheckPort, nil); v != nil {
		check.Port = v
		configured = true
	}
	// The cluster-wide defaults only apply when set to a non-zero value.
	if v := resolvePtr(errs, svc, annotation.LBSvcHealthCheckInterval, nonZero(cfg.HealthCheckInterval)); v != nil {
		check.Interval = v
		configured = true
	}
	if v := resolvePtr(errs, svc, annotation.LBSvcHealthCheckTimeout, nonZero(cfg.HealthCheckTimeout)); v != nil {
		check.Timeout = v
		configured = true
	}
	if v := resolvePtr(errs, svc, annotation.LBSvcHealthCheckRetries, nonZero(cfg.HealthCheckRetries)); v != nil {
		check.Retries = v
		configured = true
	}

	// A TCP health check has no HTTP options.
	if check.Protocol != hcloud.LoadBalancerServiceProtocolTCP {
		check.HTTP.Domain = resolvePtr(errs, svc, annotation.LBSvcHealthCheckHTTPDomain, nil)
		check.HTTP.Path = resolvePtr(errs, svc, annotation.LBSvcHealthCheckHTTPPath, nil)
		check.HTTP.TLS = resolvePtr(errs, svc, annotation.LBSvcHealthCheckHTTPValidateCertificate, nil)
		check.HTTP.StatusCodes = resolve(errs, svc, annotation.LBSvcHealthCheckHTTPStatusCodes, nil)
	}

	if !configured {
		return nil
	}
	return &check
}

func resolve[T any](errs *[]error, svc *corev1.Service, a annotation.Annotation[T], fallback T) T {
	v, err := a.FromService(svc)
	switch {
	case err == nil:
		return v
	case errors.Is(err, annotation.ErrNotSet):
		return fallback
	default:
		*errs = append(*errs, err)
		return fallback
	}
}

func resolvePtr[T any](errs *[]error, svc *corev1.Service, a annotation.Annotation[T], fallback *T) *T {
	v, err := a.FromService(svc)
	switch {
	case err == nil:
		return &v
	case errors.Is(err, annotation.ErrNotSet):
		return fallback
	default:
		*errs = append(*errs, err)
		return fallback
	}
}

func nonZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

func negate(v *bool) *bool {
	if v == nil {
		return nil
	}
	return new(!*v)
}
