package servercache

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

// newCacheTestClient spins up a [mockutil.Server] and returns a client pointed
// at it. The mock server asserts at test end that all expected requests were
// consumed (serving as an implicit cache hit/miss assertion).
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

// ----- PerServerCache -----

func TestPerServerCache_ByID_HitAfterMiss(t *testing.T) {
	// Exactly one GET /servers/1 is expected; the second ByID must be served from cache.
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: "/servers/1",
			Status: http.StatusOK,
			JSON:   schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1"}},
		},
	})
	cache := NewPerServerCache("instances_v2", client, time.Minute)

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
	cache := NewPerServerCache("instances_v2", client, time.Minute)

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
	cache := NewPerServerCache("instances_v2", client, time.Minute)

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
	cache := NewPerServerCache("instances_v2", client, 0)

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
	cache := NewPerServerCache("instances_v2", client, time.Minute)

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
	cache := NewPerServerCache("instances_v2", client, time.Minute)

	srv, err := cache.ByID(t.Context(), 1)
	require.Error(t, err)
	assert.Nil(t, srv)
}

func TestPerServerCache_LRUEviction(t *testing.T) {
	// Cache size 2; adding a third entry evicts the least-recently-used.
	// Touching server 1 before inserting 3 keeps 1 in cache and evicts 2.
	client := newCacheTestClient(t, []mockutil.Request{
		{Method: "GET", Path: "/servers/1", Status: http.StatusOK, JSON: schema.ServerGetResponse{Server: schema.Server{ID: 1, Name: "server-1"}}},
		{Method: "GET", Path: "/servers/2", Status: http.StatusOK, JSON: schema.ServerGetResponse{Server: schema.Server{ID: 2, Name: "server-2"}}},
		{Method: "GET", Path: "/servers/3", Status: http.StatusOK, JSON: schema.ServerGetResponse{Server: schema.Server{ID: 3, Name: "server-3"}}},
	})
	cache := NewPerServerCache("instances_v2", client, time.Hour)
	cache.maxSize = 2

	ctx := t.Context()
	_, err := cache.ByID(ctx, 1)
	require.NoError(t, err)
	_, err = cache.ByID(ctx, 2)
	require.NoError(t, err)

	// Touch 1 → 1 is MRU, 2 is LRU.
	_, err = cache.ByID(ctx, 1)
	require.NoError(t, err)

	// Adding 3 evicts the LRU (2).
	_, err = cache.ByID(ctx, 3)
	require.NoError(t, err)

	assert.Len(t, cache.byID, 2)
	assert.Contains(t, cache.byID, int64(1))
	assert.Contains(t, cache.byID, int64(3))
	assert.NotContains(t, cache.byID, int64(2))
	assert.Len(t, cache.byName, 2)
	assert.NotContains(t, cache.byName, "server-2")
}

// ----- AllServerCache -----

// allServersListOK is the canonical response to `Server.All()` used by the
// AllServerCache tests. [hcloud.ServerClient.All] paginates with per_page=50.
const allServersPath = "/servers?page=1&per_page=50"

func TestAllServerCache_ByID_HitAfterMiss(t *testing.T) {
	// One refresh is expected; the second lookup within TTL must not trigger another.
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{
				{ID: 1, Name: "server-1"},
				{ID: 2, Name: "server-2"},
			}},
		},
	})
	cache := NewAllServerCache("instances_v2", client, time.Minute)

	srv, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, "server-1", srv.Name)

	srv, err = cache.ByID(t.Context(), 2)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, "server-2", srv.Name)
}

func TestAllServerCache_ByName_HitAfterMiss(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
	})
	cache := NewAllServerCache("instances_v2", client, time.Minute)

	srv, err := cache.ByName(t.Context(), "server-1")
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, int64(1), srv.ID)

	srv, err = cache.ByName(t.Context(), "server-1")
	require.NoError(t, err)
	require.NotNil(t, srv)
}

func TestAllServerCache_TTLExpiry(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
	})
	cache := NewAllServerCache("instances_v2", client, 0)

	_, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	_, err = cache.ByID(t.Context(), 1)
	require.NoError(t, err)
}

func TestAllServerCache_MissTriggersRefresh(t *testing.T) {
	// Initially only server 1 is returned. A lookup for id=2 triggers a refresh
	// that now also returns server 2 (e.g. because it was just created).
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{
				{ID: 1, Name: "server-1"},
				{ID: 2, Name: "server-2"},
			}},
		},
	})
	cache := NewAllServerCache("instances_v2", client, time.Minute)
	cache.limiter = rate.NewLimiter(rate.Inf, 1) // isolate the test from the limiter

	_, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)

	srv, err := cache.ByID(t.Context(), 2)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, "server-2", srv.Name)
}

func TestAllServerCache_ServerNotFoundAfterRefresh(t *testing.T) {
	// A missing server triggers a refresh on every lookup, since we have no
	// way to distinguish "just created" from "does not exist".
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
	})
	cache := NewAllServerCache("instances_v2", client, time.Minute)
	cache.limiter = rate.NewLimiter(rate.Inf, 1) // isolate the test from the limiter

	srv, err := cache.ByID(t.Context(), 999)
	require.NoError(t, err)
	assert.Nil(t, srv)

	srv, err = cache.ByID(t.Context(), 999)
	require.NoError(t, err)
	assert.Nil(t, srv)
}

func TestAllServerCache_RateLimitedRefreshReturnsErr(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
	})
	cache := NewAllServerCache("instances_v2", client, time.Minute)
	cache.limiter = rate.NewLimiter(rate.Every(time.Hour), 1)

	srv, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)

	// Drain the limiter so the next miss-driven refresh is denied.
	require.True(t, cache.limiter.Allow())

	srv, err = cache.ByID(t.Context(), 999)
	require.ErrorIs(t, err, ErrRateLimited)
	assert.Nil(t, srv)
}

func TestAllServerCache_ExpiredRefreshFailureDoesNotConsumeLimiter(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: allServersPath,
			Status: http.StatusInternalServerError,
			JSON:   schema.ErrorResponse{Error: schema.Error{Code: "boom", Message: "upstream exploded"}},
		},
		{
			Method: "GET", Path: allServersPath, Status: http.StatusOK,
			JSON: schema.ServerListResponse{Servers: []schema.Server{{ID: 1, Name: "server-1"}}},
		},
	})
	cache := NewAllServerCache("instances_v2", client, time.Minute)
	cache.limiter = rate.NewLimiter(rate.Every(time.Hour), 1)

	_, err := cache.ByID(t.Context(), 1)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrRateLimited)

	srv, err := cache.ByID(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.Equal(t, "server-1", srv.Name)
}

func TestAllServerCache_APIError(t *testing.T) {
	client := newCacheTestClient(t, []mockutil.Request{
		{
			Method: "GET", Path: allServersPath,
			Status: http.StatusInternalServerError,
			JSON:   schema.ErrorResponse{Error: schema.Error{Code: "boom", Message: "upstream exploded"}},
		},
	})
	cache := NewAllServerCache("instances_v2", client, time.Minute)

	srv, err := cache.ByID(t.Context(), 1)
	require.Error(t, err)
	assert.Nil(t, srv)
}
