package annotation

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// AssertServiceAnnotated asserts that svc has been annotated with all
// annotations in expected.
func AssertServiceAnnotated(t *testing.T, svc *corev1.Service, expected map[Name]interface{}) {
	t.Helper()
	for ek, ev := range expected {
		var (
			actual interface{}
			ok     bool
			err    error
		)
		actual, ok = ek.StringFromService(svc)
		if !ok {
			t.Errorf("not annotated with: %v", ek)
			continue
		}

		switch ev.(type) {
		case bool:
			actual, err = ek.BoolFromService(svc)
		case int:
			actual, err = ek.IntFromService(svc)
		case []string:
			actual, err = ek.StringsFromService(svc)
		case net.IP:
			actual, err = ek.IPFromService(svc)
		case time.Duration:
			actual, err = ek.DurationFromService(svc)
		case []*hcloud.Certificate:
			actual, err = ek.CertificatesFromService(svc)
		case hcloud.LoadBalancerAlgorithmType:
			actual, err = ek.LBAlgorithmTypeFromService(svc)
		case hcloud.LoadBalancerServiceProtocol:
			actual, err = ek.LBSvcProtocolFromService(svc)
		case hcloud.NetworkZone:
			actual, err = ek.NetworkZoneFromService(svc)
		}
		assert.NoError(t, err)
		assert.Equal(t, ev, actual)
	}
}
