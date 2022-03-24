package annotation_test

import (
	"net"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/assert"
	"github.com/syself/hetzner-cloud-controller-manager/internal/annotation"
	v1 "k8s.io/api/core/v1"
)

func TestLBToService_AddAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		svc      v1.Service
		lb       hcloud.LoadBalancer
		expected map[annotation.Name]interface{}
	}{
		{
			name: "tcp load balancer",
			svc: v1.Service{
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{{Port: 1234}},
				},
			},
			lb: hcloud.LoadBalancer{
				ID:               4711,
				Name:             "common annotations lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Algorithm:        hcloud.LoadBalancerAlgorithm{Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin},
				Location: &hcloud.Location{
					Name:        "fsn1",
					NetworkZone: hcloud.NetworkZoneEUCentral,
				},
				PublicNet: hcloud.LoadBalancerPublicNet{
					IPv4: hcloud.LoadBalancerPublicNetIPv4{
						IP: net.ParseIP("1.2.3.4"),
					},
					IPv6: hcloud.LoadBalancerPublicNetIPv6{
						IP: net.ParseIP("b196:ead5:f1e5:8c66:864c:6716:5450:891c"),
					},
				},
				Services: []hcloud.LoadBalancerService{
					{
						ListenPort:    1234,
						Protocol:      hcloud.LoadBalancerServiceProtocolTCP,
						Proxyprotocol: true,
						HealthCheck: hcloud.LoadBalancerServiceHealthCheck{
							Protocol: hcloud.LoadBalancerServiceProtocolTCP,
							Port:     2525,
							Interval: time.Hour,
							Timeout:  5 * time.Minute,
							Retries:  3,
						},
					},
				},
			},
			expected: map[annotation.Name]interface{}{
				annotation.LBID:            4711,
				annotation.LBName:          "common annotations lb",
				annotation.LBType:          "lb11",
				annotation.LBAlgorithmType: hcloud.LoadBalancerAlgorithmTypeRoundRobin,
				annotation.LBLocation:      "fsn1",
				annotation.LBNetworkZone:   hcloud.NetworkZoneEUCentral,
				annotation.LBPublicIPv4:    net.ParseIP("1.2.3.4"),
				annotation.LBPublicIPv6:    net.ParseIP("b196:ead5:f1e5:8c66:864c:6716:5450:891c"),

				annotation.LBSvcProtocol:            hcloud.LoadBalancerServiceProtocolTCP,
				annotation.LBSvcProxyProtocol:       true,
				annotation.LBSvcHealthCheckProtocol: hcloud.LoadBalancerServiceProtocolTCP,
				annotation.LBSvcHealthCheckPort:     2525,
				annotation.LBSvcHealthCheckInterval: time.Hour,
				annotation.LBSvcHealthCheckTimeout:  5 * time.Minute,
				annotation.LBSvcHealthCheckRetries:  3,
			},
		},
		{
			name: "http load balancer",
			svc: v1.Service{
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{{Port: 1235}},
				},
			},
			lb: hcloud.LoadBalancer{
				ID:               4712,
				Name:             "https load balancer",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Algorithm:        hcloud.LoadBalancerAlgorithm{Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin},
				Location: &hcloud.Location{
					Name:        "fsn1",
					NetworkZone: hcloud.NetworkZoneEUCentral,
				},
				PublicNet: hcloud.LoadBalancerPublicNet{
					IPv4: hcloud.LoadBalancerPublicNetIPv4{
						IP: net.ParseIP("1.2.3.4"),
					},
					IPv6: hcloud.LoadBalancerPublicNetIPv6{
						IP: net.ParseIP("b196:ead5:f1e5:8c66:864c:6716:5450:891c"),
					},
				},
				Services: []hcloud.LoadBalancerService{
					{
						ListenPort:    1235,
						Protocol:      hcloud.LoadBalancerServiceProtocolHTTPS,
						Proxyprotocol: true,
						HTTP: hcloud.LoadBalancerServiceHTTP{
							Certificates:   []*hcloud.Certificate{{ID: 3}, {ID: 5}},
							CookieName:     "TESTCOOKIE",
							CookieLifetime: time.Hour,
							RedirectHTTP:   true,
						},
						HealthCheck: hcloud.LoadBalancerServiceHealthCheck{
							Protocol: hcloud.LoadBalancerServiceProtocolHTTPS,
							Port:     2525,
							Interval: time.Hour,
							Timeout:  5 * time.Minute,
							Retries:  3,
							HTTP: &hcloud.LoadBalancerServiceHealthCheckHTTP{
								Domain:      "example.com",
								Path:        "/internal/health",
								TLS:         true,
								StatusCodes: []string{"200", "202"},
							},
						},
					},
				},
			},
			expected: map[annotation.Name]interface{}{
				annotation.LBID:            4712,
				annotation.LBName:          "https load balancer",
				annotation.LBType:          "lb11",
				annotation.LBAlgorithmType: hcloud.LoadBalancerAlgorithmTypeRoundRobin,
				annotation.LBLocation:      "fsn1",
				annotation.LBNetworkZone:   hcloud.NetworkZoneEUCentral,
				annotation.LBPublicIPv4:    net.ParseIP("1.2.3.4"),
				annotation.LBPublicIPv6:    net.ParseIP("b196:ead5:f1e5:8c66:864c:6716:5450:891c"),

				annotation.LBSvcProtocol:           hcloud.LoadBalancerServiceProtocolHTTPS,
				annotation.LBSvcProxyProtocol:      true,
				annotation.LBSvcHTTPCookieName:     "TESTCOOKIE",
				annotation.LBSvcHTTPCookieLifetime: time.Hour,
				annotation.LBSvcRedirectHTTP:       true,
				annotation.LBSvcHTTPCertificates:   []*hcloud.Certificate{{ID: 3}, {ID: 5}},

				annotation.LBSvcHealthCheckProtocol:                hcloud.LoadBalancerServiceProtocolHTTPS,
				annotation.LBSvcHealthCheckPort:                    2525,
				annotation.LBSvcHealthCheckInterval:                time.Hour,
				annotation.LBSvcHealthCheckTimeout:                 5 * time.Minute,
				annotation.LBSvcHealthCheckRetries:                 3,
				annotation.LBSvcHealthCheckHTTPDomain:              "example.com",
				annotation.LBSvcHealthCheckHTTPPath:                "/internal/health",
				annotation.LBSvcHealthCheckHTTPValidateCertificate: true,
				annotation.LBSvcHealthCheckHTTPStatusCodes:         []string{"200", "202"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := annotation.LBToService(&tt.svc, &tt.lb)
			assert.NoError(t, err)
			annotation.AssertServiceAnnotated(t, &tt.svc, tt.expected)
		})
	}
}
