package cache

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func assertServer1(t *testing.T, server *hcloud.Server) {
	t.Helper()
	require.NotNil(t, server)
	assert.Equal(t, int64(1), server.ID)
	assert.Equal(t, "test1", server.Name)
}

func assertServer2(t *testing.T, server *hcloud.Server) {
	t.Helper()
	require.NotNil(t, server)
	assert.Equal(t, int64(2), server.ID)
	assert.Equal(t, "test2", server.Name)
}

func newTestCache(mode Mode) *Cache[hcloud.Server] {
	return newCache(
		nil,
		nil,
		nil,
		func(value *hcloud.Server) int64 { return value.ID },
		func(value *hcloud.Server) string { return value.Name },
		mode,
		10*time.Second,
	)
}

func assertCacheHasFreshServer(t *testing.T, cache *Cache[hcloud.Server], value *hcloud.Server) {
	t.Helper()
	assert.Equal(t, time.Now().Add(cache.defaultTTL), cache.byID[value.ID].expiresAt)
	assert.Equal(t, value, cache.byID[value.ID].value)
	assert.Equal(t, value, cache.byName[value.Name].value)
}

func assertCacheLen(t *testing.T, cache *Cache[hcloud.Server], length int) {
	t.Helper()
	assert.Len(t, cache.byID, length)
	assert.Len(t, cache.byName, length)
}

type testClient struct {
	t         *testing.T
	callCount int
}

func newTestClient(t *testing.T) *testClient {
	return &testClient{t: t, callCount: 0}
}

func (c *testClient) CallCount() int {
	return c.callCount
}

func (c *testClient) RequireCallCount(expected int) {
	c.t.Helper()
	require.Equal(c.t, expected, c.callCount, "expected call count to be %d, got %d", expected, c.callCount)

	// /!\ Reset call count until next assertion
	c.callCount = 0
}

func (c *testClient) FetchAllFunc(servers []*hcloud.Server, err error) func(context.Context) ([]*hcloud.Server, error) {
	return func(context.Context) ([]*hcloud.Server, error) {
		c.t.Helper()
		c.callCount++

		for i := range servers {
			servers[i].Status = hcloud.ServerStatus(fmt.Sprintf("call=%d", c.callCount))
		}

		return servers, err
	}
}

func (c *testClient) FetchOneByIDFunc(server *hcloud.Server, err error) func(context.Context, int64) (*hcloud.Server, error) {
	return func(_ context.Context, id int64) (*hcloud.Server, error) {
		c.t.Helper()
		c.callCount++

		if server != nil {
			require.Equal(c.t, server.ID, id, "fetch one by id expected id %d, got %d", server.ID, id)
			server.Status = hcloud.ServerStatus(fmt.Sprintf("call=%d", c.callCount))
		}

		return server, err
	}
}

func (c *testClient) FetchOneByNameFunc(server *hcloud.Server, err error) func(context.Context, string) (*hcloud.Server, error) {
	return func(_ context.Context, name string) (*hcloud.Server, error) {
		c.t.Helper()
		c.callCount++

		if server != nil {
			require.Equal(c.t, server.Name, name, "fetch one by name expected name %s, got %s", server.Name, name)
			server.Status = hcloud.ServerStatus(fmt.Sprintf("call=%d", c.callCount))
		}

		return server, err
	}
}

func TestCache_ModeAll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sc := newTestCache(ModeAll)
		ctx := t.Context()
		client := newTestClient(t)

		{
			// cache miss (by id), fetch all
			sc.fetchAll = client.FetchAllFunc([]*hcloud.Server{{ID: 1, Name: "test1"}, {ID: 2, Name: "test2"}}, nil)

			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 1, client.CallCount())

			assertCacheLen(t, sc, 2)
			assertCacheHasFreshServer(t, sc, srv)
		}
		{
			// cache hit (by id)
			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 1, client.CallCount())
		}
		{
			// cache hit (by name)
			srv, err := sc.ByName(ctx, "test1")
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 1, client.CallCount())
		}
		time.Sleep(sc.defaultTTL - time.Second)
		{
			// cache hit (by id)
			srv, err := sc.ByID(ctx, 2)
			require.NoError(t, err)
			assertServer2(t, srv)
			assert.Equal(t, 1, client.CallCount())
		}
		{
			// cache hit (by name)
			srv, err := sc.ByName(ctx, "test2")
			require.NoError(t, err)
			assertServer2(t, srv)
			assert.Equal(t, 1, client.CallCount())
		}
		time.Sleep(time.Second)
		{
			// cache expired (by id), fetch all
			sc.fetchAll = client.FetchAllFunc([]*hcloud.Server{{ID: 1, Name: "test1"}}, nil)

			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 2, client.CallCount())

			assertCacheLen(t, sc, 1)
			assertCacheHasFreshServer(t, sc, srv)
		}
		{
			// cache miss (by id), fetch all, not found
			srv, err := sc.ByID(ctx, 2)
			require.NoError(t, err)
			assert.Nil(t, srv)
			assert.Equal(t, 3, client.CallCount())

			assertCacheLen(t, sc, 1)
		}
		{
			// cache hit (by id)
			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 3, client.CallCount())
		}
		{
			// cache miss (by id), fetch all, not found
			srv, err := sc.ByID(ctx, 2)
			require.NoError(t, err)
			assert.Nil(t, srv)
			assert.Equal(t, 4, client.CallCount())

			assertCacheLen(t, sc, 1)
		}
	})
}

