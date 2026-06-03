package servercache

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
)

func assertServer1(t *testing.T, server *hcloud.Server) {
	t.Helper()
	require.NotNil(t, server)
	assert.Equal(t, int64(1), server.ID)
	assert.Equal(t, "test", server.Name)
}

func assertServer2(t *testing.T, server *hcloud.Server) {
	t.Helper()
	require.NotNil(t, server)
	assert.Equal(t, int64(2), server.ID)
	assert.Equal(t, "test2", server.Name)
}

func TestServerCacheModeAllServers(t *testing.T) {
	apiCalls := atomic.Int64{}

	sc := newCache[hcloud.Server](
		nil,
		nil,
		nil,
		func(value *hcloud.Server) int64 { return value.ID },
		func(value *hcloud.Server) string { return value.Name },
		ModeAll,
		10*time.Second,
	)

	ctx := t.Context()

	// Cache miss by ID 1, fetch from API
	sc.fetchAll = func(_ context.Context) ([]*hcloud.Server, error) {
		apiCalls.Add(1)
		return []*hcloud.Server{{ID: 1, Name: "test"}, {ID: 2, Name: "test2"}}, nil
	}

	srv, err := sc.ByID(ctx, 1)
	require.NoError(t, err)
	assertServer1(t, srv)

	// Fetch all returns 2 servers
	assert.Equal(t, int64(1), apiCalls.Load())
	assert.Len(t, sc.byID, 2)
	assert.Len(t, sc.byName, 2)

	assert.True(t, sc.byID[srv.ID].expiresAt.After(time.Now()))
	assert.Equal(t, srv, sc.byID[srv.ID].value)
	assert.Equal(t, srv, sc.byName[srv.Name].value)

	// Cache hit by ID 1
	srv, err = sc.ByID(ctx, 1)
	require.NoError(t, err)
	assertServer1(t, srv)

	// Cache hit by ID 2
	srv, err = sc.ByID(ctx, 2)
	require.NoError(t, err)
	assertServer2(t, srv)

	// Cache hit by Name 1
	srv, err = sc.ByName(ctx, "test")
	require.NoError(t, err)
	assertServer1(t, srv)

	// Cache hit by Name 2
	srv, err = sc.ByName(ctx, "test2")
	require.NoError(t, err)
	assertServer2(t, srv)

	// Fetched two Servers with one API call
	assert.Equal(t, int64(1), apiCalls.Load())
}

func TestServerCacheModeAllServersNotFound(t *testing.T) {
	apiCalls := atomic.Int64{}

	sc := newCache[hcloud.Server](
		nil,
		nil,
		nil,
		func(value *hcloud.Server) int64 { return value.ID },
		func(value *hcloud.Server) string { return value.Name },
		ModeAll,
		10*time.Second,
	)

	ctx := t.Context()

	// Cache miss by ID 1, fetch from API, not found
	sc.fetchAll = func(_ context.Context) ([]*hcloud.Server, error) {
		apiCalls.Add(1)
		return []*hcloud.Server{{ID: 2, Name: "test2"}, {ID: 3, Name: "test3"}}, nil
	}

	srv, err := sc.ByID(ctx, 1)
	require.NoError(t, err)
	assert.Nil(t, srv)

	// Fetch all returns 2 servers
	assert.Equal(t, int64(1), apiCalls.Load())
	assert.Len(t, sc.byID, 2)
	assert.Len(t, sc.byName, 2)

	// Cache hit by ID 2
	srv, err = sc.ByID(ctx, 2)
	require.NoError(t, err)
	assertServer2(t, srv)

	// Cache hit by Name "test2"
	srv, err = sc.ByName(ctx, "test2")
	require.NoError(t, err)
	assertServer2(t, srv)

	// Cache miss by name "test", fetch from API, not found
	srv, err = sc.ByName(ctx, "test")
	require.NoError(t, err)
	assert.Nil(t, srv)

	// Fetched two Servers with one API call
	assert.Equal(t, int64(2), apiCalls.Load())
}

