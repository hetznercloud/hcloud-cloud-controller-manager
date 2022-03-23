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
	"strings"
	"testing"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/hetznercloud/hcloud-go/hcloud/schema"
	"github.com/syself/hrobot-go/models"
	v1 "k8s.io/api/core/v1"
)

func TestNodeAddressesByProviderID(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:   1,
				Name: "hcloud//node15",
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

	env.Mux.HandleFunc("/robot/server/321", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(models.ServerResponse{
			Server: models.Server{
				ServerIP:      "123.123.123.123",
				ServerIPv6Net: "2a01:f48:111:4221::",
				ServerNumber:  321,
				Name:          "robot//server1",
			},
		})
	})

	instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
	addr, err := instances.NodeAddressesByProviderID(context.TODO(), "hcloud://1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "hcloud//node15" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "131.232.99.1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}

	addr, err = instances.NodeAddressesByProviderID(context.TODO(), "hetzner://321")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "robot//server1" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "123.123.123.123" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}
}

func TestNodeAddresses(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=hcloud%2F%2Fnode15" || strings.HasPrefix(r.URL.Path, "/robot") {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "hcloud//node15",
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

	env.Mux.HandleFunc("/robot/server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ServerResponse{
			{
				Server: models.Server{
					ServerIP:      "123.123.123.123",
					ServerIPv6Net: "2a01:f48:111:4221::",
					ServerNumber:  321,
					Name:          "robot//server1",
				},
			},
		})
	})

	instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
	addr, err := instances.NodeAddresses(context.TODO(), "hcloud//node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "hcloud//node15" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "131.232.99.1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}

	addr, err = instances.NodeAddresses(context.TODO(), "robot//server1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "robot//server1" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "123.123.123.123" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}
}

func TestNodeAddressesIPv6(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=hcloud%2F%2Fnode15" || strings.HasPrefix(r.URL.Path, "/robot") {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "hcloud//node15",
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

	env.Mux.HandleFunc("/robot/server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ServerResponse{
			{
				Server: models.Server{
					ServerIP:      "123.123.123.123",
					ServerIPv6Net: "2a01:f48:111:4221::",
					ServerNumber:  321,
					Name:          "robot//server1",
				},
			},
		})
	})

	instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv6)
	addr, err := instances.NodeAddresses(context.TODO(), "hcloud//node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "hcloud//node15" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "2001:db8:1234::1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}

	addr, err = instances.NodeAddresses(context.TODO(), "robot//server1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 2 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "robot//server1" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "2a01:f48:111:4221::1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}
}

func TestNodeAddressesDualStack(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=hcloud%2F%2Fnode15" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "hcloud//node15",
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

	env.Mux.HandleFunc("/robot/server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ServerResponse{
			{
				Server: models.Server{
					ServerIP:      "123.123.123.123",
					ServerIPv6Net: "2a01:f48:111:4221::",
					ServerNumber:  321,
					Name:          "robot//server1",
				},
			},
		})
	})

	instances := newInstances(env.Client, env.RobotClient, AddressFamilyDualStack)
	addr, err := instances.NodeAddresses(context.TODO(), "hcloud//node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 3 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "hcloud//node15" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "2001:db8:1234::1" ||
		addr[2].Type != v1.NodeExternalIP || addr[2].Address != "131.232.99.1" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}

	addr, err = instances.NodeAddresses(context.TODO(), "robot//server1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(addr) != 3 ||
		addr[0].Type != v1.NodeHostName || addr[0].Address != "robot//server1" ||
		addr[1].Type != v1.NodeExternalIP || addr[1].Address != "2a01:f48:111:4221::1" ||
		addr[2].Type != v1.NodeExternalIP || addr[2].Address != "123.123.123.123" {
		t.Errorf("Unexpected node addresses: %v", addr)
	}
}

