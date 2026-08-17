package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
)

func TestNewLBTypeCache(t *testing.T) {
	testCases := []struct {
		name     string
		mode     Mode
		requests []mockutil.Request
	}{
		{
			mode: ModeAll,
			requests: []mockutil.Request{
				{Method: "GET", Path: "/load_balancer_types?page=1&per_page=50", Status: 200, JSONRaw: `{ "load_balancer_types": [{ "id": 1, "name": "lb11" }]}`},
			},
		},
		{
			mode: ModeOne,
			requests: []mockutil.Request{
				{Method: "GET", Path: "/load_balancer_types/1", Status: 200, JSONRaw: `{ "load_balancer_type": { "id": 1, "name": "lb11" }}`},
			},
		},
		{
			mode: ModeOff,
			requests: []mockutil.Request{
				{Method: "GET", Path: "/load_balancer_types/1", Status: 200, JSONRaw: `{ "load_balancer_type": { "id": 1, "name": "lb11" }}`},
				{Method: "GET", Path: "/load_balancer_types?name=lb11", Status: 200, JSONRaw: `{ "load_balancer_types": [{ "id": 1, "name": "lb11" }]}`},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(string(tt.mode), func(t *testing.T) {
			server := mockutil.NewServer(t, tt.requests)
			client := hcloud.NewClient(hcloud.WithEndpoint(server.Server.URL))

			cache := NewLoadBalancerTypeCache(client, tt.mode, 10*time.Second)
			require.NotNil(t, cache)
			require.NotNil(t, cache.fetchOneByID)
			require.NotNil(t, cache.fetchOneByName)
			require.NotNil(t, cache.fetchAll)
			require.NotNil(t, cache.getID)
			require.NotNil(t, cache.getName)

			ctx := t.Context()

			srv, err := cache.ByID(ctx, int64(1))
			require.NoError(t, err)
			assert.NotNil(t, srv)

			srv, err = cache.ByName(ctx, "lb11")
			require.NoError(t, err)
			assert.NotNil(t, srv)
		})
	}
}
