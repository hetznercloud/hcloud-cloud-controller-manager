package hcloud

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/hetznercloud/hcloud-go/hcloud/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLoadBalancers_GetLoadBalancer(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/load_balancers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=afoobar" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.LoadBalancerListResponse{
			LoadBalancers: []schema.LoadBalancer{
				{
					ID:   1,
					Name: "afoobar",
					PublicNet: schema.LoadBalancerPublicNet{
						Enabled: true,
						IPv4: schema.LoadBalancerPublicNetIPv4{
							IP: "127.0.0.1",
						},
						// IPv6: schema.LoadBalancerPublicNetIPv6{
						// 	IP: "::1",
						// },
					},
				},
			},
		})
	})

	lbOps := &hcops.LoadBalancerOps{
		LBClient:     &env.Client.LoadBalancer,
		ActionClient: &env.Client.Action,
	}
	loadBalancers := newLoadBalancers(lbOps, &env.Client.LoadBalancer, &env.Client.Action)
	status, exists, err := loadBalancers.GetLoadBalancer(context.Background(), "my-cluster", &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:         "foobar",
			Annotations: map[string]string{},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol: "TCP",
					Port:     int32(80),
					NodePort: int32(8080),
				},
				{
					Protocol: "TCP",
					Port:     int32(443),
					NodePort: int32(8080),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !exists {
		t.Fatalf("Unexpected exists: %v", exists)
	}
	if status == nil {
		t.Fatalf("Unexpected status: %v", status)
	}

	if len(status.Ingress) != 1 {
		t.Fatalf("Unexpected status.Ingress len: %v", len(status.Ingress))
	}
	if status.Ingress[0].IP != "127.0.0.1" {
		t.Fatalf("Unexpected status.Ingress[0].IP: %v", status.Ingress[0].IP)
	}
	// if status.Ingress[1].IP != "::1" {
	// 	t.Fatalf("Unexpected status.Ingress[1].IP: %v", status.Ingress[1].IP)
	// }
}