func TestServerCacheModePerServer(t *testing.T) {
	apiCalls := atomic.Int64{}

	sc := newCache[hcloud.Server](
		nil,
		nil,
		nil,
		func(value *hcloud.Server) int64 { return value.ID },
		func(value *hcloud.Server) string { return value.Name },
		ModeOne,
		10*time.Second,
	)

	ctx := t.Context()

	// Cache miss by ID 1, fetch from API
	sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
		assert.Equal(t, int64(1), id)
		apiCalls.Add(1)
		return &hcloud.Server{ID: 1, Name: "test"}, nil
	}

	srv, err := sc.ByID(ctx, 1)
	require.NoError(t, err)
	assertServer1(t, srv)

	// Fetched one server
	assert.Equal(t, int64(1), apiCalls.Load())
	assert.Len(t, sc.byID, 1)
	assert.Len(t, sc.byName, 1)

	assert.True(t, sc.byID[srv.ID].expiresAt.After(time.Now()))
	assert.Equal(t, srv, sc.byID[srv.ID].value)
	assert.Equal(t, srv, sc.byName[srv.Name].value)

	// Cache hit by ID 1
	srv, err = sc.ByID(ctx, 1)
	require.NoError(t, err)
	assertServer1(t, srv)

	// Cache miss by ID 2, fetch from API
	sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
		assert.Equal(t, int64(2), id)
		apiCalls.Add(1)
		return &hcloud.Server{ID: 2, Name: "test2"}, nil
	}

	srv, err = sc.ByID(ctx, 2)
	require.NoError(t, err)
	assertServer2(t, srv)

	// Cache hit by Name 1
	srv, err = sc.ByName(ctx, "test")
	require.NoError(t, err)
	assertServer1(t, srv)

	// Cache hit by Name 2
	srv, err = sc.ByName(ctx, "test2")
	require.NoError(t, err)
	assertServer2(t, srv)

	// Fetched two servers individually
	assert.Equal(t, int64(2), apiCalls.Load())
}

func TestServerCacheModeOneNotFound(t *testing.T) {
	apiCalls := atomic.Int64{}

	sc := newCache[hcloud.Server](
		nil,
		nil,
		nil,
		func(value *hcloud.Server) int64 { return value.ID },
		func(value *hcloud.Server) string { return value.Name },
		ModeOne,
		10*time.Second,
	)

	ctx := t.Context()

	// Cache miss by ID 1, fetch from API, not found
	sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
		assert.Equal(t, int64(1), id)
		apiCalls.Add(1)
		return nil, nil
	}

	srv, err := sc.ByID(ctx, 1)
	require.NoError(t, err)
	assert.Nil(t, srv)

	// Cached zero server
	assert.Equal(t, int64(1), apiCalls.Load())
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)

	// Cache miss by ID 2, fetch from API
	sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
		assert.Equal(t, int64(2), id)
		apiCalls.Add(1)
		return &hcloud.Server{ID: 2, Name: "test2"}, nil
	}

	srv, err = sc.ByID(ctx, 2)
	require.NoError(t, err)
	assertServer2(t, srv)

	// Cached one server
	assert.Equal(t, int64(2), apiCalls.Load())
	assert.Len(t, sc.byID, 1)
	assert.Len(t, sc.byName, 1)

	// Cache miss by ID 1, fetch from API, not found
	sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
		assert.Equal(t, int64(1), id)
		apiCalls.Add(1)
		return nil, nil
	}

	srv, err = sc.ByID(ctx, 1)
	require.NoError(t, err)
	assert.Nil(t, srv)

	// Fetched zero server
	assert.Equal(t, int64(3), apiCalls.Load())
	assert.Len(t, sc.byID, 1)
	assert.Len(t, sc.byName, 1)
}

func TestServerCacheModeOff(t *testing.T) {
	apiCalls := atomic.Int64{}

	sc := newCache[hcloud.Server](
		nil,
		nil,
		nil,
		func(value *hcloud.Server) int64 { return value.ID },
		func(value *hcloud.Server) string { return value.Name },
		ModeOff,
		10*time.Second,
	)

	ctx := t.Context()

	// Cache miss by ID 1, fetch from API
	sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
		assert.Equal(t, int64(1), id)
		apiCalls.Add(1)
		return &hcloud.Server{ID: 1, Name: "test"}, nil
	}

	srv, err := sc.ByID(ctx, 1)
	require.NoError(t, err)
	assertServer1(t, srv)

	// Fetched one server
	assert.Equal(t, int64(1), apiCalls.Load())
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)

	// Cache miss by ID 1, fetch from API
	srv, err = sc.ByID(ctx, 1)
	require.NoError(t, err)
	assertServer1(t, srv)

	assert.Equal(t, int64(2), apiCalls.Load())
	// Entries are not cached
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)

	// Reset
	sc.fetchOneByID = nil
	apiCalls = atomic.Int64{}

	// Cache miss by Name "test", fetch from API
	sc.fetchOneByName = func(_ context.Context, name string) (*hcloud.Server, error) {
		assert.Equal(t, "test", name)
		apiCalls.Add(1)
		return &hcloud.Server{ID: 1, Name: "test"}, nil
	}

	srv, err = sc.ByName(ctx, "test")
	require.NoError(t, err)
	assertServer1(t, srv)

	// Fetched one server
	assert.Equal(t, int64(1), apiCalls.Load())
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)

	// Cache miss by Name "test", fetch from API
	srv, err = sc.ByName(ctx, "test")
	require.NoError(t, err)
	assertServer1(t, srv)

	assert.Equal(t, int64(2), apiCalls.Load())
	// Entries are not cached
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)
}

