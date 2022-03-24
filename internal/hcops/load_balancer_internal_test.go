package hcops

import (
	"fmt"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/syself/hetzner-cloud-controller-manager/internal/annotation"
	"github.com/syself/hetzner-cloud-controller-manager/internal/mocks"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestHCLBServiceOptsBuilder(t *testing.T) {
	type testCase struct {
		name               string
		servicePort        v1.ServicePort
		serviceUID         string
		serviceAnnotations map[annotation.Name]interface{}
		expectedAddOpts    hcloud.LoadBalancerAddServiceOpts
		expectedUpdateOpts hcloud.LoadBalancerUpdateServiceOpts
		mock               func(t *testing.T, tt *testCase)

		// Set during test setup
		certClient *mocks.CertificateClient
	}

	tests := []testCase{
		{
			name:        "defaults",
			servicePort: v1.ServicePort{Port: 80, NodePort: 8080},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(80),
				DestinationPort: hcloud.Int(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8080),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8080),
				},
			},
		},
		{
			name:        "enable proxy protocol",
			servicePort: v1.ServicePort{Port: 81, NodePort: 8081},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcProxyProtocol: true,
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(81),
				DestinationPort: hcloud.Int(8081),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				Proxyprotocol:   hcloud.Bool(true),
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8081),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8081),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				Proxyprotocol:   hcloud.Bool(true),
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8081),
				},
			},
		},
		{
			name:        "select HTTP protocol",
			servicePort: v1.ServicePort{Port: 82, NodePort: 8082},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcProtocol:           hcloud.LoadBalancerServiceProtocolHTTP,
				annotation.LBSvcHTTPCookieName:     "my-cookie",
				annotation.LBSvcHTTPCookieLifetime: time.Hour,
				annotation.LBSvcHTTPCertificates:   []*hcloud.Certificate{{ID: 1}, {ID: 3}},
				annotation.LBSvcRedirectHTTP:       true,
				annotation.LBSvcHTTPStickySessions: true,
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(82),
				DestinationPort: hcloud.Int(8082),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					CookieName:     hcloud.String("my-cookie"),
					CookieLifetime: hcloud.Duration(time.Hour),
					Certificates:   []*hcloud.Certificate{{ID: 1}, {ID: 3}},
					RedirectHTTP:   hcloud.Bool(true),
					StickySessions: hcloud.Bool(true),
				},
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8082),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8082),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					CookieName:     hcloud.String("my-cookie"),
					CookieLifetime: hcloud.Duration(time.Hour),
					Certificates:   []*hcloud.Certificate{{ID: 1}, {ID: 3}},
					RedirectHTTP:   hcloud.Bool(true),
					StickySessions: hcloud.Bool(true),
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8082),
				},
			},
		},
		{
			name:        "add certificates by name",
			servicePort: v1.ServicePort{Port: 83, NodePort: 8083},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcProtocol:         hcloud.LoadBalancerServiceProtocolHTTPS,
				annotation.LBSvcHTTPCertificates: []*hcloud.Certificate{{Name: "cert-1"}, {Name: "cert-2"}},
			},
			mock: func(t *testing.T, tt *testCase) {
				tt.certClient.
					On("Get", mock.Anything, "cert-1").
					Return(&hcloud.Certificate{ID: 1, Name: "cert-1"}, nil, nil)
				tt.certClient.
					On("Get", mock.Anything, "cert-2").
					Return(&hcloud.Certificate{ID: 2, Name: "cert-2"}, nil, nil)
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(83),
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}, {ID: 2}},
				},
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8083),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}, {ID: 2}},
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8083),
				},
			},
		},
		{
			name:        "add managed certificate by service uid label",
			servicePort: v1.ServicePort{Port: 83, NodePort: 8083},
			serviceUID:  "some-service-uid",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcProtocol:                      hcloud.LoadBalancerServiceProtocolHTTPS,
				annotation.LBSvcHTTPCertificateType:           "managed",
				annotation.LBSvcHTTPManagedCertificateDomains: []string{"*.example.com", "example.com"},
			},
			mock: func(t *testing.T, tt *testCase) {
				tt.certClient.
					On("AllWithOpts", mock.Anything, hcloud.CertificateListOpts{
						ListOpts: hcloud.ListOpts{
							LabelSelector: fmt.Sprintf("%s=%s", LabelServiceUID, "some-service-uid"),
						},
					}).
					Return([]*hcloud.Certificate{{ID: 1}}, nil, nil)
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(83),
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}},
				},
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8083),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					Certificates: []*hcloud.Certificate{{ID: 1}},
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8083),
				},
			},
		},
		{
			name:        "add health check with default protocol",
			servicePort: v1.ServicePort{Port: 83, NodePort: 8083},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcProtocol:        hcloud.LoadBalancerServiceProtocolTCP,
				annotation.LBSvcHealthCheckPort: 8084,
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(83),
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8084),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8084),
				},
			},
		},
		{
			name:        "add TCP health check",
			servicePort: v1.ServicePort{Port: 83, NodePort: 8083},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcHealthCheckProtocol: string(hcloud.LoadBalancerServiceProtocolTCP),
				annotation.LBSvcHealthCheckPort:     8084,
				annotation.LBSvcHealthCheckInterval: time.Hour,
				annotation.LBSvcHealthCheckTimeout:  30 * time.Second,
				annotation.LBSvcHealthCheckRetries:  5,
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(83),
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8084),
					Interval: hcloud.Duration(time.Hour),
					Timeout:  hcloud.Duration(30 * time.Second),
					Retries:  hcloud.Int(5),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Int(8084),
					Interval: hcloud.Duration(time.Hour),
					Timeout:  hcloud.Duration(30 * time.Second),
					Retries:  hcloud.Int(5),
				},
			},
		},
		{
			name:        "add HTTP health check",
			servicePort: v1.ServicePort{Port: 84, NodePort: 8084},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcHealthCheckProtocol:                hcloud.LoadBalancerServiceProtocolHTTP,
				annotation.LBSvcHealthCheckPort:                    8085,
				annotation.LBSvcHealthCheckInterval:                time.Hour,
				annotation.LBSvcHealthCheckTimeout:                 30 * time.Second,
				annotation.LBSvcHealthCheckRetries:                 5,
				annotation.LBSvcHealthCheckHTTPDomain:              "example.com",
				annotation.LBSvcHealthCheckHTTPPath:                "/internal/health",
				annotation.LBSvcHealthCheckHTTPValidateCertificate: "true",
				annotation.LBSvcHealthCheckHTTPStatusCodes:         "200,202",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(84),
				DestinationPort: hcloud.Int(8084),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
					Port:     hcloud.Int(8085),
					Interval: hcloud.Duration(time.Hour),
					Timeout:  hcloud.Duration(30 * time.Second),
					Retries:  hcloud.Int(5),
					HTTP: &hcloud.LoadBalancerAddServiceOptsHealthCheckHTTP{
						Domain:      hcloud.String("example.com"),
						Path:        hcloud.String("/internal/health"),
						TLS:         hcloud.Bool(true),
						StatusCodes: []string{"200", "202"},
					},
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8084),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
					Port:     hcloud.Int(8085),
					Interval: hcloud.Duration(time.Hour),
					Timeout:  hcloud.Duration(30 * time.Second),
					Retries:  hcloud.Int(5),
					HTTP: &hcloud.LoadBalancerUpdateServiceOptsHealthCheckHTTP{
						Domain:      hcloud.String("example.com"),
						Path:        hcloud.String("/internal/health"),
						TLS:         hcloud.Bool(true),
						StatusCodes: []string{"200", "202"},
					},
				},
			},
		},
		{
			name:        "health check port defaults to node port/destination Port if not specified",
			servicePort: v1.ServicePort{Port: 84, NodePort: 8084},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcHealthCheckProtocol:                hcloud.LoadBalancerServiceProtocolHTTP,
				annotation.LBSvcHealthCheckInterval:                time.Hour,
				annotation.LBSvcHealthCheckTimeout:                 30 * time.Second,
				annotation.LBSvcHealthCheckRetries:                 5,
				annotation.LBSvcHealthCheckHTTPDomain:              "example.com",
				annotation.LBSvcHealthCheckHTTPPath:                "/internal/health",
				annotation.LBSvcHealthCheckHTTPValidateCertificate: "true",
				annotation.LBSvcHealthCheckHTTPStatusCodes:         "200,202",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(84),
				DestinationPort: hcloud.Int(8084),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
					Port:     hcloud.Int(8084),
					Interval: hcloud.Duration(time.Hour),
					Timeout:  hcloud.Duration(30 * time.Second),
					Retries:  hcloud.Int(5),
					HTTP: &hcloud.LoadBalancerAddServiceOptsHealthCheckHTTP{
						Domain:      hcloud.String("example.com"),
						Path:        hcloud.String("/internal/health"),
						TLS:         hcloud.Bool(true),
						StatusCodes: []string{"200", "202"},
					},
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8084),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
					Port:     hcloud.Int(8084),
					Interval: hcloud.Duration(time.Hour),
					Timeout:  hcloud.Duration(30 * time.Second),
					Retries:  hcloud.Int(5),
					HTTP: &hcloud.LoadBalancerUpdateServiceOptsHealthCheckHTTP{
						Domain:      hcloud.String("example.com"),
						Path:        hcloud.String("/internal/health"),
						TLS:         hcloud.Bool(true),
						StatusCodes: []string{"200", "202"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.certClient = &mocks.CertificateClient{}
			tt.certClient.Test(t)

			if tt.mock != nil {
				tt.mock(t, &tt)
			}

			builder := &hclbServiceOptsBuilder{
				Port: tt.servicePort,
				Service: &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID(tt.serviceUID),
					},
				},
				CertOps: &CertificateOps{CertClient: tt.certClient},
			}
			for k, v := range tt.serviceAnnotations {
				if err := k.AnnotateService(builder.Service, v); err != nil {
					t.Error(err)
				}
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
