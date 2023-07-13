package hcops

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// AllServersCache caches the result of the LoadFunc and provides random access
// to servers using select hcloud.Server attributes.
//
// To simplify things the allServersCache reloads all servers on every cache
// miss, or whenever a timeout expired.
type AllServersCache struct {
	LoadFunc    func(context.Context) ([]*hcloud.Server, error)
	LoadTimeout time.Duration
	MaxAge      time.Duration

	lastRefresh time.Time
	byPrivIP    map[string]*hcloud.Server
	byName      map[string]*hcloud.Server

	mu sync.Mutex // protects by* maps
}

// ByPrivateIP obtains a server from the cache using the IP of one of its
// private networks.
//
// Note that a pointer to the object stored in the cache is returned. Modifying
// this object affects the cache and all other code parts holding a reference.
// Furthermore modifying the returned server is not concurrency safe.
func (c *AllServersCache) ByPrivateIP(ip net.IP) (*hcloud.Server, error) {
	const op = "hcops/AllServersCache.ByPrivateIP"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	srv, err := c.getCache(func() (*hcloud.Server, bool) {
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
// Furthermore modifying the returned server is not concurrency safe.
func (c *AllServersCache) ByName(name string) (*hcloud.Server, error) {
	const op = "hcops/AllServersCache.ByName"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	srv, err := c.getCache(func() (*hcloud.Server, bool) {
		srv, ok := c.byName[name]
		return srv, ok
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %s %w", op, name, err)
	}

	return srv, nil
}

func (c *AllServersCache) getCache(getSrv func() (*hcloud.Server, bool)) (*hcloud.Server, error) {
	const op = "hcops/AllServersCache.getCache"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	c.mu.Lock()
	defer c.mu.Unlock()

	// First try to get the value from the cache if the cache is not yet
	// expired.
	if srv, ok := getSrv(); ok && !c.isExpired() {
		return srv, nil
	}

	// Reload from the backend API if we didn't find srv.
	to := c.LoadTimeout
	if to == 0 {
		to = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()

	srvs, err := c.LoadFunc(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Re-initialize all maps. This effectively clears the current cache.
	c.byPrivIP = make(map[string]*hcloud.Server)
	c.byName = make(map[string]*hcloud.Server)

	for _, srv := range srvs {
		// Index servers by the IPs of their private networks
		for _, n := range srv.PrivateNet {
			if n.IP == nil {
				continue
			}
			c.byPrivIP[n.IP.String()] = srv
		}

		// Index servers by their names.
		c.byName[srv.Name] = srv
	}

	c.lastRefresh = time.Now()

	// Re-try to find the server after the reload.
	if srv, ok := getSrv(); ok {
		return srv, nil
	}
	return nil, fmt.Errorf("%s: %w", op, ErrNotFound)
}

// InvalidateCache invalidates the cache so that on the next cache call the cache gets refreshed.
func (c *AllServersCache) InvalidateCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastRefresh = time.Time{}
}

func (c *AllServersCache) isExpired() bool {
	maxAge := c.MaxAge
	if maxAge == 0 {
		maxAge = 10 * time.Minute
	}
	return time.Now().After(c.lastRefresh.Add(maxAge))
}
