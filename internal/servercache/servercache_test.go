package servercache

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

// newCacheTestClient spins up a [mockutil.Server] and returns a client pointed
// at it.
func newCacheTestClient(t *testing.T, requests []mockutil.Request) *hcloud.Client {
	t.Helper()
	server := mockutil.NewServer(t, requests)
	return hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithPollOpts(hcloud.PollOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
	)
}

var notFoundResponse = schema.ErrorResponse{Error: schema.Error{Code: string(hcloud.ErrorCodeNotFound), Message: "not found"}}

func TestNew(t *testing.T) {
	client := hcloud.NewClient()

	tests := []struct {
		name     string
		mode     Mode
		wantType any
		wantErr  bool
	}{
		{name: "all-server", mode: ModeAllServers, wantType: (*AllServerCache)(nil)},
		{name: "per-server", mode: ModePerServer, wantType: (*PerServerCache)(nil)},
		{name: "eval", mode: ModeEval, wantType: (*EvalCache)(nil)},
		{name: "off", mode: ModeOff, wantType: (*Passthrough)(nil)},
		{name: "invalid", mode: Mode("bogus"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := New(client, "instances_v2", tt.mode, time.Minute)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, cache)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cache)
			assert.IsType(t, tt.wantType, cache)
		})
	}
}

func TestPassthrough_ByID(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers/1", Status: http.StatusOK,
			JSON: schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1"}},
		},
		// Passthrough never caches, so a second lookup must hit the API again.
		{
			Method: "GET", Path: "/servers/1", Status: http.StatusOK,
			JSON: schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1"}},
		},
	})
	cache := NewPassthrough(client)

	srv, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, int64(1), srv.ID)

	srv, err = cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)
}

func TestPassthrough_ByID_NotFound(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{Method: "GET", Path: "/servers/42", Status: http.StatusNotFound, JSON: notFoundResponse},
	})
	cache := NewPassthrough(client)

	srv, err := cache.ByID(t.Context(), 42)
	require.NoError(t, err)
	assert.Nil(t, srv)
}

func TestPassthrough_ByID_APIError(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers/1",
			Status: http.StatusInternalServerError,
			JSON:   schema.ErrorResponse{Error: schema.Error{Code: "boom", Message: "upstream exploded"}},
		},
	})
	cache := NewPassthrough(client)

	srv, err := cache.ByID(t.Context(), 1)
	require.Error(t, err)
	assert.Nil(t, srv)
}

func TestPassthrough_ByName(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers?name=server-1", Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
		{
			Method: "GET", Path: "/servers?name=server-1", Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
	})
	cache := NewPassthrough(client)

	srv, err := cache.ByName(t.Context(), "server-1")
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, "server-1", srv.Name)

	srv, err = cache.ByName(t.Context(), "server-1")
	require.NoError(t, err)
	require.NotNil(t, srv)
}

func TestPassthrough_ByName_NotFound(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers?name=missing", Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{}},
		},
	})
	cache := NewPassthrough(client)

	srv, err := cache.ByName(t.Context(), "missing")
	require.NoError(t, err)
	assert.Nil(t, srv)
}

func TestPassthrough_ByName_APIError(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers?name=server-1",
			Status: http.StatusInternalServerError,
			JSON:   schema.ErrorResponse{Error: schema.Error{Code: "boom", Message: "upstream exploded"}},
		},
	})
	cache := NewPassthrough(client)

	srv, err := cache.ByName(t.Context(), "server-1")
	require.Error(t, err)
	assert.Nil(t, srv)
}
