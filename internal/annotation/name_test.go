package annotation_test

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/assert"
	"github.com/syself/hetzner-cloud-controller-manager/internal/annotation"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ann annotation.Name = "some/annotation"

func TestName_AddToService(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		svc      v1.Service
		err      error
		expected map[string]string
	}{
		{
			name:     "set string",
			value:    "some value",
			expected: map[string]string{string(ann): "some value"},
		},
		{
			name:     "set stringer",
			value:    stringer{"some value"},
			expected: map[string]string{string(ann): "some value"},
		},

		{
			name:     "set bool",
			value:    true,
			expected: map[string]string{string(ann): "true"},
		},
		{
			name:     "set int",
			value:    10,
			expected: map[string]string{string(ann): "10"},
		},
		{
			name:     "set []string",
			value:    []string{"a", "b"},
			expected: map[string]string{string(ann): "a,b"},
		},
		{
			name:     "set hcloud.LoadBalancerServiceProtocol",
			value:    hcloud.LoadBalancerServiceProtocolTCP,
			expected: map[string]string{string(ann): string(hcloud.LoadBalancerServiceProtocolTCP)},
		},
		{
			name:     "set []*hcloud.Certificate",
			value:    []*hcloud.Certificate{{ID: 1}, {ID: 2}},
			expected: map[string]string{string(ann): "1,2"},
		},
		{
			name:     "set []*hcloud.Certificate by name",
			value:    []*hcloud.Certificate{{Name: "cert-1"}, {Name: "cert-2"}},
			expected: map[string]string{string(ann): "cert-1,cert-2"},
		},
		{
			name:  "set unsupported value",
			value: struct{}{},
			err:   fmt.Errorf("annotation/Name.AnnotateService: %v: unsupported type: %T", ann, struct{}{}),
		},
		{
			name:  "does not overwrite unrelated annotations",
			value: "some value",
			svc: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"other/annotation": "other value"},
				},
			},
			expected: map[string]string{
				string(ann):        "some value",
				"other/annotation": "other value",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ann.AnnotateService(&tt.svc, tt.value)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, tt.svc.ObjectMeta.Annotations)
		})
	}
}

func TestName_StringFromService(t *testing.T) {
	tests := []struct {
		name           string
		svcAnnotations map[annotation.Name]interface{}
		ok             bool
		expected       string
	}{
		{
			name:           "value as string",
			svcAnnotations: map[annotation.Name]interface{}{ann: "some value"},
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var svc v1.Service

			for k, v := range tt.svcAnnotations {
				if err := k.AnnotateService(&svc, v); err != nil {
					t.Error(err)
				}
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
			svcAnnotations: map[annotation.Name]interface{}{
				ann: "a,b,c",
			},
			expected: []string{"a", "b", "c"},
		},
		{
			name: "value missing",
			err:  annotation.ErrNotSet,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.StringsFromService(svc)
	})
}

func TestName_BoolFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name:           "value set to true",
			svcAnnotations: map[annotation.Name]interface{}{ann: "true"},
			expected:       true,
		},
		{
			name:           "value set to false",
			svcAnnotations: map[annotation.Name]interface{}{ann: "false"},
			expected:       false,
		},
		{
			name:     "value missing",
			expected: false,
			err:      annotation.ErrNotSet,
		},
		{
			name:           "value invalid",
			svcAnnotations: map[annotation.Name]interface{}{ann: "invalid"},
			expected:       false,
			err:            strconv.ErrSyntax,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.BoolFromService(svc)
	})
}

func TestName_IntFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name:           "value set to 10",
			svcAnnotations: map[annotation.Name]interface{}{ann: 10},
			expected:       10,
		},
		{
			name:     "value missing",
			expected: 0,
			err:      fmt.Errorf("annotation/Name.IntFromService: %s: %w", ann, annotation.ErrNotSet),
		},
		{
			name:           "value invalid",
			svcAnnotations: map[annotation.Name]interface{}{ann: "invalid"},
			expected:       0,
			err:            strconv.ErrSyntax,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.IntFromService(svc)
	})
}

func TestName_IntsFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]interface{}{
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
			svcAnnotations: map[annotation.Name]interface{}{ann: "invalid"},
			err:            strconv.ErrSyntax,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.IntsFromService(svc)
	})
}

