/*
Copyright 2018 Hetzner Cloud GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/hetznercloud/hcloud-go/hcloud/schema"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

// TestInstances_InstanceExists also tests [lookupServer]. The other tests
// [instances] rely on these tests and only test their additional features.
func TestInstances_InstanceExists(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:   1,
				Name: "foobar",
			},
		})
	})
	env.Mux.HandleFunc("/servers/2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(schema.ErrorResponse{Error: schema.Error{Code: string(hcloud.ErrorCodeNotFound)}})
	})
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		var servers []schema.Server
		if r.URL.RawQuery == "name=foobar" {
			servers = append(servers, schema.Server{ID: 1, Name: "foobar"})
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{Servers: servers})
	})

	instances := newInstances(env.Client, AddressFamilyIPv4)

	tests := []struct {
		name     string
		node     *v1.Node
		expected bool
	}{
		{
			name: "existing server by id",
			node: &v1.Node{
				Spec: v1.NodeSpec{ProviderID: "hcloud://1"},
			},
			expected: true,
		}, {
			name: "missing server by id",
			node: &v1.Node{
				Spec: v1.NodeSpec{ProviderID: "hcloud://1"},
			},
			expected: true,
		}, {
			name: "existing server by name",
			node: &v1.Node{
				Spec: v1.NodeSpec{ProviderID: "hcloud://1"},
			},
			expected: true,
		}, {
			name: "missing server by name",
			node: &v1.Node{
				Spec: v1.NodeSpec{ProviderID: "hcloud://1"},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := instances.InstanceExists(context.TODO(), test.node)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if test.expected != exists {
				t.Fatalf("Expected server to exist %v but got %v", test.expected, exists)
			}
		})
	}
}

func TestInstances_InstanceShutdown(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:     1,
				Name:   "foobar",
				Status: string(hcloud.ServerStatusRunning),
			},
		})
	})
	env.Mux.HandleFunc("/servers/2", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:     2,
				Name:   "barfoo",
				Status: string(hcloud.ServerStatusOff),
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyIPv4)

	tests := []struct {
		name     string
		node     *v1.Node
		expected bool
	}{
		{
			name: "running server",
			node: &v1.Node{
				Spec: v1.NodeSpec{ProviderID: "hcloud://1"},
			},
			expected: false,
		}, {
			name: "shutdown server",
			node: &v1.Node{
				Spec: v1.NodeSpec{ProviderID: "hcloud://2"},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := instances.InstanceShutdown(context.TODO(), test.node)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if test.expected != exists {
				t.Fatalf("Expected server shutdown to be %v but got %v", test.expected, exists)
			}
		})
	}
}

func TestInstances_InstanceMetadata(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:         1,
				Name:       "foobar",
				ServerType: schema.ServerType{Name: "asdf11"},
				Datacenter: schema.Datacenter{Name: "Test DC", Location: schema.Location{Name: "Test Location"}},
				PublicNet: schema.ServerPublicNet{
					IPv6: schema.ServerPublicNetIPv6{
						IP: "2001:db8:1234::/64",
					},
					IPv4: schema.ServerPublicNetIPv4{
						IP: "203.0.113.7",
					},
				},
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyIPv4)

	metadata, err := instances.InstanceMetadata(context.TODO(), &v1.Node{
		Spec: v1.NodeSpec{ProviderID: "hcloud://1"},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedMetadata := &cloudprovider.InstanceMetadata{
		ProviderID:   "hcloud://1",
		InstanceType: "asdf11",
		NodeAddresses: []v1.NodeAddress{
			{Type: v1.NodeHostName, Address: "foobar"},
			{Type: v1.NodeExternalIP, Address: "203.0.113.7"},
		},
		Zone:   "Test DC",
		Region: "Test Location",
	}

	if !reflect.DeepEqual(metadata, expectedMetadata) {
		t.Fatalf("Expected metadata %+v but got %+v", *expectedMetadata, *metadata)
	}
}

func TestNodeAddressesByProviderID(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:   1,
				Name: "node15",
				PublicNet: schema.ServerPublicNet{
					IPv6: schema.ServerPublicNetIPv6{
						IP: "2001:db8:1234::/64",
					},
					IPv4: schema.ServerPublicNetIPv4{
						IP: "131.232.99.1",
					},
				},
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyIPv4)
	addr, err := instances.NodeAddressesByProviderID(context.TODO(), "hcloud://1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "node15" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "131.232.99.1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}
}

func TestNodeAddresses(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=node15" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "node15",
					PublicNet: schema.ServerPublicNet{
						IPv6: schema.ServerPublicNetIPv6{
							IP: "2001:db8:1234::",
						},
						IPv4: schema.ServerPublicNetIPv4{
							IP: "131.232.99.1",
						},
					},
				},
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyIPv4)
	addr, err := instances.NodeAddresses(context.TODO(), "node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "node15" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "131.232.99.1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}
}

func TestNodeAddressesIPv6(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=node15" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "node15",
					PublicNet: schema.ServerPublicNet{
						IPv6: schema.ServerPublicNetIPv6{
							IP: "2001:db8:1234::/64",
						},
						IPv4: schema.ServerPublicNetIPv4{
							IP: "131.232.99.1",
						},
					},
				},
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyIPv6)
	addr, err := instances.NodeAddresses(context.TODO(), "node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "node15" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "2001:db8:1234::1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}
}

func TestNodeAddressesDualStack(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=node15" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "node15",
					PublicNet: schema.ServerPublicNet{
						IPv6: schema.ServerPublicNetIPv6{
							IP: "2001:db8:1234::/64",
						},
						IPv4: schema.ServerPublicNetIPv4{
							IP: "131.232.99.1",
						},
					},
				},
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyDualStack)
	addr, err := instances.NodeAddresses(context.TODO(), "node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 3 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "node15" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "131.232.99.1" ||
		addr[2].Type != v1.NodeExternalIP || addr[2].Address != "2001:db8:1234::1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}
}

func TestExternalID(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=node15" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID: 1,
				},
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyIPv4)
	id, err := instances.ExternalID(context.TODO(), "node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != "1" {
		t.Errorf("Unexpected id: %v", id)
	}
}

func TestInstanceType(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=node15" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID: 1,
					ServerType: schema.ServerType{
						Name: "cx11",
					},
				},
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyIPv4)
	serverType, err := instances.InstanceType(context.TODO(), "node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if serverType != "cx11" {
		t.Errorf("Unexpected server type: %v", serverType)
	}
}

func TestInstanceTypeByProviderID(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:   1,
				Name: "node15",
				ServerType: schema.ServerType{
					Name: "cx11",
				},
			},
		})
	})

	instances := newInstances(env.Client, AddressFamilyIPv4)
	instanceType, err := instances.InstanceTypeByProviderID(context.TODO(), "hcloud://1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if instanceType != "cx11" {
		t.Errorf("Unexpected instance type: %v", instanceType)
	}
}

func TestInstanceExistsByProviderID(t *testing.T) {
	t.Run("Found", func(t *testing.T) {
		env := newTestEnv()
		defer env.Teardown()
		env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(schema.ServerGetResponse{
				Server: schema.Server{
					ID:   1,
					Name: "node15",
					ServerType: schema.ServerType{
						Name: "cx11",
					},
				},
			})
		})

		instances := newInstances(env.Client, AddressFamilyIPv4)
		exists, err := instances.InstanceExistsByProviderID(context.TODO(), "hcloud://1")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !exists {
			t.Errorf("Unexpected exist state: %v", exists)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		env := newTestEnv()
		defer env.Teardown()
		env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(schema.ErrorResponse{
				Error: schema.Error{
					Code: string(hcloud.ErrorCodeNotFound),
				},
			})
		})

		instances := newInstances(env.Client, AddressFamilyIPv4)
		exists, err := instances.InstanceExistsByProviderID(context.TODO(), "hcloud://1")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if exists {
			t.Errorf("Unexpected exist state: %v", exists)
		}
	})
}

func TestInstanceShutdownByProviderID(t *testing.T) {
	t.Run("Shutdown", func(t *testing.T) {
		env := newTestEnv()
		defer env.Teardown()

		env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(schema.ServerGetResponse{
				Server: schema.Server{
					Status: string(hcloud.ServerStatusOff),
				},
			})
		})

		instances := newInstances(env.Client, AddressFamilyIPv4)
		isOff, err := instances.InstanceShutdownByProviderID(context.TODO(), "hcloud://1")
		if !isOff {
			t.Errorf("Unexpected isOff state: %v", isOff)
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("NotShutdown", func(t *testing.T) {
		env := newTestEnv()
		defer env.Teardown()

		env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(schema.ServerGetResponse{
				Server: schema.Server{
					Status: string(hcloud.ServerStatusRunning),
				},
			})
		})

		instances := newInstances(env.Client, AddressFamilyIPv4)
		isOff, err := instances.InstanceShutdownByProviderID(context.TODO(), "hcloud://1")
		if isOff {
			t.Errorf("Unexpected isOff state: %v", isOff)
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		env := newTestEnv()
		defer env.Teardown()

		env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(schema.ErrorResponse{
				Error: schema.Error{
					Code: string(hcloud.ErrorCodeNotFound),
				},
			})
		})

		instances := newInstances(env.Client, AddressFamilyIPv4)
		isOff, err := instances.InstanceShutdownByProviderID(context.TODO(), "hcloud://1")
		if isOff {
			t.Errorf("Unexpected isOff state: %v", isOff)
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

func TestCurrentNodeName(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	instances := newInstances(env.Client, AddressFamilyIPv4)
	nodeName, err := instances.CurrentNodeName(context.TODO(), "hostname")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if nodeName != "hostname" {
		t.Errorf("Unexpected node name: %s", nodeName)
	}
}
