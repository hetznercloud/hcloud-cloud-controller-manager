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
	"net"
	"net/http"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cloudprovider "k8s.io/cloud-provider"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
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

	instances := newInstances(env.Client, config.AddressFamilyIPv4, 0)

	tests := []struct {
		name     string
		node     *corev1.Node
		expected bool
	}{
		{
			name: "existing server by id",
			node: &corev1.Node{
				Spec: corev1.NodeSpec{ProviderID: "hcloud://1"},
			},
			expected: true,
		}, {
			name: "missing server by id",
			node: &corev1.Node{
				Spec: corev1.NodeSpec{ProviderID: "hcloud://2"},
			},
			expected: false,
		}, {
			name: "existing server by name",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foobar",
				},
			},
			expected: true,
		}, {
			name: "missing server by name",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "barfoo",
				},
			},
			expected: false,
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

	instances := newInstances(env.Client, config.AddressFamilyIPv4, 0)

	tests := []struct {
		name     string
		node     *corev1.Node
		expected bool
	}{
		{
			name: "running server",
			node: &corev1.Node{
				Spec: corev1.NodeSpec{ProviderID: "hcloud://1"},
			},
			expected: false,
		}, {
			name: "shutdown server",
			node: &corev1.Node{
				Spec: corev1.NodeSpec{ProviderID: "hcloud://2"},
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

	instances := newInstances(env.Client, config.AddressFamilyIPv4, 0)

	metadata, err := instances.InstanceMetadata(context.TODO(), &corev1.Node{
		Spec: corev1.NodeSpec{ProviderID: "hcloud://1"},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedMetadata := &cloudprovider.InstanceMetadata{
		ProviderID:   "hcloud://1",
		InstanceType: "asdf11",
		NodeAddresses: []corev1.NodeAddress{
			{Type: corev1.NodeHostName, Address: "foobar"},
			{Type: corev1.NodeExternalIP, Address: "203.0.113.7"},
		},
		Zone:   "Test DC",
		Region: "Test Location",
	}

	if !reflect.DeepEqual(metadata, expectedMetadata) {
		t.Fatalf("Expected metadata %+v but got %+v", *expectedMetadata, *metadata)
	}
}

func TestNodeAddresses(t *testing.T) {
	tests := []struct {
		name           string
		addressFamily  config.AddressFamily
		server         *hcloud.Server
		privateNetwork int64
		expected       []corev1.NodeAddress
	}{
		{
			name:          "hostname",
			addressFamily: config.AddressFamilyIPv4,
			server: &hcloud.Server{
				Name: "foobar",
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
			},
		},
		{
			name:          "public ipv4",
			addressFamily: config.AddressFamilyIPv4,
			server: &hcloud.Server{
				Name: "foobar",
				PublicNet: hcloud.ServerPublicNet{
					IPv4: hcloud.ServerPublicNetIPv4{
						IP: net.ParseIP("203.0.113.7"),
					},
					IPv6: hcloud.ServerPublicNetIPv6{
						IP: net.ParseIP("2001:db8:1234::"),
					},
				},
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
				{Type: corev1.NodeExternalIP, Address: "203.0.113.7"},
			},
		},
		{
			name:          "no public ipv4",
			addressFamily: config.AddressFamilyIPv4,
			server: &hcloud.Server{
				Name: "foobar",
				PublicNet: hcloud.ServerPublicNet{
					IPv6: hcloud.ServerPublicNetIPv6{
						IP: net.ParseIP("2001:db8:1234::"),
					},
				},
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
			},
		},
		{
			name:          "public ipv6",
			addressFamily: config.AddressFamilyIPv6,
			server: &hcloud.Server{
				Name: "foobar",
				PublicNet: hcloud.ServerPublicNet{
					IPv4: hcloud.ServerPublicNetIPv4{
						IP: net.ParseIP("203.0.113.7"),
					},
					IPv6: hcloud.ServerPublicNetIPv6{
						IP: net.ParseIP("2001:db8:1234::"),
					},
				},
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
				{Type: corev1.NodeExternalIP, Address: "2001:db8:1234::1"},
			},
		},
		{
			name:          "no public ipv6",
			addressFamily: config.AddressFamilyIPv6,
			server: &hcloud.Server{
				Name: "foobar",
				PublicNet: hcloud.ServerPublicNet{
					IPv4: hcloud.ServerPublicNetIPv4{
						IP: net.ParseIP("203.0.113.7"),
					},
				},
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
			},
		},
		{
			name:          "public dual stack",
			addressFamily: config.AddressFamilyDualStack,
			server: &hcloud.Server{
				Name: "foobar",
				PublicNet: hcloud.ServerPublicNet{
					IPv4: hcloud.ServerPublicNetIPv4{
						IP: net.ParseIP("203.0.113.7"),
					},
					IPv6: hcloud.ServerPublicNetIPv6{
						IP: net.ParseIP("2001:db8:1234::"),
					},
				},
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
				{Type: corev1.NodeExternalIP, Address: "203.0.113.7"},
				{Type: corev1.NodeExternalIP, Address: "2001:db8:1234::1"},
			},
		},

		{
			name:           "unknown private network",
			addressFamily:  config.AddressFamilyIPv4,
			privateNetwork: 1,
			server: &hcloud.Server{
				Name: "foobar",
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
			},
		},
		{
			name:           "server attached to private network",
			addressFamily:  config.AddressFamilyIPv4,
			privateNetwork: 1,
			server: &hcloud.Server{
				Name: "foobar",
				PrivateNet: []hcloud.ServerPrivateNet{
					{
						Network: &hcloud.Network{
							ID:   1,
							Name: "test-existing-nw",
						},
						IP: net.ParseIP("10.0.0.2"),
					},
				},
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
				{Type: corev1.NodeInternalIP, Address: "10.0.0.2"},
			},
		},
		{
			name:           "server not attached to private network",
			addressFamily:  config.AddressFamilyIPv4,
			privateNetwork: 1,
			server: &hcloud.Server{
				Name: "foobar",
				PrivateNet: []hcloud.ServerPrivateNet{
					{
						Network: &hcloud.Network{
							ID:   2,
							Name: "other-nw",
						},
						IP: net.ParseIP("10.0.0.2"),
					},
				},
			},
			expected: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "foobar"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addresses := nodeAddresses(test.addressFamily, test.privateNetwork, test.server)

			if !reflect.DeepEqual(addresses, test.expected) {
				t.Fatalf("Expected addresses %+v but got %+v", test.expected, addresses)
			}
		})
	}
}
