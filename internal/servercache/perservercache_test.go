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

func TestPerServerCache_ByID_HitAfterMiss(t *testing.T) {
	// Exactly one GET /servers/1 is expected; the second ByID must be served from cache.
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers/1",
			Status: http.StatusOK,
			JSON:   schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1"}},
		},
	})
	cache := NewPerServerCache(client, "instances_v2", time.Minute)

	srv, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, int64(1), srv.ID)

	srv, err = cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)
}

func TestPerServerCache_ByName_HitAfterMiss(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers?name=server-1",
			Status: http.StatusOK,
			JSON:   schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
	})
	cache := NewPerServerCache(client, "instances_v2", time.Minute)

	srv, err := cache.ByName(t.Context(), "server-1")
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, "server-1", srv.Name)

	srv, err = cache.ByName(t.Context(), "server-1")
	require.NoError(t, err)
	require.NotNil(t, srv)
}

func TestPerServerCache_ByID_CrossPopulatesByName(t *testing.T) {
	// Only the initial GetByID call is expected; the subsequent ByName must hit
	// the cache populated by ByID.
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers/1",
			Status: http.StatusOK,
			JSON:   schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1"}},
		},
	})
	cache := NewPerServerCache(client, "instances_v2", time.Minute)

	srv, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)

	srv, err = cache.ByName(t.Context(), "server-1")
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, int64(1), srv.ID)
}

func TestPerServerCache_TTLExpiry(t *testing.T) {
	// Zero TTL → every lookup misses and triggers a fresh GET.
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers/1", Status: http.StatusOK,
			JSON: schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1", Status: string(hcloud.ServerStatusStarting)}},
		},
		{
			Method: "GET", Path: "/servers/1", Status: http.StatusOK,
			JSON: schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1", Status: string(hcloud.ServerStatusRunning)}},
		},
	})
	cache := NewPerServerCache(client, "instances_v2", 0)

	srv, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, hcloud.ServerStatusStarting, srv.Status)

	srv, err = cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, hcloud.ServerStatusRunning, srv.Status)
}

func TestPerServerCache_ServerNotFound(t *testing.T) {
	// A nil result is not cached → both lookups must hit the api.
	client := newCacheTestClient(t, []mockutil.Request{
		{Method: "GET", Path: "/servers/42", Status: http.StatusNotFound, JSON: notFoundResponse},
		{Method: "GET", Path: "/servers/42", Status: http.StatusNotFound, JSON: notFoundResponse},
	})
	cache := NewPerServerCache(client, "instances_v2", time.Minute)

	srv, err := cache.ByID(t.Context(), 42)
	require.NoError(t, err)
	assert.Nil(t, srv)

	srv, err = cache.ByID(t.Context(), 42)
	require.NoError(t, err)
	assert.Nil(t, srv)
}

func TestPerServerCache_APIError(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers/1",
			Status: http.StatusInternalServerError,
			JSON:   schema.ErrorResponse{Error: schema.Error{Code: "boom", Message: "upstream exploded"}},
		},
	})
	cache := NewPerServerCache(client, "instances_v2", time.Minute)

	srv, err := cache.ByID(t.Context(), 1)
	require.Error(t, err)
	assert.Nil(t, srv)
}

func TestPerServerCache_ExpiredEviction(t *testing.T) {
	// Cache size 2; adding a third entry evicts the least-recently-used.
	// Touching server 1 before inserting 3 keeps 1 in cache and evicts 2.
	client := newCacheTestClient(t, []mockutil.Request{
		{Method: "GET", Path: "/servers/1", Status: http.StatusOK, JSON: schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1"}}},
		{Method: "GET", Path: "/servers/2", Status: http.StatusOK, JSON: schema.ServerGetResponse{Server: schema.Server{ID: 2, Name: "server-2"}}},
		{Method: "GET", Path: "/servers/3", Status: http.StatusOK, JSON: schema.ServerGetResponse{Server: schema.Server{ID: 3, Name: "server-3"}}},
	})
	cache := NewPerServerCache(client, "instances_v2", time.Hour)

	ctx := t.Context()
	_, err := cache.ByID(ctx, 1)
	require.NoError(t, err)
	_, err = cache.ByID(ctx, 2)
	require.NoError(t, err)

	// Expire 2
	cache.byID[2].expiresAt = time.Unix(0, 0)

	// Adding 3 evicts the expired entry (2).
	_, err = cache.ByID(ctx, 3)
	require.NoError(t, err)

	assert.Len(t, cache.byID, 2)
	assert.Contains(t, cache.byID, int64(1))
	assert.Contains(t, cache.byID, int64(3))
	assert.NotContains(t, cache.byID, int64(2))
	assert.Len(t, cache.byName, 2)
	assert.NotContains(t, cache.byName, "server-2")
}