func TestLoadBalancers_GetLoadBalancerHostnameAnnotation(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/load_balancers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=afoobar" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.LoadBalancerListResponse{
			LoadBalancers: []schema.LoadBalancer{
				{
					ID:   1,
					Name: "afoobar",
					PublicNet: schema.LoadBalancerPublicNet{
						Enabled: true,
						IPv4: schema.LoadBalancerPublicNetIPv4{
							IP: "127.0.0.1",
						},
						// IPv6: schema.LoadBalancerPublicNetIPv6{
						// 	IP: "::1",
						// },
					},
				},
			},
		})
	})

	lbOps := &hcops.LoadBalancerOps{
		LBClient:     &env.Client.LoadBalancer,
		ActionClient: &env.Client.Action,
	}
	loadBalancers := newLoadBalancers(lbOps, &env.Client.LoadBalancer, &env.Client.Action)
	status, exists, err := loadBalancers.GetLoadBalancer(context.Background(), "my-cluster", &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID: "foobar",
			Annotations: map[string]string{
				string(annotation.LBHostname): "example.org",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol: "TCP",
					Port:     int32(80),
					NodePort: int32(8080),
				},
				{
					Protocol: "TCP",
					Port:     int32(443),
					NodePort: int32(8080),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !exists {
		t.Fatalf("Unexpected exists: %v", exists)
	}
	if status == nil {
		t.Fatalf("Unexpected status: %v", status)
	}

	if len(status.Ingress) != 1 {
		t.Fatalf("Unexpected status.Ingress len: %v", len(status.Ingress))
	}
	if status.Ingress[0].Hostname != "example.org" {
		t.Fatalf("Unexpected status.Ingress[0].Hostname: %v", status.Ingress[0].Hostname)
	}
	if status.Ingress[0].IP != "" {
		t.Fatalf("Unexpected status.Ingress[0].IP: %v", status.Ingress[0].IP)
	}
}

func TestLoadBalancers_EnsureLoadBalancerDeleted(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/load_balancers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=afoobar" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.LoadBalancerListResponse{
			LoadBalancers: []schema.LoadBalancer{
				{
					ID:   1,
					Name: "afoobar",
					PublicNet: schema.LoadBalancerPublicNet{
						Enabled: true,
						IPv4: schema.LoadBalancerPublicNetIPv4{
							IP: "127.0.0.1",
						},
						// IPv6: schema.LoadBalancerPublicNetIPv6{
						// 	IP: "::1",
						// },
					},
				},
			},
		})
	})
	env.Mux.HandleFunc("/load_balancers/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("wrong http method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(schema.LoadBalancerDeleteServiceResponse{
			Action: schema.Action{
				ID:       1,
				Progress: 0,
				Status:   string(hcloud.ActionStatusRunning),
			},
		})
	})

	lbOps := &hcops.LoadBalancerOps{
		LBClient:     &env.Client.LoadBalancer,
		ActionClient: &env.Client.Action,
	}
	loadBalancers := newLoadBalancers(lbOps, &env.Client.LoadBalancer, &env.Client.Action)
	err := loadBalancers.EnsureLoadBalancerDeleted(context.Background(), "my-cluster", &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:         "foobar",
			Annotations: map[string]string{},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol: "TCP",
					Port:     int32(80),
					NodePort: int32(8080),
				},
				{
					Protocol: "TCP",
					Port:     int32(443),
					NodePort: int32(8080),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestLoadBalancers_EnsureLoadBalancerDeletedWithProtectection(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/load_balancers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=afoobar" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.LoadBalancerListResponse{
			LoadBalancers: []schema.LoadBalancer{
				{
					ID:   1,
					Name: "afoobar",
					PublicNet: schema.LoadBalancerPublicNet{
						Enabled: true,
						IPv4: schema.LoadBalancerPublicNetIPv4{
							IP: "127.0.0.1",
						},
						// IPv6: schema.LoadBalancerPublicNetIPv6{
						// 	IP: "::1",
						// },
					},
					Protection: schema.LoadBalancerProtection{Delete: true},
				},
			},
		})
	})
	env.Mux.HandleFunc("/load_balancers/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("wrong http method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(schema.LoadBalancerDeleteServiceResponse{
			Action: schema.Action{
				ID:       1,
				Progress: 0,
				Status:   string(hcloud.ActionStatusRunning),
			},
		})
	})

	lbOps := &hcops.LoadBalancerOps{
		LBClient:     &env.Client.LoadBalancer,
		ActionClient: &env.Client.Action,
	}
	loadBalancers := newLoadBalancers(lbOps, &env.Client.LoadBalancer, &env.Client.Action)
	err := loadBalancers.EnsureLoadBalancerDeleted(context.Background(), "my-cluster", &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:         "foobar",
			Annotations: map[string]string{},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol: "TCP",
					Port:     int32(80),
					NodePort: int32(8080),
				},
				{
					Protocol: "TCP",
					Port:     int32(443),
					NodePort: int32(8080),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestLoadBalancers_EnsureLoadBalancerCreate(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	var services []schema.LoadBalancerService
	env.Mux.HandleFunc("/actions/13", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(schema.ActionGetResponse{
				Action: schema.Action{
					ID:       13,
					Status:   "success",
					Command:  "create_load_balancer",
					Progress: 100,
				},
			})
		}
	})
	env.Mux.HandleFunc("/load_balancers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "GET" {
			if r.URL.RawQuery != "name=afoobar" {
				t.Fatal("missing name query")
			}
			json.NewEncoder(w).Encode(schema.LoadBalancerListResponse{
				LoadBalancers: []schema.LoadBalancer{},
			})
		} else if r.Method == "POST" {
			json.NewEncoder(w).Encode(schema.LoadBalancerCreateResponse{
				LoadBalancer: schema.LoadBalancer{
					ID:   5,
					Name: "afoobar",
					PublicNet: schema.LoadBalancerPublicNet{
						Enabled: true,
						IPv4: schema.LoadBalancerPublicNetIPv4{
							IP: "127.0.0.1",
						},
						// IPv6: schema.LoadBalancerPublicNetIPv6{
						// 	IP: "::1",
						// },
					},
					LoadBalancerType: schema.LoadBalancerType{Name: "lb11"},
					Location:         schema.Location{Name: "nbg"},
					Services:         services,
				},
				Action: schema.Action{
					ID:       13,
					Status:   "success",
					Command:  "create_load_balancer",
					Progress: 100,
				},
			})
		}
	})
	env.Mux.HandleFunc("/load_balancers/5", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(schema.LoadBalancerGetResponse{
				LoadBalancer: schema.LoadBalancer{
					ID:   5,
					Name: "afoobar",
					PublicNet: schema.LoadBalancerPublicNet{
						Enabled: true,
						IPv4: schema.LoadBalancerPublicNetIPv4{
							IP: "127.0.0.1",
						},
						// IPv6: schema.LoadBalancerPublicNetIPv6{
						// 	IP: "::1",
						// },
					},
					LoadBalancerType: schema.LoadBalancerType{Name: "lb11"},
					Location:         schema.Location{Name: "nbg"},
					Services:         services,
				},
			})
		}
	})

	env.Mux.HandleFunc("/load_balancers/5/actions/add_service", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "POST" {
			services = append(services, schema.LoadBalancerService{
				Protocol:        "tcp",
				ListenPort:      80,
				DestinationPort: 8080,
				Proxyprotocol:   false,
				HTTP: &schema.LoadBalancerServiceHTTP{
					CookieName:     "",
					CookieLifetime: 0,
					Certificates:   nil,
					RedirectHTTP:   false,
				},
				HealthCheck: &schema.LoadBalancerServiceHealthCheck{
					Protocol: "tcp",
					Port:     8080,
					Interval: 15,
					Timeout:  10,
					Retries:  3,
				},
			})
			json.NewEncoder(w).Encode(schema.LoadBalancerActionAddServiceResponse{
				Action: schema.Action{
					ID:       13,
					Status:   "success",
					Command:  "add_service",
					Progress: 100,
				},
			})
		}
	})

	lbOps := &hcops.LoadBalancerOps{
		LBClient:     &env.Client.LoadBalancer,
		ActionClient: &env.Client.Action,
	}
	loadBalancers := newLoadBalancers(lbOps, &env.Client.LoadBalancer, &env.Client.Action)
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID: "foobar",
			Annotations: map[string]string{
				string(annotation.LBLocation): "nbg1",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol: "TCP",
					Port:     int32(80),
					NodePort: int32(8080),
				},
			},
		},
	}
	status, err := loadBalancers.EnsureLoadBalancer(context.Background(), "my-cluster", service, []*v1.Node{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if status == nil {
		t.Fatalf("Unexpected status: %v", status)
	}

	if len(status.Ingress) != 1 {
		t.Fatalf("Unexpected status.Ingress len: %v", len(status.Ingress))
	}
	if status.Ingress[0].IP != "127.0.0.1" {
		t.Fatalf("Unexpected status.Ingress[0].IP: %v", status.Ingress[0].IP)
	}
	// if status.Ingress[1].IP != "::1" {
	// 	t.Fatalf("Unexpected status.Ingress[1].IP: %v", status.Ingress[1].IP)
	// }

	annotation.AssertServiceAnnotated(t, service, map[annotation.Name]interface{}{
		annotation.LBID:       5,
		annotation.LBName:     "afoobar",
		annotation.LBType:     "lb11",
		annotation.LBLocation: "nbg",
	})
}

func TestLoadBalancers_EnsureLoadBalancerUpdate(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	services := []schema.LoadBalancerService{
		{
			Protocol:        "tcp",
			ListenPort:      80,
			DestinationPort: 8080,
			Proxyprotocol:   false,
			HTTP: &schema.LoadBalancerServiceHTTP{
				CookieName:     "",
				CookieLifetime: 0,
				Certificates:   nil,
				RedirectHTTP:   false,
			},
			HealthCheck: &schema.LoadBalancerServiceHealthCheck{
				Protocol: "tcp",
				Port:     8080,
				Interval: 15,
				Timeout:  10,
				Retries:  3,
			},
		},
	}
	var loadBalancer *schema.LoadBalancer
	env.Mux.HandleFunc("/actions/13", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(schema.ActionGetResponse{
				Action: schema.Action{
					ID:       13,
					Status:   "success",
					Command:  "create_load_balancer",
					Progress: 100,
				},
			})
		}
	})
	env.Mux.HandleFunc("/load_balancers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "GET" {
			if r.URL.RawQuery != "name=my-loadbalancer" {
				t.Fatal("missing name query")
			}
			var loadBalancers []schema.LoadBalancer
			if loadBalancer != nil {
				loadBalancers = append(loadBalancers, *loadBalancer)
			}
			json.NewEncoder(w).Encode(schema.LoadBalancerListResponse{
				LoadBalancers: loadBalancers,
			})
		} else if r.Method == "POST" {
			loadBalancer = &schema.LoadBalancer{
				ID:   5,
				Name: "my-loadbalancer",
				PublicNet: schema.LoadBalancerPublicNet{
					Enabled: true,
					IPv4: schema.LoadBalancerPublicNetIPv4{
						IP: "127.0.0.1",
					},
					// IPv6: schema.LoadBalancerPublicNetIPv6{
					// 	IP: "::1",
					// },
				},
				LoadBalancerType: schema.LoadBalancerType{Name: "lb11"},
				Location:         schema.Location{Name: "nbg"},
				Services:         services,
			}
			json.NewEncoder(w).Encode(schema.LoadBalancerCreateResponse{
				LoadBalancer: *loadBalancer,
				Action: schema.Action{
					ID:       13,
					Status:   "success",
					Command:  "create_load_balancer",
					Progress: 100,
				},
			})
		}
	})
	env.Mux.HandleFunc("/load_balancers/5", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(schema.LoadBalancerGetResponse{
				LoadBalancer: schema.LoadBalancer{
					ID:   5,
					Name: "my-loadbalancer",
					PublicNet: schema.LoadBalancerPublicNet{
						Enabled: true,
						IPv4: schema.LoadBalancerPublicNetIPv4{
							IP: "127.0.0.1",
						},
						// IPv6: schema.LoadBalancerPublicNetIPv6{
						// 	IP: "::1",
						// },
					},
					LoadBalancerType: schema.LoadBalancerType{Name: "lb11"},
					Location:         schema.Location{Name: "nbg"},
					Services:         services,
				},
			})
		}
	})
	env.Mux.HandleFunc("/load_balancers/5/actions/update_service", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method == "POST" {
			services = append(services, schema.LoadBalancerService{
				Protocol:        "tcp",
				ListenPort:      80,
				DestinationPort: 123,
				Proxyprotocol:   false,
				HTTP: &schema.LoadBalancerServiceHTTP{
					CookieName:     "",
					CookieLifetime: 0,
					Certificates:   nil,
					RedirectHTTP:   false,
				},
				HealthCheck: &schema.LoadBalancerServiceHealthCheck{
					Protocol: "tcp",
					Port:     8080,
					Interval: 15,
					Timeout:  10,
					Retries:  3,
				},
			})
			json.NewEncoder(w).Encode(schema.LoadBalancerActionAddServiceResponse{
				Action: schema.Action{
					ID:       13,
					Status:   "success",
					Command:  "add_service",
					Progress: 100,
				},
			})
		}
	})

	lbOps := &hcops.LoadBalancerOps{
		LBClient:     &env.Client.LoadBalancer,
		ActionClient: &env.Client.Action,
	}
	loadBalancers := newLoadBalancers(lbOps, &env.Client.LoadBalancer, &env.Client.Action)
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID: "foobar",
			Annotations: map[string]string{
				string(annotation.LBLocation): "nbg1",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol: "TCP",
					Port:     int32(80),
					NodePort: int32(8080),
				},
			},
		},
	}
	annotation.LBName.AnnotateService(service, "my-loadbalancer")

	status, err := loadBalancers.EnsureLoadBalancer(context.Background(), "my-cluster", service, []*v1.Node{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if status == nil {
		t.Fatalf("Unexpected status: %v", status)
	}

	if len(status.Ingress) != 1 {
		t.Fatalf("Unexpected status.Ingress len: %v", len(status.Ingress))
	}
	if status.Ingress[0].IP != "127.0.0.1" {
		t.Fatalf("Unexpected status.Ingress[0].IP: %v", status.Ingress[0].IP)
	}
	// if status.Ingress[1].IP != "::1" {
	// 	t.Fatalf("Unexpected status.Ingress[1].IP: %v", status.Ingress[1].IP)
	// }

	annotation.AssertServiceAnnotated(t, service, map[annotation.Name]interface{}{
		annotation.LBID:       5,
		annotation.LBName:     "my-loadbalancer",
		annotation.LBType:     "lb11",
		annotation.LBLocation: "nbg",
	})
}

func TestLoadBalancers_EnsureLoadBalancer_CreateLoadBalancer(t *testing.T) {
	testErr := errors.New("test error")
	tests := []LoadBalancerTestCase{
		{
			Name: "check for existing Load Balancer fails",
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByName", tt.Ctx, mock.AnythingOfType("string")).
					Return(nil, testErr)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				if !errors.Is(err, testErr) {
					t.Errorf("expected error %v; got %v", testErr, err)
				}
			},
		},
		{
			Name: "public network only",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "pub-net-only",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				Name:             "pub-net-only",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv4:    hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("1.2.3.4")},
					// IPv6:    hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("fe80::1")},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByName", tt.Ctx, mock.AnythingOfType("string")).
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("Create", tt.Ctx, tt.LB.Name, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)

			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				expected := &v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{IP: tt.LB.PublicNet.IPv4.IP.String()},
						// {IP: tt.LB.PublicNet.IPv6.IP.String()},
					},
				}
				lbStat, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
				assert.Equal(t, expected, lbStat)
			},
		},
		{
			Name:      "attach Load Balancer to public and private network",
			NetworkID: 4711,
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "with-priv-net",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				Name:             "with-priv-net",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv4:    hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("1.2.3.4")},
					// IPv6:    hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("fe80::1")},
				},
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{
							ID:   4711,
							Name: "priv-net",
						},
						IP: net.ParseIP("10.10.10.2"),
					},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByName", tt.Ctx, mock.AnythingOfType("string")).
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("Create", tt.Ctx, tt.LB.Name, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)

			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				expected := &v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{IP: tt.LB.PublicNet.IPv4.IP.String()},
						// {IP: tt.LB.PublicNet.IPv6.IP.String()},
						{IP: tt.LB.PrivateNet[0].IP.String()},
					},
				}
				lbStat, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
				assert.Equal(t, expected, lbStat)
			},
		},
		{
			Name:      "disable private ingress",
			NetworkID: 4711,
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName:                  "with-priv-net-no-priv-ingress",
				annotation.LBDisablePrivateIngress: "true",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				Name:             "with-priv-net-no-priv-ingress",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv4:    hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("1.2.3.4")},
					// IPv6:    hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("fe80::1")},
				},
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{
							ID:   4711,
							Name: "priv-net",
						},
						IP: net.ParseIP("10.10.10.2"),
					},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByName", tt.Ctx, mock.AnythingOfType("string")).
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("Create", tt.Ctx, tt.LB.Name, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)

			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				expected := &v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{IP: tt.LB.PublicNet.IPv4.IP.String()},
						// {IP: tt.LB.PublicNet.IPv6.IP.String()},
					},
				}
				lbStat, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
				assert.Equal(t, expected, lbStat)
			},
		},
		{
			Name:      "attach Load Balancer to private network only",
			NetworkID: 4711,
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName:                 "priv-net-only",
				annotation.LBDisablePublicNetwork: true,
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				Name:             "priv-net-only",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{
							ID:   4711,
							Name: "priv-net",
						},
						IP: net.ParseIP("10.10.10.2"),
					},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByName", tt.Ctx, mock.AnythingOfType("string")).
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("Create", tt.Ctx, tt.LB.Name, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				expected := &v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{IP: tt.LB.PrivateNet[0].IP.String()},
					},
				}
				lbStat, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
				assert.Equal(t, expected, lbStat)
			},
		},
	}

	RunLoadBalancerTests(t, tests)
}

func TestLoadBalancer_EnsureLoadBalancer_UpdateLoadBalancer(t *testing.T) {
	tests := []LoadBalancerTestCase{
		{
			Name: "Load balancer unchanged",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByName", tt.Ctx, "test-lb").Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name: "Load balancer changed",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               2,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByName", tt.Ctx, "test-lb").Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(true, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("GetByID", tt.Ctx, tt.LB.ID).Times(1).Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name: "Load balancer targets changed",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               3,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByName", tt.Ctx, "test-lb").Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(true, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("GetByID", tt.Ctx, tt.LB.ID).Times(1).Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name: "Load balancer services changed",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               4,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByName", tt.Ctx, "test-lb").Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(true, nil)
				tt.LBOps.On("GetByID", tt.Ctx, tt.LB.ID).Times(1).Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, tt.run)
	}
}

func TestLoadBalancer_UpdateLoadBalancer(t *testing.T) {
	tests := []LoadBalancerTestCase{
		{
			Name: "Load Balancer not found",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByName", tt.Ctx, "test-lb").Return(nil, hcops.ErrNotFound)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.UpdateLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name: "calls all reconcilement ops",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByName", tt.Ctx, "test-lb").Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.UpdateLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, tt.run)
	}
}