func TestServerCacheModePerServer_EvictExpiredEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		apiCalls := atomic.Int64{}

		sc := newCache[hcloud.Server](
			nil,
			nil,
			nil,
			func(value *hcloud.Server) int64 { return value.ID },
			func(value *hcloud.Server) string { return value.Name },
			ModeOne,
			10*time.Second,
		)

		ctx := t.Context()

		// Populate cache
		sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
			assert.Equal(t, int64(1), id)
			apiCalls.Add(1)
			return &hcloud.Server{ID: 1, Name: "test"}, nil
		}
		srv, err := sc.ByID(ctx, 1)
		require.NoError(t, err)

		assert.Equal(t, time.Now().Add(sc.defaultTTL), sc.byID[srv.ID].expiresAt)

		// Wait for expiration
		time.Sleep(sc.defaultTTL + 1)

		// Cache miss by ID 2, fetch from API
		sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
			assert.Equal(t, int64(2), id)
			apiCalls.Add(1)
			return &hcloud.Server{ID: 2, Name: "test2"}, nil
		}

		srv, err = sc.ByID(ctx, 2)
		require.NoError(t, err)
		assertServer2(t, srv)

		// Fetched two servers individually
		assert.Equal(t, int64(2), apiCalls.Load())

		// Server ID 1 has been evicted
		assert.Len(t, sc.byID, 1)
		assert.Len(t, sc.byName, 1)
		assert.Nil(t, sc.byID[1])
		assert.Nil(t, sc.byName["test"])
	})
}

func TestServerCacheModePerServer_WithTTLRefreshOpts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		apiCalls := atomic.Int64{}

		sc := newCache[hcloud.Server](
			nil,
			nil,
			nil,
			func(value *hcloud.Server) int64 { return value.ID },
			func(value *hcloud.Server) string { return value.Name },
			ModeOne,
			5*time.Second,
		)

		ctx := t.Context()

		// Populate cache with default TTL
		sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
			assert.Equal(t, int64(1), id)
			apiCalls.Add(1)
			return &hcloud.Server{ID: 1, Name: "test"}, nil
		}
		srv, err := sc.ByID(ctx, 1)
		require.NoError(t, err)
		assertServer1(t, srv)
		assert.Equal(t, time.Now().Add(sc.defaultTTL), sc.byID[srv.ID].expiresAt)

		// Cache miss by ID 2, fetch from API with different TTL
		sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
			assert.Equal(t, int64(2), id)
			apiCalls.Add(1)
			return &hcloud.Server{ID: 2, Name: "test2"}, nil
		}

		// Fetch Server ID 2, use larger TTL
		srv, err = sc.ByID(ctx, 2, WithTTL(2*sc.defaultTTL))
		require.NoError(t, err)
		assertServer2(t, srv)
		// Server ID 2 should have different TTL
		assert.Equal(t, time.Now().Add(2*sc.defaultTTL), sc.byID[srv.ID].expiresAt)

		// Wait for expiration of Server ID 1
		time.Sleep(sc.defaultTTL + 1)

		// Fetch Server ID 2 again, Server ID 1 is not evicted as no refresh happens
		srv, err = sc.ByID(ctx, 2)
		require.NoError(t, err)
		assertServer2(t, srv)

		// Expect two API calls
		assert.Equal(t, int64(2), apiCalls.Load())

		// Server ID 1 is not evicted, because no refresh happened
		assert.Len(t, sc.byID, 2)
		assert.Len(t, sc.byName, 2)
		assertServer1(t, sc.byID[1].value)
		assertServer2(t, sc.byID[2].value)

		// Server ID 1 is expired with default TTL
		assert.False(t, time.Now().Before(sc.byID[1].expiresAt))
		// Server ID 2 is still fresh -> higher TTL with `WithTTL` option
		assert.True(t, time.Now().Before(sc.byID[2].expiresAt))
	})
}

