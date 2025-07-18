package annotation

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// ErrNotSet signals that an annotation was not set.
var ErrNotSet = errors.New("not set")

// Name defines the name of a K8S annotation.
type Name string

// StringFromService retrieves the value belonging to the annotation from svc.
//
// If svc has no value for the annotation the second return value is false.
func (s Name) StringFromService(svc *corev1.Service) (string, bool) {
	if svc.Annotations == nil {
		return "", false
	}
	v, ok := svc.Annotations[string(s)]
	return v, ok
}

// StringsFromService retrieves the []string value belonging to the annotation
// from svc.
//
// StringsFromService returns ErrNotSet annotation was not set.
func (s Name) StringsFromService(svc *corev1.Service) ([]string, error) {
	const op = "annotation/Name.StringsFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var ss []string

	err := s.applyToValue(op, svc, func(v string) error {
		ss = strings.Split(v, ",")
		return nil
	})

	return ss, err
}

// BoolFromService retrieves the boolean value belonging to the annotation from
// svc.
//
// BoolFromService returns an error if the value could not be converted to a
// boolean, or the annotation was not set. In the case of a missing value, the
// error wraps ErrNotSet.
func (s Name) BoolFromService(svc *corev1.Service) (bool, error) {
	const op = "annotation/Name.BoolFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	v, ok := s.StringFromService(svc)
	if !ok {
		return false, fmt.Errorf("%s: %v: %w", op, s, ErrNotSet)
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %v: %w", op, s, err)
	}
	return b, nil
}

// IntFromService retrieves the int value belonging to the annotation from svc.
//
// IntFromService returns an error if the value could not be converted to an
// int, or the annotation was not set. In the case of a missing value, the
// error wraps ErrNotSet.
func (s Name) IntFromService(svc *corev1.Service) (int, error) {
	const op = "annotation/Name.IntFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	v, ok := s.StringFromService(svc)
	if !ok {
		return 0, fmt.Errorf("%s: %v: %w", op, s, ErrNotSet)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %v: %w", op, s, err)
	}
	return i, nil
}

// IntsFromService retrieves the []int value belonging to the annotation from
// svc.
//
// IntsFromService returns an error if the value could not be converted to a
// []int, or the annotation was not set. In the case of a missing value, the
// error wraps ErrNotSet.
func (s Name) IntsFromService(svc *corev1.Service) ([]int, error) {
	const op = "annotation/Name.IntsFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var is []int

	err := s.applyToValue(op, svc, func(v string) error {
		ss := strings.Split(v, ",")
		is = make([]int, len(ss))

		for i, s := range ss {
			iv, err := strconv.Atoi(s)
			if err != nil {
				return err
			}
			is[i] = iv
		}
		return nil
	})

	return is, err
}

// IPFromService retrieves the net.IP value belonging to the annotation from
// svc.
//
// IPFromService returns an error if the value could not be converted to a
// net.IP, or the annotation was not set. In the case of a missing value, the
// error wraps ErrNotSet.
func (s Name) IPFromService(svc *corev1.Service) (net.IP, error) {
	const op = "annotation/Name.IPFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var ip net.IP

	err := s.applyToValue(op, svc, func(v string) error {
		ip = net.ParseIP(v)
		if ip == nil {
			return fmt.Errorf("invalid ip address: %s", v)
		}
		return nil
	})

	return ip, err
}

// DurationFromService retrieves the time.Duration value belonging to the
// annotation from svc.
//
// DurationFromService returns an error if the value could not be converted to
// a time.Duration, or the annotation was not set. In the case of a missing
// value, the error wraps ErrNotSet.
func (s Name) DurationFromService(svc *corev1.Service) (time.Duration, error) {
	const op = "annotation/Name.DurationFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var d time.Duration

	err := s.applyToValue(op, svc, func(v string) error {
		var err error

		d, err = time.ParseDuration(v)
		return err
	})

	return d, err
}