func TestExternalID(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=hcloud%2F%2Fnode15" {
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

	env.Mux.HandleFunc("/robot/server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ServerResponse{
			{
				Server: models.Server{
					ServerNumber: 1,
					Name:         "robot//server1",
				},
			},
		})
	})

	instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
	id, err := instances.ExternalID(context.TODO(), "hcloud//node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != "1" {
		t.Errorf("Unexpected id: %v", id)
	}

	id, err = instances.ExternalID(context.TODO(), "robot//server1")
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
		if r.URL.RawQuery != "name=hcloud%2F%2Fnode15" {
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

	env.Mux.HandleFunc("/robot/server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ServerResponse{
			{
				Server: models.Server{
					ServerNumber: 1,
					Product:      "dedicated_server",
					Name:         "robot//server1",
				},
			},
		})
	})

	instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
	serverType, err := instances.InstanceType(context.TODO(), "hcloud//node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if serverType != "cx11" {
		t.Errorf("Unexpected server type: %v", serverType)
	}

	serverType, err = instances.InstanceType(context.TODO(), "robot//server1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if serverType != "dedicated_server" {
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
				Name: "hcloud//node15",
				ServerType: schema.ServerType{
					Name: "cx11",
				},
			},
		})
	})

	env.Mux.HandleFunc("/robot/server/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(models.ServerResponse{
			Server: models.Server{
				ServerNumber: 1,
				Product:      "dedicated_server",
				Name:         "robot//server1",
			},
		})
	})

	instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
	instanceType, err := instances.InstanceTypeByProviderID(context.TODO(), "hcloud://1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if instanceType != "cx11" {
		t.Errorf("Unexpected instance type: %v", instanceType)
	}

	instanceType, err = instances.InstanceTypeByProviderID(context.TODO(), "hetzner://1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if instanceType != "dedicated_server" {
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
					Name: "hcloud//node15",
					ServerType: schema.ServerType{
						Name: "cx11",
					},
				},
			})
		})

		env.Mux.HandleFunc("/robot/server/1", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(models.ServerResponse{
				Server: models.Server{
					ServerNumber: 1,
					Product:      "dedicated_server",
					Name:         "robot//server1",
				},
			})
		})

		instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
		exists, err := instances.InstanceExistsByProviderID(context.TODO(), "hcloud://1")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !exists {
			t.Errorf("Unexpected exist state: %v", exists)
		}

		exists, err = instances.InstanceExistsByProviderID(context.TODO(), "hetzner://1")
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

		env.Mux.HandleFunc("/robot/server/1", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(schema.ErrorResponse{
				Error: schema.Error{
					Code: string(models.ErrorCodeNotFound),
				},
			})
		})

		instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
		exists, err := instances.InstanceExistsByProviderID(context.TODO(), "hcloud://1")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if exists {
			t.Errorf("Unexpected exist state: %v", exists)
		}

		exists, err = instances.InstanceExistsByProviderID(context.TODO(), "hetzner://1")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if exists {
			t.Errorf("Unexpected exist state: %v", exists)
		}

	})

	t.Run("Dedicated server found, but not in cluster", func(t *testing.T) {
		env := newTestEnv()
		defer env.Teardown()

		env.Mux.HandleFunc("/robot/server/1", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(models.ServerResponse{
				Server: models.Server{
					ServerNumber: 1,
					Product:      "dedicated_server",
					Name:         "server1",
				},
			})
		})

		instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
		exists, err := instances.InstanceExistsByProviderID(context.TODO(), "hetzner://1")
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

		env.Mux.HandleFunc("/robot/server/1", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(models.ServerResponse{
				Server: models.Server{
					ServerNumber: 1,
					Product:      "dedicated_server",
					Name:         "server1",
				},
			})
		})

		instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
		isOff, err := instances.InstanceShutdownByProviderID(context.TODO(), "hcloud://1")
		if !isOff {
			t.Errorf("Unexpected isOff state: %v", isOff)
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Dedicated servers are never shut down
		isOff, err = instances.InstanceShutdownByProviderID(context.TODO(), "hetzner://1")
		if isOff {
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

		instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
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

		instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
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
	instances := newInstances(env.Client, env.RobotClient, AddressFamilyIPv4)
	nodeName, err := instances.CurrentNodeName(context.TODO(), "hostname")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if nodeName != "hostname" {
		t.Errorf("Unexpected node name: %s", nodeName)
	}
}
