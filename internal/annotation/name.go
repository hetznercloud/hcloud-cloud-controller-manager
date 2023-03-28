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
	"github.com/hetznercloud/hcloud-go/hcloud"
)

// ErrNotSet signals that an annotation was not set.
var ErrNotSet = errors.New("not set")

// Name defines the name of a K8S annotation.
type Name string

// AnnotateService adds the value v as an annotation with s.Name to svc.
//
// AnnotateService returns an error if converting v to a string fails.
func (s Name) AnnotateService(svc *corev1.Service, v interface{}) error {
	const op = "annotation/Name.AnnotateService"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	if svc.ObjectMeta.Annotations == nil {
		svc.ObjectMeta.Annotations = make(map[string]string)
	}
	k := string(s)
	switch vt := v.(type) {
	case bool:
		svc.ObjectMeta.Annotations[k] = strconv.FormatBool(vt)
	case int:
		svc.ObjectMeta.Annotations[k] = strconv.Itoa(vt)
	case string:
		svc.ObjectMeta.Annotations[k] = vt
	case []string:
		svc.ObjectMeta.Annotations[k] = strings.Join(vt, ",")
	case hcloud.CertificateType:
		svc.ObjectMeta.Annotations[k] = string(vt)
	case []*hcloud.Certificate:
		idsOrNames := make([]string, len(vt))
		for i, c := range vt {
			if c.ID == 0 && c.Name != "" {
				idsOrNames[i] = c.Name
				continue
			}
			idsOrNames[i] = strconv.Itoa(c.ID)
		}
		svc.ObjectMeta.Annotations[k] = strings.Join(idsOrNames, ",")
	case hcloud.NetworkZone:
		svc.ObjectMeta.Annotations[k] = string(vt)
	case hcloud.LoadBalancerAlgorithmType:
		svc.ObjectMeta.Annotations[k] = string(vt)
	case hcloud.LoadBalancerServiceProtocol:
		svc.ObjectMeta.Annotations[k] = string(vt)
	case fmt.Stringer:
		svc.ObjectMeta.Annotations[k] = vt.String()
	default:
		return fmt.Errorf("%s: %v: unsupported type: %T", op, s, v)
	}
	return nil
}

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
			id, err := strconv.Atoi(s)
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

type serviceAnnotator struct {
	Svc *corev1.Service
	Err error
}

func (sa *serviceAnnotator) Annotate(n Name, v interface{}) {
	if sa.Err != nil {
		return
	}
	sa.Err = n.AnnotateService(sa.Svc, v)
}
