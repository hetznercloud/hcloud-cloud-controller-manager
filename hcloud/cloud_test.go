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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/assert"
)

type testEnv struct {
	Server *httptest.Server
	Mux    *http.ServeMux
	Client *hcloud.Client
}

func (env *testEnv) Teardown() {
	env.Server.Close()
	env.Server = nil
	env.Mux = nil
	env.Client = nil
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
	return testEnv{
		Server: server,
		Mux:    mux,
		Client: client,
	}
}

func TestNewCloud(t *testing.T) {
	SkipEnv(t, "HCLOUD_ENDPOINT", "HCLOUD_TOKEN", "NODE_NAME")

	var config bytes.Buffer
	_, err := newCloud(&config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestNewCloudWrongTokenSize(t *testing.T) {
	resetEnv := Setenv(t, "HCLOUD_TOKEN", "0123456789abcdef")
	defer resetEnv()

	var config bytes.Buffer
	_, err := newCloud(&config)
	if err == nil || err.Error() != "entered token is invalid (must be exactly 64 characters long)" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestNewCloudConnectionNotPossible(t *testing.T) {
	SkipEnv(t, "HCLOUD_TOKEN", "NODE_NAME")
	resetEnv := Setenv(t, "HCLOUD_ENDPOINT", "http://127.0.0.1:4711/v1")
	defer resetEnv()

	_, err := newCloud(&bytes.Buffer{})
	assert.EqualError(t, err,
		`hcloud/newCloud: Get "http://127.0.0.1:4711/v1/servers?": dial tcp 127.0.0.1:4711: connect: connection refused`)
}

func TestNewCloudInvalidToken(t *testing.T) {
	resetEnv := Setenv(t,
		"HCLOUD_ENDPOINT", "https://api.hetzner.cloud/v1",
		"HCLOUD_TOKEN", "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jN_NOT_VALID_dzhepnahq",
		"NODE_NAME", "test",
	)
	defer resetEnv()

	_, err := newCloud(&bytes.Buffer{})
	assert.EqualError(t, err, "hcloud/newCloud: unable to authenticate (unauthorized)")
}

func TestCloud(t *testing.T) {
	SkipEnv(t, "HCLOUD_ENDPOINT", "HCLOUD_TOKEN", "NODE_NAME")

	resetEnv := Setenv(t,
		"HCLOUD_ENDPOINT", "http://127.0.0.1:4000/v1",
		"HCLOUD_TOKEN", "jr5g7ZHpPptyhJzZyHw2Pqu4g9gTqDvEceYpngPf79jNZXCeTYQ4uArypFM3nh75",
		"NODE_NAME", "test",
	)
	defer resetEnv()

	cloud, err := newCloud(&bytes.Buffer{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Run("Instances", func(t *testing.T) {
		_, supported := cloud.Instances()
		if !supported {
			t.Error("Instances interface should be supported")
		}
	})

	t.Run("Zones", func(t *testing.T) {
		_, supported := cloud.Zones()
		if !supported {
			t.Error("Zones interface should be supported")
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
		resetEnv := Setenv(t, "HCLOUD_NETWORK", "1")
		defer resetEnv()

		c, _ := newCloud(&bytes.Buffer{})
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
