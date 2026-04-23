package hcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	cloudprovider "k8s.io/cloud-provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

func nodeLister(t *testing.T, nodes ...*corev1.Node) corelisters.NodeLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, n := range nodes {
		if err := indexer.Add(n); err != nil {
			t.Fatalf("seed node lister: %v", err)
		}
	}
	return corelisters.NewNodeLister(indexer)
}

const DefaultClusterCIDR = "10.244.0.0/16"

func TestRoutes_CreateRoute(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "node15",
					PrivateNet: []schema.ServerPrivateNet{
						{
							Network: 1,
							IP:      "10.0.0.2",
						},
					},
				},
			},
		})
	})
	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.NetworkGetResponse{
			Network: schema.Network{
				ID:      1,
				Name:    "network-1",
				IPRange: "10.0.0.0/8",
			},
		})
	})
	env.Mux.HandleFunc("/actions", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.ActionListResponse{
			Actions: []schema.Action{
				{
					ID:       1,
					Status:   string(hcloud.ActionStatusSuccess),
					Progress: 100,
				},
			},
		})
	})
	env.Mux.HandleFunc("/networks/1/actions/add_route", func(w http.ResponseWriter, r *http.Request) {
		var reqBody schema.NetworkActionAddRouteRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatal(err)
		}
		if reqBody.Destination != "10.5.0.0/24" {
			t.Errorf("unexpected Destination: %v", reqBody.Destination)
		}
		if reqBody.Gateway != "10.0.0.2" {
			t.Errorf("unexpected Gateway: %v", reqBody.Gateway)
		}
		json.NewEncoder(w).Encode(schema.NetworkActionAddRouteResponse{
			Action: schema.Action{
				ID:       1,
				Progress: 0,
				Status:   string(hcloud.ActionStatusRunning),
			},
		})
	})
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node15"},
		Spec:       corev1.NodeSpec{ProviderID: "hcloud://1"},
	}
	routes, err := newRoutes(env.Client, 1, DefaultClusterCIDR, env.Recorder, nodeLister(t, node))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = routes.CreateRoute(context.TODO(), "my-cluster", "route", &cloudprovider.Route{
		Name:            "route",
		TargetNode:      "node15",
		DestinationCIDR: "10.5.0.0/24",
		TargetNodeAddresses: []corev1.NodeAddress{
			{Type: corev1.NodeInternalIP, Address: "10.0.0.2"},
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestRoutes_ListRoutes(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "node15",
					PrivateNet: []schema.ServerPrivateNet{
						{
							Network: 1,
							IP:      "10.0.0.2",
						},
					},
				},
			},
		})
	})
	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.NetworkGetResponse{
			Network: schema.Network{
				ID:      1,
				Name:    "network-1",
				IPRange: "10.0.0.0/8",
				Routes: []schema.NetworkRoute{
					{
						Destination: "10.5.0.0/24",
						Gateway:     "10.0.0.2",
					},
				},
			},
		})
	})
	routes, err := newRoutes(env.Client, 1, DefaultClusterCIDR, env.Recorder, nodeLister(t))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	r, err := routes.ListRoutes(context.TODO(), "my-cluster")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(r) != 1 {
		t.Errorf("Unexpected routes %v", len(r))
	}
	if r[0].DestinationCIDR != "10.5.0.0/24" {
		t.Errorf("Unexpected DestinationCIDR %v", r[0].DestinationCIDR)
	}
	if r[0].TargetNode != "node15" {
		t.Errorf("Unexpected TargetNode %v", r[0].TargetNode)
	}
}