func TestName_IPFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set to valid IPv4",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: net.ParseIP("1.2.3.4"),
			},
			expected: net.ParseIP("1.2.3.4"),
		},
		{
			name: "value set to valid IPv6",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: net.ParseIP("3c2e:2ef9:a7e9:1a5b:30ba:4912:e3fe:91b2"),
			},
			expected: net.ParseIP("3c2e:2ef9:a7e9:1a5b:30ba:4912:e3fe:91b2"),
		},
		{
			name: "value invalid",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: "invalid",
			},
			err: errors.New("annotation/Name.IPFromService: invalid ip address: invalid"),
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.IPFromService(svc)
	})
}

func TestName_DurationFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: time.Hour,
			},
			expected: time.Hour,
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
		{
			name: "value invalid",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: "invalid",
			},
			err: errors.New("annotation/Name.DurationFromService: time: invalid duration \"invalid\""),
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.DurationFromService(svc)
	})
}

func TestName_LBSvcProtocolFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: hcloud.LoadBalancerServiceProtocolHTTP,
			},
			expected: hcloud.LoadBalancerServiceProtocolHTTP,
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
		{
			name: "value invalid",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: "invalid",
			},
			err: errors.New("annotation/Name.LBSvcProtocolFromService: annotation/validateServiceProtocol: invalid: invalid"),
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.LBSvcProtocolFromService(svc)
	})
}

func TestName_LBAlgorithmTypeFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: hcloud.LoadBalancerAlgorithmTypeLeastConnections,
			},
			expected: hcloud.LoadBalancerAlgorithmTypeLeastConnections,
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
		{
			name: "value invalid",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: "invalid",
			},
			err: errors.New("annotation/Name.LBAlgorithmTypeFromService: annotation/validateAlgorithmType: invalid: invalid"),
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.LBAlgorithmTypeFromService(svc)
	})
}

func TestName_NetworkZoneFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "value set",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: hcloud.NetworkZoneEUCentral,
			},
			expected: hcloud.NetworkZoneEUCentral,
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.NetworkZoneFromService(svc)
	})
}

func TestName_CertificatesFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "ids set",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: []*hcloud.Certificate{{ID: 3}, {ID: 5}},
			},
			expected: []*hcloud.Certificate{{ID: 3}, {ID: 5}},
		},
		{
			name: "names set",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: []*hcloud.Certificate{{Name: "cert-1"}, {Name: "cert-2"}},
			},
			expected: []*hcloud.Certificate{{Name: "cert-1"}, {Name: "cert-2"}},
		},
		{
			name: "value not set",
			err:  annotation.ErrNotSet,
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.CertificatesFromService(svc)
	})
}

func TestName_CertificateTypeFromService(t *testing.T) {
	tests := []typedAccessorTest{
		{
			name: "uploaded certificate",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: hcloud.CertificateTypeUploaded,
			},
			expected: hcloud.CertificateTypeUploaded,
		},
		{
			name: "managed certificate",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: hcloud.CertificateTypeManaged,
			},
			expected: hcloud.CertificateTypeManaged,
		},
		{
			name: "unsupported certificate type",
			svcAnnotations: map[annotation.Name]interface{}{
				ann: "unsupported type",
			},
			err: fmt.Errorf("annotation/Name.CertificateTypeFromService: annotation/Name.CertificateTypeFromService: unsupported certificate type: unsupported type"),
		},
	}

	runAllTypedAccessorTests(t, tests, func(svc *v1.Service) (interface{}, error) {
		return ann.CertificateTypeFromService(svc)
	})
}

type stringer struct{ Value string }

func (s stringer) String() string {
	return s.Value
}

type typedAccessorTest struct {
	name           string
	svcAnnotations map[annotation.Name]interface{}
	err            error
	expected       interface{}
}

func (tt *typedAccessorTest) run(t *testing.T, call func(svc *v1.Service) (interface{}, error)) {
	var svc v1.Service

	t.Helper()

	for k, v := range tt.svcAnnotations {
		if err := k.AnnotateService(&svc, v); !assert.NoError(t, err) {
			return
		}
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
	t *testing.T, tests []typedAccessorTest, call func(svc *v1.Service) (interface{}, error),
) {
	t.Helper()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, call)
		})
	}
}