func TestServerCacheModePerServer_WithModeRefreshOpts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		apiCalls := atomic.Int64{}

		sc := newCache[hcloud.Server](
			nil,
			nil,
			nil,
			func(value *hcloud.Server) int64 { return value.ID },
			func(value *hcloud.Server) string { return value.Name },
			ModeOne,
			5*time.Second,
		)

		ctx := t.Context()

		// Populate cache with default TTL
		sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
			assert.Equal(t, int64(1), id)
			apiCalls.Add(1)
			return &hcloud.Server{ID: 1, Name: "test", Status: hcloud.ServerStatusRunning}, nil
		}
		srv, err := sc.ByID(ctx, 1)
		require.NoError(t, err)
		assertServer1(t, srv)
		assert.Equal(t, time.Now().Add(sc.defaultTTL), sc.byID[srv.ID].expiresAt)

		// Cache miss by ID 2, fetch from API with different TTL
		sc.fetchOneByID = func(_ context.Context, id int64) (*hcloud.Server, error) {
			assert.Equal(t, int64(2), id)
			apiCalls.Add(1)
			return &hcloud.Server{ID: 2, Name: "test2", Status: hcloud.ServerStatusOff}, nil
		}
		// Fetch Server ID 2, use larger TTL
		srv, err = sc.ByID(ctx, 2)
		require.NoError(t, err)
		assertServer2(t, srv)
		assert.Equal(t, hcloud.ServerStatusOff, srv.Status)
		assert.Equal(t, time.Now().Add(sc.defaultTTL), sc.byID[srv.ID].expiresAt)

		// Wait for expiration of Server ID 1 and 2
		time.Sleep(sc.defaultTTL + 1)

		// Ensure we only call fetchAll
		sc.fetchOneByID = nil
		sc.fetchAll = func(_ context.Context) ([]*hcloud.Server, error) {
			apiCalls.Add(1)
			return []*hcloud.Server{
				{ID: 1, Name: "test", Status: hcloud.ServerStatusRunning},
				{ID: 2, Name: "test2", Status: hcloud.ServerStatusRunning},
			}, nil
		}
		srv, err = sc.ByID(ctx, 1, WithMode(ModeAll))
		require.NoError(t, err)
		assertServer1(t, srv)

		// Server ID 2 is still valid and got powered on with the last fetch all
		srv, err = sc.ByID(ctx, 2)
		require.NoError(t, err)
		assertServer2(t, srv)
		assert.Equal(t, hcloud.ServerStatusRunning, srv.Status)

		// Expect two API calls
		assert.Equal(t, int64(3), apiCalls.Load())

		// Server ID 1 is not evicted, because no refresh happened
		assert.Len(t, sc.byID, 2)
		assert.Len(t, sc.byName, 2)
		assertServer1(t, sc.byID[1].value)
		assertServer2(t, sc.byID[2].value)

		// Server ID 1 is expired with default TTL
		assert.True(t, time.Now().Before(sc.byID[1].expiresAt))
		// Server ID 2 is still fresh -> higher TTL with `WithTTL` option
		assert.True(t, time.Now().Before(sc.byID[2].expiresAt))
	})
}

func TestServerCacheAllModesError(t *testing.T) {
	testCase := func(t *testing.T, mode Mode) {
		apiCalls := atomic.Int64{}

		sc := newCache[hcloud.Server](
			func(_ context.Context, id int64) (*hcloud.Server, error) {
				assert.Equal(t, int64(1), id)
				apiCalls.Add(1)
				return nil, fmt.Errorf("test error")
			},
			func(_ context.Context, name string) (*hcloud.Server, error) {
				assert.Equal(t, "test", name)
				apiCalls.Add(1)
				return nil, fmt.Errorf("test error")
			},
			func(_ context.Context) ([]*hcloud.Server, error) {
				apiCalls.Add(1)
				return nil, fmt.Errorf("test error")
			},
			func(value *hcloud.Server) int64 { return value.ID },
			func(value *hcloud.Server) string { return value.Name },
			mode,
			10*time.Second,
		)

		ctx := t.Context()

		// Cache miss by ID 1, fetch from API
		srv, err := sc.ByID(ctx, 1)
		require.ErrorContains(t, err, "test error")
		assert.Nil(t, srv)

		// Error - nothing stored in cache
		assert.Empty(t, sc.byID)
		assert.Empty(t, sc.byName)

		// Second time still errors - two API calls
		srv, err = sc.ByID(ctx, 1)
		require.ErrorContains(t, err, "test error")
		assert.Nil(t, srv)
		assert.Equal(t, int64(2), apiCalls.Load())

		// Reset for fetch by Name
		apiCalls = atomic.Int64{}

		// Cache miss by name "test", fetch from API
		srv, err = sc.ByName(ctx, "test")
		require.ErrorContains(t, err, "test error")
		assert.Nil(t, srv)

		// Error - nothing stored in cache
		assert.Empty(t, sc.byID)
		assert.Empty(t, sc.byName)

		// Second time still errors - two API calls
		srv, err = sc.ByName(ctx, "test")
		require.ErrorContains(t, err, "test error")
		assert.Nil(t, srv)
		assert.Equal(t, int64(2), apiCalls.Load())
	}

	for _, mode := range []Mode{ModeAll, ModeOne, ModeOff} {
		t.Run(string(mode), func(t *testing.T) { testCase(t, mode) })
	}
}

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
