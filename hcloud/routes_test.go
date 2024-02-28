package hcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	cloudprovider "k8s.io/cloud-provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

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
	env.Mux.HandleFunc("/actions/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.NetworkActionAddRouteResponse{
			Action: schema.Action{
				ID:       1,
				Status:   string(hcloud.ActionStatusSuccess),
				Progress: 100,
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
	routes, err := newRoutes(env.Client, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = routes.CreateRoute(context.TODO(), "my-cluster", "route", &cloudprovider.Route{
		Name:            "route",
		TargetNode:      "node15",
		DestinationCIDR: "10.5.0.0/24",
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
	routes, err := newRoutes(env.Client, 1)
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
	env.Mux.HandleFunc("/actions/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(schema.NetworkActionAddRouteResponse{
			Action: schema.Action{
				ID:       1,
				Status:   string(hcloud.ActionStatusSuccess),
				Progress: 100,
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
	routes, err := newRoutes(env.Client, 1)
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
