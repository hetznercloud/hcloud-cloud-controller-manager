package annotation

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
)

func TestName_ProtocolPortsFromService(t *testing.T) {
	tests := []struct {
		name        string
		annotation  string
		expected    map[int]hcloud.LoadBalancerServiceProtocol
		expectError bool
	}{
		{
			name:        "valid protocol ports",
			annotation:  "80:http,443:https,9000:tcp",
			expected: map[int]hcloud.LoadBalancerServiceProtocol{
				80:   hcloud.LoadBalancerServiceProtocolHTTP,
				443:  hcloud.LoadBalancerServiceProtocolHTTPS,
				9000: hcloud.LoadBalancerServiceProtocolTCP,
			},
		},
		{
			name:        "single protocol port",
			annotation:  "80:http",
			expected: map[int]hcloud.LoadBalancerServiceProtocol{
				80: hcloud.LoadBalancerServiceProtocolHTTP,
			},
		},
		{
			name:        "empty annotation",
			annotation:  "",
			expected:    map[int]hcloud.LoadBalancerServiceProtocol{},
		},
		{
			name:        "invalid format - missing colon",
			annotation:  "80http",
			expectError: true,
		},
		{
			name:        "invalid format - invalid port",
			annotation:  "abc:http",
			expectError: true,
		},
		{
			name:        "invalid format - invalid protocol",
			annotation:  "80:invalid",
			expectError: true,
		},
		{
			name:        "whitespace handling",
			annotation:  " 80 : http , 443 : https ",
			expected: map[int]hcloud.LoadBalancerServiceProtocol{
				80:  hcloud.LoadBalancerServiceProtocolHTTP,
				443: hcloud.LoadBalancerServiceProtocolHTTPS,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						string(LBSvcProtocolPorts): tt.annotation,
					},
				},
			}

			result, err := LBSvcProtocolPorts.ProtocolPortsFromService(svc)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestName_ProtocolPortsFromService_NoAnnotation(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	result, err := LBSvcProtocolPorts.ProtocolPortsFromService(svc)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestName_CertificatePortsFromService(t *testing.T) {
	tests := []struct {
		name        string
		annotation  string
		expected    map[int][]*hcloud.Certificate
		expectError bool
	}{
		{
			name:       "valid certificate ports",
			annotation: "443:cert1,cert2;8443:cert3",
			expected: map[int][]*hcloud.Certificate{
				443: {
					{Name: "cert1"},
					{Name: "cert2"},
				},
				8443: {
					{Name: "cert3"},
				},
			},
		},
		{
			name:       "single certificate port",
			annotation: "443:cert1",
			expected: map[int][]*hcloud.Certificate{
				443: {
					{Name: "cert1"},
				},
			},
		},
		{
			name:       "certificate IDs",
			annotation: "443:123,456",
			expected: map[int][]*hcloud.Certificate{
				443: {
					{ID: 123},
					{ID: 456},
				},
			},
		},
		{
			name:       "mixed names and IDs",
			annotation: "443:cert1,123",
			expected: map[int][]*hcloud.Certificate{
				443: {
					{Name: "cert1"},
					{ID: 123},
				},
			},
		},
		{
			name:        "empty annotation",
			annotation:  "",
			expected:    map[int][]*hcloud.Certificate{},
		},
		{
			name:        "invalid format - missing colon",
			annotation:  "443cert1",
			expectError: true,
		},
		{
			name:        "invalid format - invalid port",
			annotation:  "abc:cert1",
			expectError: true,
		},
		{
			name:        "invalid format - empty certificate",
			annotation:  "443:cert1,,cert2",
			expectError: true,
		},
		{
			name:       "whitespace handling",
			annotation: " 443 : cert1 , cert2 ; 8443 : cert3 ",
			expected: map[int][]*hcloud.Certificate{
				443: {
					{Name: "cert1"},
					{Name: "cert2"},
				},
				8443: {
					{Name: "cert3"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						string(LBSvcHTTPCertificatesPorts): tt.annotation,
					},
				},
			}

			result, err := LBSvcHTTPCertificatesPorts.CertificatePortsFromService(svc)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestName_CertificatePortsFromService_NoAnnotation(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	result, err := LBSvcHTTPCertificatesPorts.CertificatePortsFromService(svc)

	assert.NoError(t, err)
	assert.Empty(t, result)
}