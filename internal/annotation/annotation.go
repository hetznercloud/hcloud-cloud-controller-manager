// Package annotation defines the Kubernetes annotations that configure the
// resources managed by the cloud controller manager.
//
// Every annotation is declared with the type of its value, so reading one
// yields that type and nothing else:
//
//	const LBUsePrivateIP Bool = "load-balancer.hetzner.cloud/use-private-ip"
//
//	usePrivateIP, err := LBUsePrivateIP.FromService(svc)
//
// Annotations are declared as constants of a named string type. The reference
// documentation in docs/reference is generated from those declarations by
// tools/doc_generation.go, which expects constants with a string literal.
package annotation

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var ErrNotSet = errors.New("not set")

type Annotation[T any] interface {
	FromService(svc *corev1.Service) (T, error)
}

type (
	String        string
	Bool          string
	Int           string
	Duration      string // e.g. "30s" or "1h"
	Strings       string // comma separated list of strings
	IP            string
	Protocol      string // Load Balancer service protocol
	AlgorithmType string // Load Balancer algorithm type
	// Certificates is an annotation holding a comma separated list of Certificates,
	// referenced either by ID or by name.
	Certificates string
)

func (a String) FromService(svc *corev1.Service) (string, error) {
	return value(string(a), svc)
}

func (a Bool) FromService(svc *corev1.Service) (bool, error) {
	return parse(string(a), svc, strconv.ParseBool)
}

func (a Int) FromService(svc *corev1.Service) (int, error) {
	return parse(string(a), svc, strconv.Atoi)
}

func (a Duration) FromService(svc *corev1.Service) (time.Duration, error) {
	return parse(string(a), svc, time.ParseDuration)
}

func (a Strings) FromService(svc *corev1.Service) ([]string, error) {
	return parse(string(a), svc, func(v string) ([]string, error) {
		return strings.Split(v, ","), nil
	})
}

func (a IP) FromService(svc *corev1.Service) (net.IP, error) {
	return parse(string(a), svc, parseIP)
}

func (a Protocol) FromService(svc *corev1.Service) (hcloud.LoadBalancerServiceProtocol, error) {
	return parse(string(a), svc, parseServiceProtocol)
}

func (a AlgorithmType) FromService(svc *corev1.Service) (hcloud.LoadBalancerAlgorithmType, error) {
	return parse(string(a), svc, parseAlgorithmType)
}

func (a Certificates) FromService(svc *corev1.Service) ([]*hcloud.Certificate, error) {
	return parse(string(a), svc, parseCertificates)
}

// value returns the raw value of the annotation with name from svc.
func value(name string, svc *corev1.Service) (string, error) {
	v, ok := svc.Annotations[name]
	if !ok {
		return "", fmt.Errorf("%q: %w", name, ErrNotSet)
	}
	return v, nil
}

func parse[T any](name string, svc *corev1.Service, convert func(string) (T, error)) (T, error) {
	var zero T

	v, err := value(name, svc)
	if err != nil {
		return zero, err
	}

	converted, err := convert(v)
	if err != nil {
		return zero, fmt.Errorf("%q: %w", name, err)
	}

	return converted, nil
}

func parseIP(v string) (net.IP, error) {
	ip := net.ParseIP(v)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip address: %s", v)
	}
	return ip, nil
}

func parseAlgorithmType(v string) (hcloud.LoadBalancerAlgorithmType, error) {
	algorithm := hcloud.LoadBalancerAlgorithmType(strings.ToLower(v))

	switch algorithm {
	case hcloud.LoadBalancerAlgorithmTypeLeastConnections,
		hcloud.LoadBalancerAlgorithmTypeRoundRobin:
		return algorithm, nil
	default:
		return "", fmt.Errorf("invalid algorithm type: %s", v)
	}
}

func parseServiceProtocol(v string) (hcloud.LoadBalancerServiceProtocol, error) {
	protocol := hcloud.LoadBalancerServiceProtocol(strings.ToLower(v))

	switch protocol {
	case hcloud.LoadBalancerServiceProtocolTCP,
		hcloud.LoadBalancerServiceProtocolHTTP,
		hcloud.LoadBalancerServiceProtocolHTTPS:
		return protocol, nil
	default:
		return "", fmt.Errorf("invalid protocol: %s", v)
	}
}

func parseCertificates(v string) ([]*hcloud.Certificate, error) {
	values := strings.Split(v, ",")
	certificates := make([]*hcloud.Certificate, len(values))

	for i, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			// If we could not parse the string as an integer we assume it is a
			// name, not an id.
			certificates[i] = &hcloud.Certificate{Name: value}
			continue
		}
		certificates[i] = &hcloud.Certificate{ID: id}
	}

	return certificates, nil
}