func TestCache_ModeOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sc := newTestCache(ModeOne)
		ctx := t.Context()
		client := newTestClient(t)

		{
			// cache miss (by id), fetch one
			sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 1, Name: "test1"}, nil)

			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 1, client.CallCount())

			// - test1 is fresh
			assertCacheLen(t, sc, 1)
			assertCacheHasFreshServer(t, sc, srv)
		}
		time.Sleep(sc.defaultTTL - time.Second)
		{
			// cache hit (by id)
			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 1, client.CallCount())
		}
		{
			// cache hit (by name)
			srv, err := sc.ByName(ctx, "test1")
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 1, client.CallCount())
		}
		{
			// cache miss (by id), fetch one
			sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 2, Name: "test2"}, nil)

			srv, err := sc.ByID(ctx, 2)
			require.NoError(t, err)
			assertServer2(t, srv)
			assert.Equal(t, 2, client.CallCount())

			// - test1 is soon expired
			// - test2 is fresh
			assertCacheLen(t, sc, 2)
			assertCacheHasFreshServer(t, sc, srv)
		}
		time.Sleep(time.Second)
		{
			// cache hit (by id)
			srv, err := sc.ByID(ctx, 2)
			require.NoError(t, err)
			assertServer2(t, srv)
			assert.Equal(t, 2, client.CallCount())
		}
		{
			// cache hit (by name)
			srv, err := sc.ByName(ctx, "test2")
			require.NoError(t, err)
			assertServer2(t, srv)
			assert.Equal(t, 2, client.CallCount())
		}
		{
			// cache expired (by id), fetch one
			sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 1, Name: "test1"}, nil)

			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 3, client.CallCount())

			// - test1 is fresh
			// - test2 is fresh
			assertCacheLen(t, sc, 2)
			assertCacheHasFreshServer(t, sc, srv)
		}
		time.Sleep(sc.defaultTTL)
		{
			// cache expired (by id), fetch one, not found
			sc.fetchOneByID = client.FetchOneByIDFunc(nil, nil)

			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assert.Nil(t, srv)
			assert.Equal(t, 4, client.CallCount())

			// - test1 is expired
			// - test2 is expired
			assertCacheLen(t, sc, 2)
		}
		time.Sleep(time.Hour + time.Second)
		{
			// cache expired (by id), fetch one
			sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 2, Name: "test2"}, nil)

			srv, err := sc.ByID(ctx, 2)
			require.NoError(t, err)
			assertServer2(t, srv)
			assert.Equal(t, 5, client.CallCount())

			// - test1 was evicted
			// - test2 is fresh
			assertCacheLen(t, sc, 1)
			assertCacheHasFreshServer(t, sc, srv)
		}
		// We evict only after ttl+hour, not after hour.
		time.Sleep(sc.defaultTTL)
		time.Sleep(time.Hour + time.Second)
		{
			// cache miss (by id), fetch one, not found
			sc.fetchOneByID = client.FetchOneByIDFunc(nil, nil)

			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assert.Nil(t, srv)
			assert.Equal(t, 6, client.CallCount())

			// - test1 was not found and the cache untouched
			// - test2 was not evicted because test1 was not found
			assertCacheLen(t, sc, 1)
		}
		{
			// cache miss (by id), fetch one
			sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 1, Name: "test1"}, nil)

			srv, err := sc.ByID(ctx, 1)
			require.NoError(t, err)
			assertServer1(t, srv)
			assert.Equal(t, 7, client.CallCount())

			// - test1 is fresh
			// - test2 was evicted
			assertCacheLen(t, sc, 1)
			assertCacheHasFreshServer(t, sc, srv)
		}
	})
}

func TestCache_ModeOne_WithTTLRefreshOpts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sc := newTestCache(ModeOne)

		ctx := t.Context()
		client := newTestClient(t)

		// Populate cache with default TTL
		sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 1, Name: "test1"}, nil)

		srv, err := sc.ByID(ctx, 1)
		require.NoError(t, err)
		assertServer1(t, srv)
		assert.Equal(t, time.Now(), sc.byID[srv.ID].refreshedAt)

		// Sleep a bit so refreshedAt has to change
		time.Sleep(15 * time.Second)

		// Fetch Server ID 1, use mode off
		srv, err = sc.ByID(ctx, 1, WithMaxAge(15*time.Second))
		require.NoError(t, err)
		assertServer1(t, srv)
		assert.Equal(t, hcloud.ServerStatusRunning, srv.Status)
		assert.Equal(t, time.Now(), sc.byID[srv.ID].refreshedAt.Add(15*time.Second))

		assert.Equal(t, 1, client.CallCount())
	})
}

