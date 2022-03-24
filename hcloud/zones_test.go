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

	"github.com/hetznercloud/hcloud-go/hcloud/schema"
	"github.com/syself/hrobot-go/models"
)

func TestGetZone(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=hcloud%2F%2Fnode6" {
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

	env.Mux.HandleFunc("/robot/server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ServerResponse{
			{
				Server: models.Server{
					ServerNumber: 1,
					Name:         "robot-server1",
					Dc:           "FSN1-DC1",
				},
			},
		})
	})

	zones := newZones(env.Client, env.RobotClient, "hcloud-node6")
	zone, err := zones.GetZone(context.TODO())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "eu-central" {
		t.Errorf("Unexpected zone.FailureDomain: %s", zone.FailureDomain)
	}

	zones = newZones(env.Client, env.RobotClient, "robot-server1")
	zone, err = zones.GetZone(context.TODO())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "eu-central" {
		t.Errorf("Unexpected zone.FailureDomain: %s", zone.FailureDomain)
	}
}

func TestGetZoneForServer(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "name=hcloud%2F%2Fnode15" || strings.HasPrefix(r.URL.Path, "/robot") {
			t.Log("urlPath", r.URL.Path)
			t.Log("r.URL.RawQuery", r.URL.RawQuery)
			t.Fatal("missing name query")
		}
		json.NewEncoder(w).Encode(schema.ServerListResponse{
			Servers: []schema.Server{
				{
					ID:   1,
					Name: "hcloud-node15",
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

	env.Mux.HandleFunc("/robot/server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ServerResponse{
			{
				Server: models.Server{
					ServerNumber: 1,
					Name:         "robot-server1",
					Dc:           "FSN1-DC1",
				},
			},
		})
	})

	zones := newZones(env.Client, env.RobotClient, "hcloud-node6")
	zone, err := zones.GetZoneByNodeName(context.TODO(), "hcloud-node15")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "eu-central" {
		t.Errorf("Unexpected zone.FailureDomain: %s", zone.FailureDomain)
	}

	zone, err = zones.GetZoneByNodeName(context.TODO(), "robot-server1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "eu-central" {
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

	env.Mux.HandleFunc("/robot/server/1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(models.ServerResponse{
			Server: models.Server{
				ServerNumber: 1,
				Name:         "robot-server1",
				Dc:           "FSN1-DC1",
			},
		})
	})

	zones := newZones(env.Client, env.RobotClient, "hcloud-node6")
	zone, err := zones.GetZoneByProviderID(context.TODO(), "hcloud://1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "eu-central" {
		t.Errorf("Unexpected zone.FailureDomain: %s", zone.FailureDomain)
	}

	zone, err = zones.GetZoneByProviderID(context.TODO(), "hetzner://1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if zone.Region != "fsn1" {
		t.Errorf("Unexpected zone.Region: %s", zone.Region)
	}
	if zone.FailureDomain != "eu-central" {
		t.Errorf("Unexpected zone.FailureDomain: %s", zone.FailureDomain)
	}
}