// LBSvcProtocolFromService retrieves the hcloud.LoadBalancerServiceProtocol
// value belonging to the annotation from svc.
//
// LBSvcProtocolFromService returns an error if the value could not be
// converted to a hcloud.LoadBalancerServiceProtocol, or the annotation was not
// set. In the case of a missing value, the error wraps ErrNotSet.
func (s Name) LBSvcProtocolFromService(svc *corev1.Service) (hcloud.LoadBalancerServiceProtocol, error) {
	const op = "annotation/Name.LBSvcProtocolFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var p hcloud.LoadBalancerServiceProtocol

	err := s.applyToValue(op, svc, func(v string) error {
		var err error

		p, err = validateServiceProtocol(v)
		return err
	})

	return p, err
}

// LBAlgorithmTypeFromService retrieves the hcloud.LoadBalancerAlgorithmType
// value belonging to the annotation from svc.
//
// LBAlgorithmTypeFromService returns an error if the value could not be
// converted to a hcloud.LoadBalancerAlgorithmType, or the annotation was not
// set. In the case of a missing value, the error wraps ErrNotSet.
func (s Name) LBAlgorithmTypeFromService(svc *corev1.Service) (hcloud.LoadBalancerAlgorithmType, error) {
	const op = "annotation/Name.LBAlgorithmTypeFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var alg hcloud.LoadBalancerAlgorithmType

	err := s.applyToValue(op, svc, func(v string) error {
		var err error

		alg, err = validateAlgorithmType(v)
		return err
	})

	return alg, err
}

// NetworkZoneFromService retrieves the hcloud.NetworkZone value belonging to
// the annotation from svc.
//
// NetworkZoneFromService returns ErrNotSet if the annotation was not set.
func (s Name) NetworkZoneFromService(svc *corev1.Service) (hcloud.NetworkZone, error) {
	const op = "annotation/Name.NetworkZoneFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var nz hcloud.NetworkZone

	err := s.applyToValue(op, svc, func(v string) error {
		nz = hcloud.NetworkZone(v)
		return nil
	})

	return nz, err
}

// CertificatesFromService retrieves the []*hcloud.Certificate value belonging
// to the annotation from svc.
//
// CertificatesFromService returns an error if the value could not be converted
// to a []*hcloud.Certificate, or the annotation was not set. In the case of a
// missing value, the error wraps ErrNotSet.
func (s Name) CertificatesFromService(svc *corev1.Service) ([]*hcloud.Certificate, error) {
	const op = "annotation/Name.CertificatesFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var cs []*hcloud.Certificate

	err := s.applyToValue(op, svc, func(v string) error {
		ss := strings.Split(v, ",")
		cs = make([]*hcloud.Certificate, len(ss))

		for i, s := range ss {
			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				// If we could not parse the string as an integer we assume it
				// is a name not an id.
				cs[i] = &hcloud.Certificate{Name: s}
				continue
			}
			cs[i] = &hcloud.Certificate{ID: id}
		}

		return nil
	})

	return cs, err
}

// ProtocolPortsFromService retrieves the protocol configuration per port from svc.
// The annotation format is "port:protocol,port:protocol" (e.g. "80:http,443:https,9000:tcp")
// 
// Returns a map of port -> protocol. Returns an empty map if the annotation was not set.
func (s Name) ProtocolPortsFromService(svc *corev1.Service) (map[int]hcloud.LoadBalancerServiceProtocol, error) {
	const op = "annotation/Name.ProtocolPortsFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	result := make(map[int]hcloud.LoadBalancerServiceProtocol)

	v, ok := s.StringFromService(svc)
	if !ok {
		return result, nil // Return empty map if not set
	}

	if strings.TrimSpace(v) == "" {
		return result, nil
	}

	pairs := strings.Split(v, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s: invalid format for port:protocol pair: %s", op, pair)
		}

		port, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("%s: invalid port number: %s", op, parts[0])
		}

		protocol, err := validateServiceProtocol(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		result[port] = protocol
	}

	return result, nil
}

