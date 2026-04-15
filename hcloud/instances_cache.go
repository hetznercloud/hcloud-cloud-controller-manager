package hcloud

import (
	"context"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type ServerCache interface {
	ByID(context.Context, int64) (*hcloud.Server, error)
	ByName(context.Context, string) (*hcloud.Server, error)
}

const (
	DefaultServerCacheMaxSize = 64
)

var _ ServerCache = (*PerServerCache)(nil)

type PerServerCache struct {
	ttl     time.Duration
	maxSize int

	client *hcloud.Client

	byID   map[int64]*perServerCacheEntry
	byName map[string]*perServerCacheEntry

	mu sync.Mutex
}

type perServerCacheEntry struct {
	server    *hcloud.Server
	expiredAt time.Time
}

func NewPerServerCache(client *hcloud.Client, ttl time.Duration) *PerServerCache {
	return &PerServerCache{
		ttl:     ttl,
		client:  client,
		maxSize: DefaultServerCacheMaxSize,
		byID:    make(map[int64]*perServerCacheEntry, DefaultServerCacheMaxSize),
		byName:  make(map[string]*perServerCacheEntry, DefaultServerCacheMaxSize),
	}
}

func (c *PerServerCache) getOrFetch(
	lookup func() *perServerCacheEntry,
	fetch func() (*hcloud.Server, *hcloud.Response, error),
) (*hcloud.Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry := lookup(); entry != nil && time.Now().Before(entry.expiredAt) {
		metrics.CacheRequests.WithLabelValues("server", "hit").Inc()
		klog.V(4).InfoS("per-server cache hit", "id", entry.server.ID, "name", entry.server.Name)
		return entry.server, nil
	}

	klog.V(4).InfoS("per-server cache miss, fetching from api")
	server, _, err := fetch()
	metrics.CacheRequests.WithLabelValues("server", "miss").Inc()
	if err != nil {
		return nil, err
	}
	if server != nil {
		klog.V(4).InfoS("per-server cache: fetched server from api", "id", server.ID, "name", server.Name)
		c.addToCache(server)
	} else {
		klog.V(4).InfoS("per-server cache: server not found via api")
	}

	return server, nil
}

func (c *PerServerCache) ByID(ctx context.Context, id int64) (*hcloud.Server, error) {
	return c.getOrFetch(
		func() *perServerCacheEntry { return c.byID[id] },
		func() (*hcloud.Server, *hcloud.Response, error) { return c.client.Server.GetByID(ctx, id) },
	)
}

func (c *PerServerCache) ByName(ctx context.Context, name string) (*hcloud.Server, error) {
	return c.getOrFetch(
		func() *perServerCacheEntry { return c.byName[name] },
		func() (*hcloud.Server, *hcloud.Response, error) { return c.client.Server.GetByName(ctx, name) },
	)
}

// addToCache adds a single Server to the cache.
// The caller must hold the mutex.
func (c *PerServerCache) addToCache(server *hcloud.Server) {
	if len(c.byID) >= c.maxSize {
		klog.V(4).InfoS("per-server cache full: evicting expired entries", "size", len(c.byID), "maxSize", c.maxSize)
		c.evictExpired()
		klog.V(4).InfoS("per-server cache: eviction complete", "size", len(c.byID))
	}

	if len(c.byID) >= c.maxSize {
		klog.V(4).InfoS("per-server cache still full after eviction: clearing cache", "size", len(c.byID), "maxSize", c.maxSize)
		c.byID = make(map[int64]*perServerCacheEntry, c.maxSize)
		c.byName = make(map[string]*perServerCacheEntry, c.maxSize)
	}

	entry := &perServerCacheEntry{
		server:    server,
		expiredAt: time.Now().Add(c.ttl),
	}
	c.byID[server.ID] = entry
	c.byName[server.Name] = entry
}

// evictExpired deletes all expired Servers from the cache.
// The caller must hold the mutex.
func (c *PerServerCache) evictExpired() {
	now := time.Now()
	for id, entry := range c.byID {
		if now.After(entry.expiredAt) {
			delete(c.byName, entry.server.Name)
			delete(c.byID, id)
		}
	}
}

var _ ServerCache = (*AllServerCache)(nil)

type AllServerCache struct {
	ttl       time.Duration
	expiredAt time.Time

	client *hcloud.Client

	byID   map[int64]*hcloud.Server
	byName map[string]*hcloud.Server

	mu sync.Mutex
}

func NewAllServerCache(client *hcloud.Client, ttl time.Duration) *AllServerCache {
	return &AllServerCache{
		ttl:       ttl,
		client:    client,
		expiredAt: time.Now(),
	}
}

func (c *AllServerCache) refresh(ctx context.Context) error {
	klog.V(4).InfoS("all-server cache: refreshing from api")
	servers, err := c.client.Server.All(ctx)
	if err != nil {
		return err
	}

	c.byID = make(map[int64]*hcloud.Server, len(servers))
	c.byName = make(map[string]*hcloud.Server, len(servers))

	for _, server := range servers {
		c.byID[server.ID] = server
		c.byName[server.Name] = server
	}

	c.expiredAt = time.Now().Add(c.ttl)
	klog.V(4).InfoS("all-server cache: refresh complete", "count", len(servers), "expiredAt", c.expiredAt)

	return nil
}

func (c *AllServerCache) getFromCache(ctx context.Context, lookup func() *hcloud.Server) (*hcloud.Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Now().Before(c.expiredAt) {
		if server := lookup(); server != nil {
			metrics.CacheRequests.WithLabelValues("server", "hit").Inc()
			klog.V(4).InfoS("all-server cache hit", "id", server.ID, "name", server.Name)
			return server, nil
		}
		klog.V(4).InfoS("all-server cache miss: server not found in fresh cache, refreshing")
	} else {
		klog.V(4).InfoS("all-server cache miss: cache expired, refreshing", "expiredAt", c.expiredAt)
	}

	// The cache is expired or server was not found. Refreshing
	// cache to catch a server which was just created.
	if err := c.refresh(ctx); err != nil {
		return nil, err
	}

	metrics.CacheRequests.WithLabelValues("server", "miss").Inc()

	// Return server or nil, if the server does still not exist.
	server := lookup()
	if server == nil {
		klog.V(4).InfoS("all-server cache: server not found after refresh")
	} else {
		klog.V(4).InfoS("all-server cache: server found after refresh", "id", server.ID, "name", server.Name)
	}
	return server, nil
}

// ByID implements [ServerCache].
func (c *AllServerCache) ByID(ctx context.Context, id int64) (*hcloud.Server, error) {
	return c.getFromCache(ctx, func() *hcloud.Server { return c.byID[id] })
}

// ByName implements [ServerCache].
func (c *AllServerCache) ByName(ctx context.Context, name string) (*hcloud.Server, error) {
	return c.getFromCache(ctx, func() *hcloud.Server { return c.byName[name] })
}
