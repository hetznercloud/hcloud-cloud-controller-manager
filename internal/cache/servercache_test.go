package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
)

func TestNewServerCache(t *testing.T) {
	// Really want to hit 100% coverage :3
	testCases := []struct {
		name     string
		mode     Mode
		requests []mockutil.Request
	}{
		{
			mode: ModeAll,
			requests: []mockutil.Request{
				{Method: "GET", Path: "/servers?page=1&per_page=50", Status: 200, JSONRaw: `{ "servers": [{ "id": 1, "name": "test" }]}`},
			},
		},
		{
			mode: ModeOne,
			requests: []mockutil.Request{
				{Method: "GET", Path: "/servers/1", Status: 200, JSONRaw: `{ "server": { "id": 1, "name": "test" }}`},
			},
		},
		{
			mode: ModeOff,
			requests: []mockutil.Request{
				{Method: "GET", Path: "/servers/1", Status: 200, JSONRaw: `{ "server": { "id": 1, "name": "test" }}`},
				{Method: "GET", Path: "/servers?name=test", Status: 200, JSONRaw: `{ "servers": [{ "id": 1, "name": "test" }]}`},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(string(tt.mode), func(t *testing.T) {
			server := mockutil.NewServer(t, tt.requests)
			client := hcloud.NewClient(hcloud.WithEndpoint(server.Server.URL))

			cache := NewServerCache(client, tt.mode, 10*time.Second)
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

			srv, err = cache.ByName(ctx, "test")
			require.NoError(t, err)
			assert.NotNil(t, srv)
		})
	}
}
