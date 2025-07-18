package hcops

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestRealWorldPerPortConfiguration(t *testing.T) {
	certClient := &mocks.CertificateClient{}
	certClient.Test(t)

	// Set up mock expectations for certificates
	certClient.
		On("Get", mock.Anything, "web-cert").
		Return(&hcloud.Certificate{ID: 1, Name: "web-cert"}, nil, nil)
	certClient.
		On("Get", mock.Anything, "api-cert").
		Return(&hcloud.Certificate{ID: 2, Name: "api-cert"}, nil, nil)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-service",
			Namespace: "default",
			UID:       types.UID("test-uid"),
			Annotations: map[string]string{
				string(annotation.LBSvcProtocol):               "tcp",
				string(annotation.LBSvcProtocolPorts):         "80:http,443:https,9000:tcp",
				string(annotation.LBSvcHTTPCertificatesPorts): "443:web-cert,api-cert",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					NodePort: 32080,
				},
				{
					Name:     "https",
					Port:     443,
					NodePort: 32443,
				},
				{
					Name:     "tcp",
					Port:     9000,
					NodePort: 32900,
				},
			},
		},
	}

	testCases := []struct {
		name                string
		port                corev1.ServicePort
		expectedProtocol    hcloud.LoadBalancerServiceProtocol
		expectedCertificates []*hcloud.Certificate
	}{
		{
			name:             "HTTP port 80",
			port:             service.Spec.Ports[0],
			expectedProtocol: hcloud.LoadBalancerServiceProtocolHTTP,
			expectedCertificates: nil,
		},
		{
			name:             "HTTPS port 443 with certificates",
			port:             service.Spec.Ports[1],
			expectedProtocol: hcloud.LoadBalancerServiceProtocolHTTPS,
			expectedCertificates: []*hcloud.Certificate{
				{ID: 1}, {ID: 2},
			},
		},
		{
			name:             "TCP port 9000",
			port:             service.Spec.Ports[2],
			expectedProtocol: hcloud.LoadBalancerServiceProtocolTCP,
			expectedCertificates: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := &hclbServiceOptsBuilder{
				Port:    tc.port,
				Service: service,
				CertOps: &CertificateOps{CertClient: certClient},
			}

			addOpts, err := builder.buildAddServiceOpts()
			assert.NoError(t, err)

			// Verify protocol
			assert.Equal(t, tc.expectedProtocol, addOpts.Protocol)

			// Verify certificates
			if tc.expectedCertificates != nil {
				assert.NotNil(t, addOpts.HTTP)
				assert.Equal(t, tc.expectedCertificates, addOpts.HTTP.Certificates)
			} else {
				if addOpts.HTTP != nil {
					assert.Nil(t, addOpts.HTTP.Certificates)
				}
			}

			updateOpts, err := builder.buildUpdateServiceOpts()
			assert.NoError(t, err)

			// Verify protocol
			assert.Equal(t, tc.expectedProtocol, updateOpts.Protocol)

			// Verify certificates
			if tc.expectedCertificates != nil {
				assert.NotNil(t, updateOpts.HTTP)
				assert.Equal(t, tc.expectedCertificates, updateOpts.HTTP.Certificates)
			} else {
				if updateOpts.HTTP != nil {
					assert.Nil(t, updateOpts.HTTP.Certificates)
				}
			}
		})
	}
}