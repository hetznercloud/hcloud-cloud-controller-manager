package hcops

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestHCLBServiceOptsBuilder(t *testing.T) {
	type testCase struct {
		name               string
		servicePort        corev1.ServicePort
		serviceUID         string
		serviceAnnotations map[annotation.Name]string
		expectedAddOpts    hcloud.LoadBalancerAddServiceOpts
		expectedUpdateOpts hcloud.LoadBalancerUpdateServiceOpts
		mock               func(t *testing.T, tt *testCase)

		// Set during test setup
		certClient *mocks.CertificateClient
	}

	tests := []testCase{
		{
			name:        "defaults",
			servicePort: corev1.ServicePort{Port: 80, NodePort: 8080},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(80),
				DestinationPort: hcloud.Ptr(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8080),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8080),
				},
			},
		},
		{
			name:        "enable proxy protocol",
			servicePort: corev1.ServicePort{Port: 81, NodePort: 8081},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProxyProtocol: "true",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(81),
				DestinationPort: hcloud.Ptr(8081),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				Proxyprotocol:   hcloud.Ptr(true),
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8081),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8081),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				Proxyprotocol:   hcloud.Ptr(true),
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8081),
				},
			},
		},
		{
			name:        "select HTTP protocol",
			servicePort: corev1.ServicePort{Port: 82, NodePort: 8082},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:           string(hcloud.LoadBalancerServiceProtocolHTTP),
				annotation.LBSvcHTTPCookieName:     "my-cookie",
				annotation.LBSvcHTTPCookieLifetime: "1h",
				annotation.LBSvcHTTPCertificates:   "1,3",
				annotation.LBSvcRedirectHTTP:       "true",
				annotation.LBSvcHTTPStickySessions: "true",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(82),
				DestinationPort: hcloud.Ptr(8082),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					CookieName:     hcloud.Ptr("my-cookie"),
					CookieLifetime: hcloud.Ptr(time.Hour),
					Certificates:   []*hcloud.Certificate{{ID: 1}, {ID: 3}},
					RedirectHTTP:   hcloud.Ptr(true),
					StickySessions: hcloud.Ptr(true),
				},
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8082),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8082),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					CookieName:     hcloud.Ptr("my-cookie"),
					CookieLifetime: hcloud.Ptr(time.Hour),
					Certificates:   []*hcloud.Certificate{{ID: 1}, {ID: 3}},
					RedirectHTTP:   hcloud.Ptr(true),
					StickySessions: hcloud.Ptr(true),
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8082),
				},
			},
		},
		{
			name:        "add certificates by name",
			servicePort: corev1.ServicePort{Port: 83, NodePort: 8083},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:         string(hcloud.LoadBalancerServiceProtocolHTTPS),
				annotation.LBSvcHTTPCertificates: "cert-1,cert-2",
			},
			mock: func(_ *testing.T, tt *testCase) {
				tt.certClient.
					On("Get", mock.Anything, "cert-1").
					Return(&hcloud.Certificate{ID: 1, Name: "cert-1"}, nil, nil)
				tt.certClient.
					On("Get", mock.Anything, "cert-2").
					Return(&hcloud.Certificate{ID: 2, Name: "cert-2"}, nil, nil)
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(83),
				DestinationPort: hcloud.Ptr(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}, {ID: 2}},
				},
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8083),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}, {ID: 2}},
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8083),
				},
			},
		},
		{
			name:        "add managed certificate by service uid label",
			servicePort: corev1.ServicePort{Port: 83, NodePort: 8083},
			serviceUID:  "some-service-uid",
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:                      string(hcloud.LoadBalancerServiceProtocolHTTPS),
				annotation.LBSvcHTTPCertificateType:           "managed",
				annotation.LBSvcHTTPManagedCertificateDomains: "*.example.com,example.com",
			},
			mock: func(_ *testing.T, tt *testCase) {
				tt.certClient.
					On("AllWithOpts", mock.Anything, hcloud.CertificateListOpts{
						ListOpts: hcloud.ListOpts{
							LabelSelector: fmt.Sprintf("%s=%s", LabelServiceUID, "some-service-uid"),
						},
					}).
					Return([]*hcloud.Certificate{{ID: 1}}, nil, nil)
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(83),
				DestinationPort: hcloud.Ptr(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}},
				},
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8083),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}},
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8083),
				},
			},
		},
		{
			name:        "add health check with default protocol",
			servicePort: corev1.ServicePort{Port: 83, NodePort: 8083},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:        string(hcloud.LoadBalancerServiceProtocolTCP),
				annotation.LBSvcHealthCheckPort: "8084",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(83),
				DestinationPort: hcloud.Ptr(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8084),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8084),
				},
			},
		},
		{
			name:        "add TCP health check",
			servicePort: corev1.ServicePort{Port: 83, NodePort: 8083},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcHealthCheckProtocol: string(hcloud.LoadBalancerServiceProtocolTCP),
				annotation.LBSvcHealthCheckPort:     "8084",
				annotation.LBSvcHealthCheckInterval: "1h",
				annotation.LBSvcHealthCheckTimeout:  "30s",
				annotation.LBSvcHealthCheckRetries:  "5",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(83),
				DestinationPort: hcloud.Ptr(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8084),
					Interval: hcloud.Ptr(time.Hour),
					Timeout:  hcloud.Ptr(30 * time.Second),
					Retries:  hcloud.Ptr(5),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8084),
					Interval: hcloud.Ptr(time.Hour),
					Timeout:  hcloud.Ptr(30 * time.Second),
					Retries:  hcloud.Ptr(5),
				},
			},
		},
		{
			name:        "add HTTP health check",
			servicePort: corev1.ServicePort{Port: 84, NodePort: 8084},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcHealthCheckProtocol:                string(hcloud.LoadBalancerServiceProtocolHTTP),
				annotation.LBSvcHealthCheckPort:                    "8085",
				annotation.LBSvcHealthCheckInterval:                "1h",
				annotation.LBSvcHealthCheckTimeout:                 "30s",
				annotation.LBSvcHealthCheckRetries:                 "5",
				annotation.LBSvcHealthCheckHTTPDomain:              "example.com",
				annotation.LBSvcHealthCheckHTTPPath:                "/internal/health",
				annotation.LBSvcHealthCheckHTTPValidateCertificate: "true",
				annotation.LBSvcHealthCheckHTTPStatusCodes:         "200,202",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(84),
				DestinationPort: hcloud.Ptr(8084),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
					Port:     hcloud.Ptr(8085),
					Interval: hcloud.Ptr(time.Hour),
					Timeout:  hcloud.Ptr(30 * time.Second),
					Retries:  hcloud.Ptr(5),
					HTTP: &hcloud.LoadBalancerAddServiceOptsHealthCheckHTTP{
						Domain:      hcloud.Ptr("example.com"),
						Path:        hcloud.Ptr("/internal/health"),
						TLS:         hcloud.Ptr(true),
						StatusCodes: []string{"200", "202"},
					},
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8084),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
					Port:     hcloud.Ptr(8085),
					Interval: hcloud.Ptr(time.Hour),
					Timeout:  hcloud.Ptr(30 * time.Second),
					Retries:  hcloud.Ptr(5),
					HTTP: &hcloud.LoadBalancerUpdateServiceOptsHealthCheckHTTP{
						Domain:      hcloud.Ptr("example.com"),
						Path:        hcloud.Ptr("/internal/health"),
						TLS:         hcloud.Ptr(true),
						StatusCodes: []string{"200", "202"},
					},
				},
			},
		},
		{
			name:        "health check port defaults to node port/destination Port if not specified",
			servicePort: corev1.ServicePort{Port: 84, NodePort: 8084},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcHealthCheckProtocol:                string(hcloud.LoadBalancerServiceProtocolHTTP),
				annotation.LBSvcHealthCheckInterval:                "1h",
				annotation.LBSvcHealthCheckTimeout:                 "30s",
				annotation.LBSvcHealthCheckRetries:                 "5",
				annotation.LBSvcHealthCheckHTTPDomain:              "example.com",
				annotation.LBSvcHealthCheckHTTPPath:                "/internal/health",
				annotation.LBSvcHealthCheckHTTPValidateCertificate: "true",
				annotation.LBSvcHealthCheckHTTPStatusCodes:         "200,202",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(84),
				DestinationPort: hcloud.Ptr(8084),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
					Port:     hcloud.Ptr(8084),
					Interval: hcloud.Ptr(time.Hour),
					Timeout:  hcloud.Ptr(30 * time.Second),
					Retries:  hcloud.Ptr(5),
					HTTP: &hcloud.LoadBalancerAddServiceOptsHealthCheckHTTP{
						Domain:      hcloud.Ptr("example.com"),
						Path:        hcloud.Ptr("/internal/health"),
						TLS:         hcloud.Ptr(true),
						StatusCodes: []string{"200", "202"},
					},
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8084),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
					Port:     hcloud.Ptr(8084),
					Interval: hcloud.Ptr(time.Hour),
					Timeout:  hcloud.Ptr(30 * time.Second),
					Retries:  hcloud.Ptr(5),
					HTTP: &hcloud.LoadBalancerUpdateServiceOptsHealthCheckHTTP{
						Domain:      hcloud.Ptr("example.com"),
						Path:        hcloud.Ptr("/internal/health"),
						TLS:         hcloud.Ptr(true),
						StatusCodes: []string{"200", "202"},
					},
				},
			},
		},
		{
			name:        "per-port protocol configuration - HTTP",
			servicePort: corev1.ServicePort{Port: 80, NodePort: 8080},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:      string(hcloud.LoadBalancerServiceProtocolTCP), // Global default
				annotation.LBSvcProtocolPorts: "80:http,443:https,9000:tcp",               // Per-port override
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(80),
				DestinationPort: hcloud.Ptr(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP, // Should use per-port config
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8080),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8080),
				},
			},
		},
		{
			name:        "per-port protocol configuration - HTTPS",
			servicePort: corev1.ServicePort{Port: 443, NodePort: 8443},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:      string(hcloud.LoadBalancerServiceProtocolTCP), // Global default
				annotation.LBSvcProtocolPorts: "80:http,443:https,9000:tcp",               // Per-port override
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(443),
				DestinationPort: hcloud.Ptr(8443),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS, // Should use per-port config
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8443),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8443),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8443),
				},
			},
		},
		{
			name:        "per-port protocol configuration - TCP fallback",
			servicePort: corev1.ServicePort{Port: 9000, NodePort: 9000},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:      string(hcloud.LoadBalancerServiceProtocolHTTP), // Global default
				annotation.LBSvcProtocolPorts: "80:http,443:https,9000:tcp",                // Per-port override
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(9000),
				DestinationPort: hcloud.Ptr(9000),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP, // Should use per-port config
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(9000),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(9000),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(9000),
				},
			},
		},
		{
			name:        "per-port protocol configuration - not configured port uses global",
			servicePort: corev1.ServicePort{Port: 8080, NodePort: 8080},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:      string(hcloud.LoadBalancerServiceProtocolHTTP), // Global default
				annotation.LBSvcProtocolPorts: "80:http,443:https,9000:tcp",                // Per-port override
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(8080),
				DestinationPort: hcloud.Ptr(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP, // Should use global config
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8080),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8080),
				},
			},
		},
		{
			name:        "per-port certificate configuration",
			servicePort: corev1.ServicePort{Port: 443, NodePort: 8443},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:               string(hcloud.LoadBalancerServiceProtocolHTTPS),
				annotation.LBSvcHTTPCertificates:       "global-cert1,global-cert2",                  // Global default
				annotation.LBSvcHTTPCertificatesPorts:  "443:port-cert1,port-cert2;8443:port-cert3", // Per-port override
			},
			mock: func(_ *testing.T, tt *testCase) {
				tt.certClient.
					On("Get", mock.Anything, "port-cert1").
					Return(&hcloud.Certificate{ID: 1, Name: "port-cert1"}, nil, nil)
				tt.certClient.
					On("Get", mock.Anything, "port-cert2").
					Return(&hcloud.Certificate{ID: 2, Name: "port-cert2"}, nil, nil)
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(443),
				DestinationPort: hcloud.Ptr(8443),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}, {ID: 2}}, // Should use per-port config
				},
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8443),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8443),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}, {ID: 2}},
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8443),
				},
			},
		},
		{
			name:        "per-port certificate configuration - not configured port uses global",
			servicePort: corev1.ServicePort{Port: 8443, NodePort: 8443},
			serviceAnnotations: map[annotation.Name]string{
				annotation.LBSvcProtocol:               string(hcloud.LoadBalancerServiceProtocolHTTPS),
				annotation.LBSvcHTTPCertificates:       "global-cert1,global-cert2",                  // Global default
				annotation.LBSvcHTTPCertificatesPorts:  "443:port-cert1,port-cert2;9443:port-cert3", // Per-port override
			},
			mock: func(_ *testing.T, tt *testCase) {
				tt.certClient.
					On("Get", mock.Anything, "global-cert1").
					Return(&hcloud.Certificate{ID: 10, Name: "global-cert1"}, nil, nil)
				tt.certClient.
					On("Get", mock.Anything, "global-cert2").
					Return(&hcloud.Certificate{ID: 11, Name: "global-cert2"}, nil, nil)
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Ptr(8443),
				DestinationPort: hcloud.Ptr(8443),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 10}, {ID: 11}}, // Should use global config
				},
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8443),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Ptr(8443),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 10}, {ID: 11}},
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(8443),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.certClient = &mocks.CertificateClient{}
			tt.certClient.Test(t)

			if tt.mock != nil {
				tt.mock(t, &tt)
			}

			builder := &hclbServiceOptsBuilder{
				Port: tt.servicePort,
				Service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						UID:         types.UID(tt.serviceUID),
						Annotations: map[string]string{},
					},
				},
				CertOps: &CertificateOps{CertClient: tt.certClient},
			}
			for k, v := range tt.serviceAnnotations {
				builder.Service.Annotations[string(k)] = v
			}
			addOpts, err := builder.buildAddServiceOpts()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedAddOpts, addOpts)

			updateOpts, err := builder.buildUpdateServiceOpts()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedUpdateOpts, updateOpts)
		})
	}
}
