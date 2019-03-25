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
	"testing"

	"github.com/hetznercloud/hcloud-go/hcloud/schema"
)

func TestGetZone(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=node6" {
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "node15",
					Datacenter: schema.Datacenter{
						Name: "fsn1-dc8",
						Location: schema.Location{
							Name: "fsn1",
						},
					},
				},
			},
		})
	})

	zones := newZones(env.Client, "node6")
	zone, err := zones.GetZone(context.TODO())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "fsn1-dc8" {
		t.Errorf("Unexpected zone.FailureDomain: %s", zone.FailureDomain)
	}
}

func TestGetZoneForServer(t *testing.T) {
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
					Datacenter: schema.Datacenter{
						Name: "fsn1-dc8",
						Location: schema.Location{
							Name: "fsn1",
						},
					},
				},
			},
		})
	})

	zones := newZones(env.Client, "node6")
	zone, err := zones.GetZoneByNodeName(context.TODO(), "node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "fsn1-dc8" {
		t.Errorf("Unexpected zone.FailureDomain: %s", zone.FailureDomain)
	}
}

func TestGetZoneByProviderID(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:   1,
				Name: "node15",
				Datacenter: schema.Datacenter{
					Name: "fsn1-dc8",
					Location: schema.Location{
						Name: "fsn1",
					},
				},
			},
		})
	})

	zones := newZones(env.Client, "node6")
	zone, err := zones.GetZoneByProviderID(context.TODO(), "hcloud://1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "fsn1-dc8" {
		t.Errorf("Unexpected zone.FailureDomain: %s", zone.FailureDomain)
	}
}