// CertificatePortsFromService retrieves the certificate configuration per port from svc.
// The annotation format is "port:cert1,cert2;port:cert3,cert4" (e.g. "443:cert1,cert2;8443:cert3")
//
// Returns a map of port -> certificates. Returns an empty map if the annotation was not set.
func (s Name) CertificatePortsFromService(svc *corev1.Service) (map[int][]*hcloud.Certificate, error) {
	const op = "annotation/Name.CertificatePortsFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	result := make(map[int][]*hcloud.Certificate)

	v, ok := s.StringFromService(svc)
	if !ok {
		return result, nil // Return empty map if not set
	}

	if strings.TrimSpace(v) == "" {
		return result, nil
	}

	// Split by semicolon to get port configurations
	portConfigs := strings.Split(v, ";")
	for _, portConfig := range portConfigs {
		portConfig = strings.TrimSpace(portConfig)
		if portConfig == "" {
			continue
		}

		// Split by colon to get port and certificates
		parts := strings.Split(portConfig, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s: invalid format for port:certificates pair: %s", op, portConfig)
		}

		port, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("%s: invalid port number: %s", op, parts[0])
		}

		// Parse certificates (same logic as CertificatesFromService)
		certStrings := strings.Split(strings.TrimSpace(parts[1]), ",")
		certificates := make([]*hcloud.Certificate, len(certStrings))

		for i, certString := range certStrings {
			certString = strings.TrimSpace(certString)
			if certString == "" {
				return nil, fmt.Errorf("%s: empty certificate reference", op)
			}

			id, err := strconv.ParseInt(certString, 10, 64)
			if err != nil {
				// If we could not parse the string as an integer we assume it
				// is a name not an id.
				certificates[i] = &hcloud.Certificate{Name: certString}
			} else {
				certificates[i] = &hcloud.Certificate{ID: id}
			}
		}

		result[port] = certificates
	}

	return result, nil
}

// CertificateTypeFromService retrieves the hcloud.CertificateType value
// belonging to the annotation from svc.
//
// CertificateTypeFromService returns an error if the value could not be
// converted to a hcloud.CertificateType. In the case of a missing value, the
// error wraps ErrNotSet.
func (s Name) CertificateTypeFromService(svc *corev1.Service) (hcloud.CertificateType, error) {
	const op = "annotation/Name.CertificateTypeFromService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	var ct hcloud.CertificateType

	err := s.applyToValue(op, svc, func(v string) error {
		switch strings.ToLower(v) {
		case string(hcloud.CertificateTypeUploaded):
			ct = hcloud.CertificateTypeUploaded
		case string(hcloud.CertificateTypeManaged):
			ct = hcloud.CertificateTypeManaged
		default:
			return fmt.Errorf("%s: unsupported certificate type: %s", op, v)
		}
		return nil
	})

	return ct, err
}

func (s Name) applyToValue(op string, svc *corev1.Service, f func(string) error) error {
	v, ok := s.StringFromService(svc)
	if !ok {
		return fmt.Errorf("%s: %v: %w", op, s, ErrNotSet)
	}
	if err := f(v); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func validateAlgorithmType(algorithmType string) (hcloud.LoadBalancerAlgorithmType, error) {
	const op = "annotation/validateAlgorithmType"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	algorithmType = strings.ToLower(algorithmType) // Lowercase because all our protocols are lowercase
	hcloudAlgorithmType := hcloud.LoadBalancerAlgorithmType(algorithmType)

	switch hcloudAlgorithmType {
	case hcloud.LoadBalancerAlgorithmTypeLeastConnections:
	case hcloud.LoadBalancerAlgorithmTypeRoundRobin:
	default:
		return "", fmt.Errorf("%s: invalid: %s", op, algorithmType)
	}

	return hcloudAlgorithmType, nil
}

func validateServiceProtocol(protocol string) (hcloud.LoadBalancerServiceProtocol, error) {
	const op = "annotation/validateServiceProtocol"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	protocol = strings.ToLower(protocol) // Lowercase because all our protocols are lowercase
	hcloudProtocol := hcloud.LoadBalancerServiceProtocol(protocol)
	switch hcloudProtocol {
	case hcloud.LoadBalancerServiceProtocolTCP:
	case hcloud.LoadBalancerServiceProtocolHTTPS:
	case hcloud.LoadBalancerServiceProtocolHTTP:
		// Valid
		break
	default:
		return "", fmt.Errorf("%s: invalid: %s", op, protocol)
	}
	return hcloudProtocol, nil
}
