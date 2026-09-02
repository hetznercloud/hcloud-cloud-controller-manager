package annotation_test

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// Every accessor is exercised through the same annotation name, declared once
// per type.
const annName = "some/annotation"

var qAnnName = fmt.Sprintf("%q", annName)

const (
	annString        annotation.String        = annName
	annBool          annotation.Bool          = annName
	annInt           annotation.Int           = annName
	annDuration      annotation.Duration      = annName
	annStrings       annotation.Strings       = annName
	annIP            annotation.IP            = annName
	annProtocol      annotation.Protocol      = annName
	annAlgorithmType annotation.AlgorithmType = annName
	annCertificates  annotation.Certificates  = annName
)

func TestString(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "value as string",
			value:    "some value",
			expected: "some value",
		},
		{
			// An annotation set to the empty string is set, and some settings
			// use that to opt out of a cluster-wide default.
			name:     "value set to the empty string",
			value:    "",
			expected: "",
		},
		{
			name:     "value not set",
			notSet:   true,
			expected: "",
			err:      errors.New(qAnnName + ": not set"),
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annString.FromService(svc)
	})

	// The harness always assigns an annotation map, so a Service without any
	// annotations is covered separately.
	t.Run("Service has no annotations", func(t *testing.T) {
		actual, err := annString.FromService(&corev1.Service{})

		assert.ErrorIs(t, err, annotation.ErrNotSet)
		assert.Empty(t, actual)
	})
}

func TestBool(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "value set to true",
			value:    "true",
			expected: true,
		},
		{
			name:     "value set to false",
			value:    "false",
			expected: false,
		},
		{
			name:     "value not set",
			notSet:   true,
			expected: false,
			err:      annotation.ErrNotSet,
		},
		{
			name:     "value invalid",
			value:    "invalid",
			expected: false,
			err:      strconv.ErrSyntax,
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annBool.FromService(svc)
	})
}

func TestInt(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "value set to 10",
			value:    "10",
			expected: 10,
		},
		{
			name:     "value not set",
			notSet:   true,
			expected: 0,
			err:      errors.New(qAnnName + ": not set"),
		},
		{
			name:     "value invalid",
			value:    "invalid",
			expected: 0,
			err:      strconv.ErrSyntax,
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annInt.FromService(svc)
	})
}

func TestDuration(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "value set",
			value:    "1h",
			expected: time.Hour,
		},
		{
			name:   "value not set",
			notSet: true,
			err:    annotation.ErrNotSet,
		},
		{
			name:  "value invalid",
			value: "invalid",
			err:   errors.New(qAnnName + `: time: invalid duration "invalid"`),
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annDuration.FromService(svc)
	})
}

func TestStrings(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "value set",
			value:    "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:   "value missing",
			notSet: true,
			err:    annotation.ErrNotSet,
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annStrings.FromService(svc)
	})
}

func TestIP(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "value set to valid IPv4",
			value:    "1.2.3.4",
			expected: netip.MustParseAddr("1.2.3.4"),
		},
		{
			name:     "value set to valid IPv6",
			value:    "3c2e:2ef9:a7e9:1a5b:30ba:4912:e3fe:91b2",
			expected: netip.MustParseAddr("3c2e:2ef9:a7e9:1a5b:30ba:4912:e3fe:91b2"),
		},
		{
			name:  "value invalid",
			value: "invalid",
			err:   errors.New(qAnnName + ": invalid ip address: invalid"),
		},
		{
			name:   "value not set",
			notSet: true,
			err:    annotation.ErrNotSet,
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annIP.FromService(svc)
	})
}

func TestProtocol(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "value set",
			value:    string(hcloud.LoadBalancerServiceProtocolHTTP),
			expected: hcloud.LoadBalancerServiceProtocolHTTP,
		},
		{
			name:     "value is uppercased",
			value:    "HTTPS",
			expected: hcloud.LoadBalancerServiceProtocolHTTPS,
		},
		{
			name:   "value not set",
			notSet: true,
			err:    annotation.ErrNotSet,
		},
		{
			name:  "value invalid",
			value: "hppt",
			err:   errors.New(qAnnName + ": invalid protocol: hppt"),
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annProtocol.FromService(svc)
	})
}

func TestAlgorithmType(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "value set",
			value:    string(hcloud.LoadBalancerAlgorithmTypeLeastConnections),
			expected: hcloud.LoadBalancerAlgorithmTypeLeastConnections,
		},
		{
			name:     "value is uppercased",
			value:    "ROUND_ROBIN",
			expected: hcloud.LoadBalancerAlgorithmTypeRoundRobin,
		},
		{
			name:   "value not set",
			notSet: true,
			err:    annotation.ErrNotSet,
		},
		{
			name:  "value invalid",
			value: "round_ronald",
			err:   errors.New(qAnnName + ": invalid algorithm type: round_ronald"),
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annAlgorithmType.FromService(svc)
	})
}

func TestCertificates(t *testing.T) {
	tests := []accessorTest{
		{
			name:     "ids set",
			value:    "3,5",
			expected: []*hcloud.Certificate{{ID: 3}, {ID: 5}},
		},
		{
			name:     "names set",
			value:    "cert-1,cert-2",
			expected: []*hcloud.Certificate{{Name: "cert-1"}, {Name: "cert-2"}},
		},
		{
			name:     "ids and names mixed",
			value:    "3,cert-2",
			expected: []*hcloud.Certificate{{ID: 3}, {Name: "cert-2"}},
		},
		{
			name:   "value not set",
			notSet: true,
			err:    annotation.ErrNotSet,
		},
	}

	runAllAccessorTests(t, tests, func(svc *corev1.Service) (any, error) {
		return annCertificates.FromService(svc)
	})
}

type accessorTest struct {
	name string
	// value is put on the Service unless notSet is true.
	value    string
	notSet   bool
	err      error
	expected any
}

func (tt *accessorTest) run(t *testing.T, call func(svc *corev1.Service) (any, error)) {
	t.Helper()

	var svc corev1.Service
	svc.Annotations = map[string]string{}

	if !tt.notSet {
		svc.Annotations[annName] = tt.value
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

func runAllAccessorTests(
	t *testing.T, tests []accessorTest, call func(svc *corev1.Service) (any, error),
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, call)
		})
	}
}
