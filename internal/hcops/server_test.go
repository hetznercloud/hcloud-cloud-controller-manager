package hcops_test

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestAllServersCache_CacheMiss(t *testing.T) {
	srv := &hcloud.Server{
		ID:   12345,
		Name: "cache-miss",
		PrivateNet: []hcloud.ServerPrivateNet{
			{
				IP: net.ParseIP("10.0.0.2"),
			},
		},
	}
	cacheOps := newAllServersCacheOps(t, srv)
	tmpl := allServersCacheTestCase{
		SetUp: func(_ *testing.T, tt *allServersCacheTestCase) {
			tt.ServerClient.
				On("All", mock.Anything).
				Return([]*hcloud.Server{srv}, nil)
		},
		Expected: srv,
	}

	runAllServersCacheTests(t, "Cache miss", tmpl, cacheOps)
}

func TestAllServersCache_CacheHit(t *testing.T) {
	srv := &hcloud.Server{
		ID:   54321,
		Name: "cache-hit",
		PrivateNet: []hcloud.ServerPrivateNet{
			{
				IP: net.ParseIP("10.0.0.3"),
			},
		},
	}
	cacheOps := newAllServersCacheOps(t, srv)
	tmpl := allServersCacheTestCase{
		SetUp: func(t *testing.T, tt *allServersCacheTestCase) {
			tt.ServerClient.
				On("All", mock.Anything).
				Return([]*hcloud.Server{srv}, nil).
				Once()

			// Perform any cache op to initialize caches
			if _, err := tt.Cache.ByName(srv.Name); err != nil {
				t.Fatalf("SetUp: %v", err)
			}
		},
		Assert: func(t *testing.T, tt *allServersCacheTestCase) {
			// All must be called only once. This call has happened during the
			// test SetUp method. All additional calls indicate an error.
			tt.ServerClient.AssertNumberOfCalls(t, "All", 1)
		},
		Expected: srv,
	}

	runAllServersCacheTests(t, "Cache hit", tmpl, cacheOps)
}

func TestAllServersCache_InvalidateCache(t *testing.T) {
	srv := &hcloud.Server{
		ID:   54321,
		Name: "cache-hit",
		PrivateNet: []hcloud.ServerPrivateNet{
			{
				IP: net.ParseIP("10.0.0.3"),
			},
		},
	}
	cacheOps := newAllServersCacheOps(t, srv)
	tmpl := allServersCacheTestCase{
		SetUp: func(t *testing.T, tt *allServersCacheTestCase) {
			tt.ServerClient.
				On("All", mock.Anything).
				Return([]*hcloud.Server{srv}, nil).
				Times(2)

			// Perform any cache op to initialize caches
			if _, err := tt.Cache.ByName(srv.Name); err != nil {
				t.Fatalf("SetUp: %v", err)
			}

			// Invalidate Cache
			tt.Cache.InvalidateCache()

			// Perform a second cache lookup
			if _, err := tt.Cache.ByName(srv.Name); err != nil {
				t.Fatalf("SetUp: %v", err)
			}
		},
		Assert: func(t *testing.T, tt *allServersCacheTestCase) {
			// All must be called twice. This call has happened during the
			// test SetUp method.
			tt.ServerClient.AssertNumberOfCalls(t, "All", 2)
		},
		Expected: srv,
	}

	runAllServersCacheTests(t, "Cache hit", tmpl, cacheOps)
}

func TestAllServersCache_CacheRefresh(t *testing.T) {
	srv := &hcloud.Server{
		ID:   56789,
		Name: "cache-refresh",
		PrivateNet: []hcloud.ServerPrivateNet{
			{
				IP: net.ParseIP("10.0.0.9"),
			},
		},
	}
	cacheOps := newAllServersCacheOps(t, srv)
	tmpl := allServersCacheTestCase{
		SetUp: func(t *testing.T, tt *allServersCacheTestCase) {
			tt.ServerClient.
				On("All", mock.Anything).
				Return([]*hcloud.Server{srv}, nil)

			if _, err := tt.Cache.ByName(srv.Name); err != nil {
				t.Fatalf("SetUp: %v", err)
			}
			// Set the maximum cache age to a ridiculously low time to
			// speed the test up.
			tt.Cache.MaxAge = time.Nanosecond
			time.Sleep(2 * tt.Cache.MaxAge)
		},
		Assert: func(t *testing.T, tt *allServersCacheTestCase) {
			// All must be called only twice. Once during set-up and once
			// during the refresh because of the cache age expired.
			tt.ServerClient.AssertNumberOfCalls(t, "All", 2)
		},
		Expected: srv,
	}

	runAllServersCacheTests(t, "Cache refresh", tmpl, cacheOps)
}

func TestAllServersCache_NotFound(t *testing.T) {
	srv := &hcloud.Server{
		ID:   101010,
		Name: "not-found",
		PrivateNet: []hcloud.ServerPrivateNet{
			{
				IP: net.ParseIP("10.0.0.4"),
			},
		},
	}
	cacheOps := newAllServersCacheOps(t, srv)
	tmpl := allServersCacheTestCase{
		SetUp: func(_ *testing.T, tt *allServersCacheTestCase) {
			tt.ServerClient.
				On("All", mock.Anything).
				Return(nil, nil)
		},
		ExpectedErr: hcops.ErrNotFound,
	}

	runAllServersCacheTests(t, "Not found", tmpl, cacheOps)
}

