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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	hrobot "github.com/syself/hrobot-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

type testEnv struct {
	Server      *httptest.Server
	Mux         *http.ServeMux
	Client      *hcloud.Client
	RobotClient hrobot.RobotClient
	Recorder    record.EventRecorder
}

func (env *testEnv) Teardown() {
	env.Server.Close()
	env.Server = nil
	env.Mux = nil
	env.Client = nil
	env.RobotClient = nil
	env.Recorder = nil
}

func newTestEnv() testEnv {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	client := hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithToken("jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jNZXCeTYQ4uArypFM3nh75"),
		hcloud.WithBackoffFunc(func(_ int) time.Duration { return 0 }),
		hcloud.WithDebugWriter(os.Stdout),
	)
	robotClient := hrobot.NewBasicAuthClient("", "")
	robotClient.SetBaseURL(server.URL + "/robot")
	recorder := record.NewBroadcaster().NewRecorder(scheme.Scheme, corev1.EventSource{Component: "hcloud-cloud-controller-manager"})
	return testEnv{
		Server:      server,
		Mux:         mux,
		Client:      client,
		RobotClient: robotClient,
		Recorder:    recorder,
	}
}

func TestNewCloud(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()

	resetEnv := testsupport.Setenv(t,
		"HCLOUD_ENDPOINT", env.Server.URL,
		"HCLOUD_TOKEN", "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
		"HCLOUD_METRICS_ENABLED", "false",
	)
	defer resetEnv()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(
			schema.ServerListResponse{
				Servers: []schema.Server{},
			},
		)
	})
	var config bytes.Buffer
	_, err := newCloud(&config)
	assert.NoError(t, err)
}

func TestNewCloudConnectionNotPossible(t *testing.T) {
	resetEnv := testsupport.Setenv(t,
		"HCLOUD_ENDPOINT", "http://127.0.0.1:4711/v1",
		"HCLOUD_TOKEN", "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
		"HCLOUD_METRICS_ENABLED", "false",
	)
	defer resetEnv()

	_, err := newCloud(&bytes.Buffer{})
	assert.EqualError(t, err,
		`hcloud/newCloud: Get "http://127.0.0.1:4711/v1/servers?": dial tcp 127.0.0.1:4711: connect: connection refused`)
}

func TestNewCloudInvalidToken(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()

	resetEnv := testsupport.Setenv(t,
		"HCLOUD_ENDPOINT", env.Server.URL,
		"HCLOUD_TOKEN", "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
		"HCLOUD_METRICS_ENABLED", "false",
	)
	defer resetEnv()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(
			schema.ErrorResponse{
				Error: schema.Error{
					Code:    "unauthorized",
					Message: "unable to authenticate",
				},
			},
		)
	})

	_, err := newCloud(&bytes.Buffer{})
	assert.EqualError(t, err, "hcloud/newCloud: unable to authenticate (unauthorized)")
}

func TestCloud(t *testing.T) {
	env := newTestEnv()
	defer env.Teardown()

	resetEnv := testsupport.Setenv(t,
		"HCLOUD_ENDPOINT", env.Server.URL,
		"HCLOUD_TOKEN", "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
		"HCLOUD_METRICS_ENABLED", "false",
		"ROBOT_USER", "user",
		"ROBOT_PASSWORD", "pass123",
	)
	defer resetEnv()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(
			schema.ServerListResponse{
				Servers: []schema.Server{
					{
						ID:              1,
						Name:            "test",
						Status:          "running",
						Created:         time.Time{},
						PublicNet:       schema.ServerPublicNet{},
						PrivateNet:      nil,
						ServerType:      schema.ServerType{},
						IncludedTraffic: 0,
						OutgoingTraffic: nil,
						IngoingTraffic:  nil,
						BackupWindow:    nil,
						RescueEnabled:   false,
						ISO:             nil,
						Locked:          false,
						Datacenter:      schema.Datacenter{},
						Image:           nil,
						Protection:      schema.ServerProtection{},
						Labels:          nil,
						Volumes:         nil,
					},
				},
			},
		)
	})
	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(
			schema.NetworkGetResponse{
				Network: schema.Network{
					ID:         1,
					Name:       "test",
					Created:    time.Time{},
					IPRange:    "10.0.0.8",
					Subnets:    nil,
					Routes:     nil,
					Servers:    nil,
					Protection: schema.NetworkProtection{},
					Labels:     nil,
				},
			},
		)
	})

	cloud, err := newCloud(&bytes.Buffer{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Run("Instances", func(t *testing.T) {
		_, supported := cloud.Instances()
		if supported {
			t.Error("Instances interface should not be supported")
		}
	})

	t.Run("Zones", func(t *testing.T) {
		_, supported := cloud.Zones()
		if supported {
			t.Error("Zones interface should not be supported")
		}
	})

	t.Run("InstancesV2", func(t *testing.T) {
		_, supported := cloud.InstancesV2()
		if !supported {
			t.Error("InstancesV2 interface should be supported")
		}
	})

	t.Run("LoadBalancer", func(t *testing.T) {
		_, supported := cloud.LoadBalancer()
		if !supported {
			t.Error("LoadBalancer interface should be supported")
		}
	})

	t.Run("Clusters", func(t *testing.T) {
		_, supported := cloud.Clusters()
		if supported {
			t.Error("Clusters interface should not be supported")
		}
	})

	t.Run("Routes", func(t *testing.T) {
		_, supported := cloud.Routes()
		if supported {
			t.Error("Routes interface should not be supported")
		}
	})

	t.Run("RoutesWithNetworks", func(t *testing.T) {
		resetEnv := testsupport.Setenv(t,
			"HCLOUD_NETWORK", "1",
			"HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK", "true",
			"HCLOUD_METRICS_ENABLED", "false",
			"ROBOT_USER", "",
			"ROBOT_PASSWORD", "",
		)
		defer resetEnv()

		c, err := newCloud(&bytes.Buffer{})
		if err != nil {
			t.Errorf("%s", err)
		}
		_, supported := c.Routes()
		if !supported {
			t.Error("Routes interface should be supported")
		}
	})

	t.Run("HasClusterID", func(t *testing.T) {
		if cloud.HasClusterID() {
			t.Error("HasClusterID should be false")
		}
	})

	t.Run("ProviderName", func(t *testing.T) {
		if cloud.ProviderName() != "hcloud" {
			t.Error("ProviderName should be hcloud")
		}
	})
}
