package hcops

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// AllServersCache caches the result of the LoadFunc and provides random access
// to servers using select hcloud.Server attributes.
//
// To simplify things the AllServersCache reloads all servers on every cache
// miss, or whenever the data is older than MaxAge.
type AllServersCache struct {
	LoadFunc    func(context.Context) ([]*hcloud.Server, error)
	LoadTimeout time.Duration
	MaxAge      time.Duration
	// Set to limit the amount of refreshes due to cache misses
	CacheMissRefreshLimiter *rate.Limiter

	// If set, only IPs in this network will be considered for [ByPrivateIP]
	Network *hcloud.Network

	lastRefresh time.Time
	byPrivIP    map[string]*hcloud.Server
	byName      map[string]*hcloud.Server

	mu sync.Mutex
}

// ByPrivateIP obtains a server from the cache using the IP of one of its
// private networks.
//
// Note that a pointer to the object stored in the cache is returned. Modifying
// this object affects the cache and all other code parts holding a reference.
// Furthermore, modifying the returned server is not concurrency safe.
func (c *AllServersCache) ByPrivateIP(ctx context.Context, ip net.IP) (*hcloud.Server, error) {
	const op = "hcops/AllServersCache.ByPrivateIP"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	srv, err := c.getFromCache(ctx, func() (*hcloud.Server, bool) {
		srv, ok := c.byPrivIP[ip.String()]
		return srv, ok
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %v %w", op, ip, err)
	}

	return srv, nil
}

// ByName obtains a server from the cache using the servers name.
//
// Note that a pointer to the object stored in the cache is returned. Modifying
// this object affects the cache and all other code parts holding a reference.
// Furthermore, modifying the returned server is not concurrency safe.
func (c *AllServersCache) ByName(ctx context.Context, name string) (*hcloud.Server, error) {
	const op = "hcops/AllServersCache.ByName"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	srv, err := c.getFromCache(ctx, func() (*hcloud.Server, bool) {
		srv, ok := c.byName[name]
		return srv, ok
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %s %w", op, name, err)
	}

	return srv, nil
}

// getFromCache wraps the cache maps with expiry time and "get-on-unavailable" functionality.
func (c *AllServersCache) getFromCache(ctx context.Context, retrieveFromCacheMaps func() (*hcloud.Server, bool)) (*hcloud.Server, error) {
	const op = "hcops/AllServersCache.getCache"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	c.mu.Lock()
	defer c.mu.Unlock()

	cacheRefreshed := false

	// Refresh the cache if its expired
	if c.isExpired() {
		if err := c.refreshCache(ctx); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		cacheRefreshed = true
	}

	server, ok := retrieveFromCacheMaps()
	if ok {
		return server, nil
	}

	// If the server was not in the cache, we want to refresh if we did not already in this call and if there is available limit.
	if !cacheRefreshed && c.CacheMissRefreshLimiter.Allow() {
		if err := c.refreshCache(ctx); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		// Check once again if the server was found
		server, ok = retrieveFromCacheMaps()
		if ok {
			return server, nil
		}
	}

	return nil, fmt.Errorf("%s: %w", op, ErrNotFound)
}

// Caller must hold the mutex.
func (c *AllServersCache) refreshCache(ctx context.Context) error {
	const op = "hcops/AllServersCache.refreshCache"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	to := c.LoadTimeout
	if to == 0 {
		to = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	servers, err := c.LoadFunc(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Re-initialize all maps. This effectively clears the current cache.
	c.byPrivIP = make(map[string]*hcloud.Server)
	c.byName = make(map[string]*hcloud.Server)

	for _, server := range servers {
		// Index servers by the IPs of their private networks
		for _, n := range server.PrivateNet {
			if c.Network != nil {
				if n.Network.ID != c.Network.ID {
					// Only consider private IPs in the right network
					continue
				}
			}
			if n.IP == nil {
				continue
			}
			if _, ok := c.byPrivIP[n.IP.String()]; ok {
				klog.Warningf("Overriding AllServersCache entry for private ip %s with server %s\n", n.IP.String(), server.Name)
			}
			c.byPrivIP[n.IP.String()] = server
		}

		// Index servers by their names.
		c.byName[server.Name] = server
	}

	c.lastRefresh = time.Now()

	return nil
}

// InvalidateCache invalidates the cache so that on the next cache call the cache gets refreshed.
func (c *AllServersCache) InvalidateCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastRefresh = time.Time{}
}

// Caller must hold the mutex.
func (c *AllServersCache) isExpired() bool {
	maxAge := c.MaxAge
	if maxAge == 0 {
		maxAge = 10 * time.Minute
	}
	return time.Now().After(c.lastRefresh.Add(maxAge))
}
