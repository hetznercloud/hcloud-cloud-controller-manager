package annotation_test

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const ann annotation.Name = "some/annotation"

func TestName_StringFromService(t *testing.T) {
	tests := []struct {
		name           string
		svcAnnotations map[annotation.Name]string
		ok             bool
		expected       string
	}{
		{
			name:           "value as string",
			svcAnnotations: map[annotation.Name]string{ann: "some value"},
			ok:             true,
			expected:       "some value",
		},
		{
			name: "Service has no annotations",
		},
		{
			name: "value not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var svc corev1.Service
			svc.Annotations = map[string]string{}

			for k, v := range tt.svcAnnotations {
				svc.Annotations[string(k)] = v
			}
			actual, ok := ann.StringFromService(&svc)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestName_StringsFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]string{
				ann: "a,b,c",
			},
			expected: []string{"a", "b", "c"},
		},
		{
			name: "value missing",
			err:  annotation.ErrNotSet,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.StringsFromService(svc)
	})
}

func TestName_BoolFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name:           "value set to true",
			svcAnnotations: map[annotation.Name]string{ann: "true"},
			expected:       true,
		},
		{
			name:           "value set to false",
			svcAnnotations: map[annotation.Name]string{ann: "false"},
			expected:       false,
		},
		{
			name:     "value missing",
			expected: false,
			err:      annotation.ErrNotSet,
		},
		{
			name:           "value invalid",
			svcAnnotations: map[annotation.Name]string{ann: "invalid"},
			expected:       false,
			err:            strconv.ErrSyntax,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.BoolFromService(svc)
	})
}

func TestName_IntFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name:           "value set to 10",
			svcAnnotations: map[annotation.Name]string{ann: "10"},
			expected:       10,
		},
		{
			name:     "value missing",
			expected: 0,
			err:      fmt.Errorf("annotation/Name.IntFromService: %s: %w", ann, annotation.ErrNotSet),
		},
		{
			name:           "value invalid",
			svcAnnotations: map[annotation.Name]string{ann: "invalid"},
			expected:       0,
			err:            strconv.ErrSyntax,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.IntFromService(svc)
	})
}

func TestName_IntsFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]string{
				ann: "5,8",
			},
			expected: []int{5, 8},
		},
		{
			name: "value missing",
			err:  annotation.ErrNotSet,
		},
		{
			name:           "value invalid",
			svcAnnotations: map[annotation.Name]string{ann: "invalid"},
			err:            strconv.ErrSyntax,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.IntsFromService(svc)
	})
}

func TestName_IPFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set to valid IPv4",
			svcAnnotations: map[annotation.Name]string{
				ann: "1.2.3.4",
			},
			expected: net.ParseIP("1.2.3.4"),
		},
		{
			name: "value set to valid IPv6",
			svcAnnotations: map[annotation.Name]string{
				ann: "3c2e:2ef9:a7e9:1a5b:30ba:4912:e3fe:91b2",
			},
			expected: net.ParseIP("3c2e:2ef9:a7e9:1a5b:30ba:4912:e3fe:91b2"),
		},
		{
			name: "value invalid",
			svcAnnotations: map[annotation.Name]string{
				ann: "invalid",
			},
			err: errors.New("annotation/Name.IPFromService: invalid ip address: invalid"),
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.IPFromService(svc)
	})
}

func TestName_DurationFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]string{
				ann: "1h",
			},
			expected: time.Hour,
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
		{
			name: "value invalid",
			svcAnnotations: map[annotation.Name]string{
				ann: "invalid",
			},
			err: errors.New("annotation/Name.DurationFromService: time: invalid duration \"invalid\""),
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.DurationFromService(svc)
	})
}

func TestName_LBSvcProtocolFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]string{
				ann: string(hcloud.LoadBalancerServiceProtocolHTTP),
			},
			expected: hcloud.LoadBalancerServiceProtocolHTTP,
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
		{
			name: "value invalid",
			svcAnnotations: map[annotation.Name]string{
				ann: "invalid",
			},
			err: errors.New("annotation/Name.LBSvcProtocolFromService: annotation/validateServiceProtocol: invalid: invalid"),
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.LBSvcProtocolFromService(svc)
	})
}

func TestName_LBAlgorithmTypeFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]string{
				ann: string(hcloud.LoadBalancerAlgorithmTypeLeastConnections),
			},
			expected: hcloud.LoadBalancerAlgorithmTypeLeastConnections,
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
		{
			name: "value invalid",
			svcAnnotations: map[annotation.Name]string{
				ann: "invalid",
			},
			err: errors.New("annotation/Name.LBAlgorithmTypeFromService: annotation/validateAlgorithmType: invalid: invalid"),
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.LBAlgorithmTypeFromService(svc)
	})
}

func TestName_NetworkZoneFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]string{
				ann: string(hcloud.NetworkZoneEUCentral),
			},
			expected: hcloud.NetworkZoneEUCentral,
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.NetworkZoneFromService(svc)
	})
}

func TestName_CertificatesFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "ids set",
			svcAnnotations: map[annotation.Name]string{
				ann: "3,5",
			},
			expected: []*hcloud.Certificate{{ID: 3}, {ID: 5}},
		},
		{
			name: "names set",
			svcAnnotations: map[annotation.Name]string{
				ann: "cert-1,cert-2",
			},
			expected: []*hcloud.Certificate{{Name: "cert-1"}, {Name: "cert-2"}},
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.CertificatesFromService(svc)
	})
}

func TestName_CertificateTypeFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "uploaded certificate",
			svcAnnotations: map[annotation.Name]string{
				ann: string(hcloud.CertificateTypeUploaded),
			},
			expected: hcloud.CertificateTypeUploaded,
		},
		{
			name: "managed certificate",
			svcAnnotations: map[annotation.Name]string{
				ann: string(hcloud.CertificateTypeManaged),
			},
			expected: hcloud.CertificateTypeManaged,
		},
		{
			name: "unsupported certificate type",
			svcAnnotations: map[annotation.Name]string{
				ann: "unsupported type",
			},
			err: fmt.Errorf("annotation/Name.CertificateTypeFromService: annotation/Name.CertificateTypeFromService: unsupported certificate type: unsupported type"),
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *corev1.Service) (interface{}, error) {
		return ann.CertificateTypeFromService(svc)
	})
}

type typedAccessorTest struct {
	name           string
	svcAnnotations map[annotation.Name]string
	err            error
	expected       interface{}
}

func (tt *typedAccessorTest) run(t *testing.T, call func(svc *corev1.Service) (interface{}, error)) {
	t.Helper()

	var svc corev1.Service
	svc.Annotations = map[string]string{}

	for k, v := range tt.svcAnnotations {
		svc.Annotations[string(k)] = v
	}

	actual, err := call(&svc)
	if tt.err != nil {
		if errors.Is(err, tt.err) {
			return
		}
		assert.EqualError(t, err, tt.err.Error())
		return
	}
	assert.NoError(t, err)
	// Don't use assert.Equal to compare nil values, as it requires the nil
	// values to be casted to the correct type.
	if tt.expected == nil && actual == nil {
		return
	}
	assert.Equal(t, tt.expected, actual)
}

func runAllTypedAccessorTests(
	t *testing.T, tests []typedAccessorTest, call func(svc *corev1.Service) (interface{}, error),
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, call)
		})
	}
}
