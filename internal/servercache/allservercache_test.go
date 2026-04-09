package servercache

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

// allServersPath is the canonical response to `Server.All()` used by the
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
	cache := NewAllServerCache(client, "instances_v2", time.Minute)

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
	cache := NewAllServerCache(client, "instances_v2", time.Minute)

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
	cache := NewAllServerCache(client, "instances_v2", 0)

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
	cache := NewAllServerCache(client, "instances_v2", time.Minute)
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
	cache := NewAllServerCache(client, "instances_v2", time.Minute)
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
	cache := NewAllServerCache(client, "instances_v2", time.Minute)
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
	cache := NewAllServerCache(client, "instances_v2", time.Minute)
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
	cache := NewAllServerCache(client, "instances_v2", time.Minute)

	srv, err := cache.ByID(t.Context(), 1)
	require.Error(t, err)
	assert.Nil(t, srv)
}