func TestRoutes_DeleteRoute(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.NetworkGetResponse{
			Network: schema.Network{
				ID:      1,
				Name:    "network-1",
				IPRange: "10.0.0.0/8",
				Routes: []schema.NetworkRoute{
					{
						Destination: "10.5.0.0/24",
						Gateway:     "10.0.0.2",
					},
				},
			},
		})
	})
	env.Mux.HandleFunc("/actions", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.ActionListResponse{
			Actions: []schema.Action{
				{
					ID:       1,
					Status:   string(hcloud.ActionStatusSuccess),
					Progress: 100,
				},
			},
		})
	})
	env.Mux.HandleFunc("/networks/1/actions/delete_route", func(w http.ResponseWriter, r *http.Request) {
		var reqBody schema.NetworkActionDeleteRouteRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatal(err)
		}
		if reqBody.Destination != "10.5.0.0/24" {
			t.Errorf("unexpected Destination: %v", reqBody.Destination)
		}
		if reqBody.Gateway != "10.0.0.2" {
			t.Errorf("unexpected Gateway: %v", reqBody.Gateway)
		}
		json.NewEncoder(w).Encode(schema.NetworkActionDeleteRouteResponse{
			Action: schema.Action{
				ID:       1,
				Progress: 0,
				Status:   string(hcloud.ActionStatusRunning),
			},
		})
	})
	routes, err := newRoutes(env.Client, 1, DefaultClusterCIDR, env.Recorder, nodeLister(t))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = routes.DeleteRoute(context.TODO(), "my-cluster", &cloudprovider.Route{
		Name:            "route",
		TargetNode:      "node15",
		DestinationCIDR: "10.5.0.0/24",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestRoutes_CreateRoute_RobotProviderID(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()

	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.NetworkGetResponse{
			Network: schema.Network{ID: 1, Name: "network-1", IPRange: "10.0.0.0/8"},
		})
	})

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "robot-node"},
		Spec:       corev1.NodeSpec{ProviderID: "hrobot://1"},
	}

	routes, err := newRoutes(env.Client, 1, DefaultClusterCIDR, env.Recorder, nodeLister(t, node))
	require.NoError(t, err)

	err = routes.CreateRoute(context.TODO(), "my-cluster", "route", &cloudprovider.Route{
		TargetNode:      "robot-node",
		DestinationCIDR: "10.5.0.0/24",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "not a Cloud server")
}

// TestRoutes_CreateRoute_NodeNameDrift proves the routes controller still works when the
// k8s node name differs from the hcloud server name — the core fix in this PR. The server is
// resolved by its immutable ProviderID rather than by k8s node name.
func TestRoutes_CreateRoute_NodeNameDrift(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()

	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   42,
					Name: "original-hostname", // intentionally differs from the k8s node name
					PrivateNet: []schema.ServerPrivateNet{
						{Network: 1, IP: "10.0.0.2"},
					},
				},
			},
		})
	})
	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.NetworkGetResponse{
			Network: schema.Network{ID: 1, Name: "network-1", IPRange: "10.0.0.0/8"},
		})
	})
	env.Mux.HandleFunc("/actions", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.ActionListResponse{
			Actions: []schema.Action{{ID: 1, Status: string(hcloud.ActionStatusSuccess), Progress: 100}},
		})
	})
	env.Mux.HandleFunc("/networks/1/actions/add_route", func(w http.ResponseWriter, r *http.Request) {
		var reqBody schema.NetworkActionAddRouteRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "10.0.0.2", reqBody.Gateway)
		json.NewEncoder(w).Encode(schema.NetworkActionAddRouteResponse{
			Action: schema.Action{ID: 1, Status: string(hcloud.ActionStatusRunning)},
		})
	})

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "drifted-name"},
		Spec:       corev1.NodeSpec{ProviderID: "hcloud://42"},
	}

	routes, err := newRoutes(env.Client, 1, DefaultClusterCIDR, env.Recorder, nodeLister(t, node))
	require.NoError(t, err)

	err = routes.CreateRoute(context.TODO(), "my-cluster", "route", &cloudprovider.Route{
		TargetNode:      "drifted-name",
		DestinationCIDR: "10.5.0.0/24",
		TargetNodeAddresses: []corev1.NodeAddress{
			{Type: corev1.NodeInternalIP, Address: "10.0.0.2"},
		},
	})
	require.NoError(t, err)
}

