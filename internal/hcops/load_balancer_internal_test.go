package hcops

import (
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestHCLBServiceOptsBuilder(t *testing.T) {
	tests := []struct {
		name               string
		servicePort        v1.ServicePort
		serviceAnnotations map[annotation.Name]interface{}
		expectedAddOpts    hcloud.LoadBalancerAddServiceOpts
		expectedUpdateOpts hcloud.LoadBalancerUpdateServiceOpts
	}{
		{
			name:        "defaults",
			servicePort: v1.ServicePort{Port: 80, NodePort: 8080},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(80),
				DestinationPort: hcloud.Int(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				Proxyprotocol:   hcloud.Bool(false),
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8080),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				Proxyprotocol:   hcloud.Bool(false),
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
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8081),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				Proxyprotocol:   hcloud.Bool(true),
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
				Proxyprotocol:   hcloud.Bool(false),
				HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
					CookieName:     hcloud.String("my-cookie"),
					CookieLifetime: hcloud.Duration(time.Hour),
					Certificates:   []*hcloud.Certificate{{ID: 1}, {ID: 3}},
					RedirectHTTP:   hcloud.Bool(true),
					StickySessions: hcloud.Bool(true),
				},
			},
			expectedUpdateOpts: hcloud.LoadBalancerUpdateServiceOpts{
				DestinationPort: hcloud.Int(8082),
				Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
				Proxyprotocol:   hcloud.Bool(false),
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					CookieName:     hcloud.String("my-cookie"),
					CookieLifetime: hcloud.Duration(time.Hour),
					Certificates:   []*hcloud.Certificate{{ID: 1}, {ID: 3}},
					RedirectHTTP:   hcloud.Bool(true),
					StickySessions: hcloud.Bool(true),
				},
			},
		},
		{
			name:        "add TCP health check",
			servicePort: v1.ServicePort{Port: 83, NodePort: 8083},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcHealthCheckProtocol: string(hcloud.LoadBalancerServiceProtocolTCP),
				annotation.LBSvcHealthCheckPort:     "8084",
				annotation.LBSvcHealthCheckInterval: "3600",
				annotation.LBSvcHealthCheckTimeout:  "30",
				annotation.LBSvcHealthCheckRetries:  "5",
			},
			expectedAddOpts: hcloud.LoadBalancerAddServiceOpts{
				ListenPort:      hcloud.Int(83),
				DestinationPort: hcloud.Int(8083),
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				Proxyprotocol:   hcloud.Bool(false),
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
				Proxyprotocol:   hcloud.Bool(false),
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
				annotation.LBSvcHealthCheckInterval:                3600,
				annotation.LBSvcHealthCheckTimeout:                 30,
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
				Proxyprotocol:   hcloud.Bool(false),
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
				Proxyprotocol:   hcloud.Bool(false),
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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			builder := &hclbServiceOptsBuilder{
				Port:    tt.servicePort,
				Service: &v1.Service{},
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