func TestCache_ModeOne_WithModeRefreshOpts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sc := newTestCache(ModeOne)

		ctx := t.Context()
		client := newTestClient(t)

		// Populate cache with default TTL
		sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 1, Name: "test1"}, nil)

		srv, err := sc.ByID(ctx, 1)
		require.NoError(t, err)
		assertServer1(t, srv)
		assert.Equal(t, time.Now().Add(sc.defaultTTL), sc.byID[srv.ID].expiresAt)

		// Cache miss by ID 2, fetch from API with different TTL
		sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 2, Name: "test2"}, nil)

		// Fetch Server ID 2, use larger TTL
		srv, err = sc.ByID(ctx, 2)
		require.NoError(t, err)
		assertServer2(t, srv)
		assert.Equal(t, "call=2", string(srv.Status))
		assert.Equal(t, time.Now().Add(sc.defaultTTL), sc.byID[srv.ID].expiresAt)

		// Wait for expiration of Server ID 1 and 2
		time.Sleep(sc.defaultMaxAge + 1)

		// Ensure we only call fetchAll
		sc.fetchOneByID = nil
		sc.fetchAll = client.FetchAllFunc([]*hcloud.Server{
			{ID: 1, Name: "test1"},
			{ID: 2, Name: "test2"},
		}, nil)

		srv, err = sc.ByID(ctx, 1, WithMode(ModeAll))
		require.NoError(t, err)
		assertServer1(t, srv)

		// Server ID 2 is still valid and got powered on with the last fetch all
		srv, err = sc.ByID(ctx, 2)
		require.NoError(t, err)
		assertServer2(t, srv)
		assert.Equal(t, "call=3", string(srv.Status))

		// Expect two API calls
		assert.Equal(t, 3, client.CallCount())

		// Server ID 1 is not evicted, because no refresh happened
		assert.Len(t, sc.byID, 2)
		assert.Len(t, sc.byName, 2)
		assertServer1(t, sc.byID[1].value)
		assertServer2(t, sc.byID[2].value)
	})
}

func TestCache_ModeOff(t *testing.T) {
	sc := newTestCache(ModeOff)

	ctx := t.Context()
	client := newTestClient(t)

	// Cache miss by ID 1, fetch from API
	sc.fetchOneByID = client.FetchOneByIDFunc(&hcloud.Server{ID: 1, Name: "test1"}, nil)

	srv, err := sc.ByID(ctx, 1)
	require.NoError(t, err)
	assertServer1(t, srv)

	assert.Equal(t, 1, client.CallCount())
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)

	// Cache miss by ID 1, fetch from API
	srv, err = sc.ByID(ctx, 1)
	require.NoError(t, err)
	assertServer1(t, srv)

	assert.Equal(t, 2, client.CallCount())
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)

	// Cache miss by Name "test1", fetch from API
	sc.fetchOneByID = nil
	sc.fetchOneByName = client.FetchOneByNameFunc(&hcloud.Server{ID: 1, Name: "test1"}, nil)

	srv, err = sc.ByName(ctx, "test1")
	require.NoError(t, err)
	assertServer1(t, srv)

	assert.Equal(t, 3, client.CallCount())
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)

	// Cache miss by Name "test1", fetch from API
	srv, err = sc.ByName(ctx, "test1")
	require.NoError(t, err)
	assertServer1(t, srv)

	assert.Equal(t, 4, client.CallCount())
	assert.Empty(t, sc.byID)
	assert.Empty(t, sc.byName)
}

func TestCache_Error(t *testing.T) {
	testCase := func(t *testing.T, mode Mode) {
		sc := newTestCache(mode)

		ctx := t.Context()
		client := newTestClient(t)

		sc.fetchOneByID = client.FetchOneByIDFunc(nil, fmt.Errorf("test error"))
		sc.fetchOneByName = client.FetchOneByNameFunc(nil, fmt.Errorf("test error"))
		sc.fetchAll = client.FetchAllFunc(nil, fmt.Errorf("test error"))

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
		assert.Equal(t, 2, client.CallCount())

		// Reset for fetch by Name
		client = newTestClient(t)
		sc.fetchOneByID = client.FetchOneByIDFunc(nil, fmt.Errorf("test error"))
		sc.fetchOneByName = client.FetchOneByNameFunc(nil, fmt.Errorf("test error"))
		sc.fetchAll = client.FetchAllFunc(nil, fmt.Errorf("test error"))

		// Cache miss by name "test1", fetch from API
		srv, err = sc.ByName(ctx, "test1")
		require.ErrorContains(t, err, "test error")
		assert.Nil(t, srv)

		// Error - nothing stored in cache
		assert.Empty(t, sc.byID)
		assert.Empty(t, sc.byName)

		// Second time still errors - two API calls
		srv, err = sc.ByName(ctx, "test1")
		require.ErrorContains(t, err, "test error")
		assert.Nil(t, srv)
		assert.Equal(t, 2, client.CallCount())
	}

	for _, mode := range []Mode{ModeAll, ModeOne, ModeOff} {
		t.Run(string(mode), func(t *testing.T) { testCase(t, mode) })
	}
}