// TestRoutes_CreateRoute_ReplaceStaleRoute asserts that a pre-existing route with a wrong
// gateway is deleted and a new one is added in the same call (upsert-in-place).
func TestRoutes_CreateRoute_ReplaceStaleRoute(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()

	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, _ *http.Request) {
		err := json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{ID: 1, Name: "node15", PrivateNet: []schema.ServerPrivateNet{{Network: 1, IP: "10.0.0.2"}}},
			},
		})
		assert.NoError(t, err)
	})
	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, _ *http.Request) {
		err := json.NewEncoder(w).Encode(schema.NetworkGetResponse{
			Network: schema.Network{
				ID: 1, Name: "network-1", IPRange: "10.0.0.0/8",
				Routes: []schema.NetworkRoute{
					{Destination: "10.5.0.0/24", Gateway: "10.99.99.99"},
				},
			},
		})
		assert.NoError(t, err)
	})
	env.Mux.HandleFunc("/actions", func(w http.ResponseWriter, _ *http.Request) {
		err := json.NewEncoder(w).Encode(schema.ActionListResponse{
			Actions: []schema.Action{{ID: 1, Status: string(hcloud.ActionStatusSuccess), Progress: 100}},
		})
		assert.NoError(t, err)
	})
	env.Mux.HandleFunc("/networks/1/actions/delete_route", func(w http.ResponseWriter, r *http.Request) {
		var reqBody schema.NetworkActionDeleteRouteRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "10.99.99.99", reqBody.Gateway)
		err := json.NewEncoder(w).Encode(schema.NetworkActionDeleteRouteResponse{
			Action: schema.Action{ID: 1, Status: string(hcloud.ActionStatusRunning)},
		})
		assert.NoError(t, err)
	})
	env.Mux.HandleFunc("/networks/1/actions/add_route", func(w http.ResponseWriter, r *http.Request) {
		var reqBody schema.NetworkActionAddRouteRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "10.0.0.2", reqBody.Gateway)
		err := json.NewEncoder(w).Encode(schema.NetworkActionAddRouteResponse{
			Action: schema.Action{ID: 1, Status: string(hcloud.ActionStatusRunning)},
		})
		assert.NoError(t, err)
	})

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node15"},
		Spec:       corev1.NodeSpec{ProviderID: "hcloud://1"},
	}

	routes, err := newRoutes(env.Client, 1, DefaultClusterCIDR, env.Recorder, nodeLister(t, node))
	require.NoError(t, err)

	err = routes.CreateRoute(context.TODO(), "my-cluster", "route", &cloudprovider.Route{
		TargetNode:      "node15",
		DestinationCIDR: "10.5.0.0/24",
		TargetNodeAddresses: []corev1.NodeAddress{
			{Type: corev1.NodeInternalIP, Address: "10.0.0.2"},
		},
	})
	require.NoError(t, err)
}

// TestRoutes_CreateRoute_AlreadyExists asserts no API mutations happen when the route already
// matches the desired destination + gateway.
func TestRoutes_CreateRoute_AlreadyExists(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()

	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{ID: 1, Name: "node15", PrivateNet: []schema.ServerPrivateNet{{Network: 1, IP: "10.0.0.2"}}},
			},
		})
	})
	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.NetworkGetResponse{
			Network: schema.Network{
				ID: 1, Name: "network-1", IPRange: "10.0.0.0/8",
				Routes: []schema.NetworkRoute{
					{Destination: "10.5.0.0/24", Gateway: "10.0.0.2"},
				},
			},
		})
	})

	calledAdd, calledDelete := false, false
	env.Mux.HandleFunc("/networks/1/actions/delete_route", func(_ http.ResponseWriter, _ *http.Request) {
		calledDelete = true
	})
	env.Mux.HandleFunc("/networks/1/actions/add_route", func(_ http.ResponseWriter, _ *http.Request) {
		calledAdd = true
	})

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node15"},
		Spec:       corev1.NodeSpec{ProviderID: "hcloud://1"},
	}

	routes, err := newRoutes(env.Client, 1, DefaultClusterCIDR, env.Recorder, nodeLister(t, node))
	require.NoError(t, err)

	err = routes.CreateRoute(context.TODO(), "my-cluster", "route", &cloudprovider.Route{
		TargetNode:      "node15",
		DestinationCIDR: "10.5.0.0/24",
		TargetNodeAddresses: []corev1.NodeAddress{
			{Type: corev1.NodeInternalIP, Address: "10.0.0.2"},
		},
	})
	require.NoError(t, err)
	assert.False(t, calledAdd, "route should already exist: /add_route must not be called")
	assert.False(t, calledDelete, "route should already exist: /delete_route must not be called")
}