func TestAllServersCache_ClientError(t *testing.T) {
	err := errors.New("client-error")
	srv := &hcloud.Server{
		ID:   202020,
		Name: "client-error",
		PrivateNet: []hcloud.ServerPrivateNet{
			{
				IP: net.ParseIP("10.0.0.5"),
			},
		},
	}
	cacheOps := newAllServersCacheOps(t, srv)
	tmpl := allServersCacheTestCase{
		SetUp: func(_ *testing.T, tt *allServersCacheTestCase) {
			tt.ServerClient.
				On("All", mock.Anything).
				Return(nil, err)
		},
		ExpectedErr: err,
	}

	runAllServersCacheTests(t, "Not found", tmpl, cacheOps)
}

func TestAllServersCache_DuplicatePrivateIP(t *testing.T) {
	// Regression test for https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/470

	network := &hcloud.Network{
		ID:   12345,
		Name: "cluster-network",
	}
	srv := &hcloud.Server{
		ID:   101010,
		Name: "cluster-node",
		PrivateNet: []hcloud.ServerPrivateNet{
			{
				IP:      net.ParseIP("10.0.0.4"),
				Network: network,
			},
		},
	}
	srvInvalid := &hcloud.Server{
		ID:   101012,
		Name: "invalid-node",
		PrivateNet: []hcloud.ServerPrivateNet{
			{
				IP: net.ParseIP("10.0.0.4"),
				Network: &hcloud.Network{
					ID:   54321,
					Name: "invalid-network",
				},
			},
		},
	}

	cacheOps := newAllServersCacheOps(t, srv)
	tmpl := allServersCacheTestCase{
		SetUp: func(_ *testing.T, tt *allServersCacheTestCase) {
			tt.Cache.Network = network

			tt.ServerClient.
				On("All", mock.Anything).
				Return([]*hcloud.Server{srv, srvInvalid}, nil)
		},
		Expected: srv,
	}

	runAllServersCacheTests(t, "DuplicatePrivateIP", tmpl, cacheOps)
}

type allServersCacheOp func(c *hcops.AllServersCache) (*hcloud.Server, error)

func newAllServersCacheOps(t *testing.T, srv *hcloud.Server) map[string]allServersCacheOp {
	return map[string]allServersCacheOp{
		"ByPrivateIP": func(c *hcops.AllServersCache) (*hcloud.Server, error) {
			if len(srv.PrivateNet) == 0 {
				t.Fatal("ByPrivateIP: server has no private net")
			}
			if len(srv.PrivateNet) > 1 {
				t.Fatal("ByPrivateIP: server more than one private net")
			}
			ip := srv.PrivateNet[0].IP
			if ip == nil {
				t.Fatal("ByPrivateIP: server has no private ip")
			}
			return c.ByPrivateIP(ip)
		},
		"ByName": func(c *hcops.AllServersCache) (*hcloud.Server, error) {
			if srv.Name == "" {
				t.Fatal("ByName: server has no name")
			}
			return c.ByName(srv.Name)
		},
	}
}

type allServersCacheTestCase struct {
	SetUp       func(t *testing.T, tt *allServersCacheTestCase)
	CacheOp     allServersCacheOp
	Expected    *hcloud.Server
	ExpectedErr error
	Assert      func(t *testing.T, tt *allServersCacheTestCase)

	// set in run method.
	ServerClient *mocks.ServerClient
	Cache        *hcops.AllServersCache
}

func (tt *allServersCacheTestCase) run(t *testing.T) {
	tt.ServerClient = mocks.NewServerClient(t)
	tt.Cache = &hcops.AllServersCache{
		LoadFunc: tt.ServerClient.All,
	}

	if tt.SetUp != nil {
		tt.SetUp(t, tt)
	}
	actual, err := tt.CacheOp(tt.Cache)
	if tt.ExpectedErr == nil && !assert.NoError(t, err) {
		return
	}
	if tt.ExpectedErr != nil {
		assert.Truef(t, errors.Is(err, tt.ExpectedErr), "expected error: %v; got %v", tt.ExpectedErr, err)
		return
	}
	assert.Equal(t, tt.Expected, actual)

	if tt.Assert != nil {
		tt.Assert(t, tt)
	}
	tt.ServerClient.AssertExpectations(t)
}

func runAllServersCacheTests(
	t *testing.T, name string, tmpl allServersCacheTestCase, cacheOps map[string]allServersCacheOp,
) {
	for opName, op := range cacheOps {
		assert.Nil(t, tmpl.CacheOp) // just make sure tmpl was not modified

		// Copy tmpl as we do not want to modify the original
		tt := tmpl
		tt.CacheOp = op
		t.Run(fmt.Sprintf("%s: %s", name, opName), tt.run)
	}
}
