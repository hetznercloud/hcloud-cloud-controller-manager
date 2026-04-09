package servercache

import (
	"context"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var _ ServerCache = (*PerServerCache)(nil)

// PerServerCache caches each Server with a separate expiration time.
// The caches removes expired entries.
type PerServerCache struct {
	subsystem string
	mode      Mode
	ttl       time.Duration

	client *hcloud.Client

	byID   map[int64]*perServerCacheEntry
	byName map[string]*perServerCacheEntry

	mu sync.Mutex
}

type perServerCacheEntry struct {
	server    *hcloud.Server
	expiresAt time.Time
}

func NewPerServerCache(client *hcloud.Client, subsystem string, ttl time.Duration) *PerServerCache {
	return &PerServerCache{
		subsystem: subsystem,
		mode:      ModePerServer,
		ttl:       ttl,
		client:    client,
		byID:      make(map[int64]*perServerCacheEntry),
		byName:    make(map[string]*perServerCacheEntry),
	}
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

func (c *PerServerCache) getOrFetch(
	lookup func() *perServerCacheEntry,
	fetch func() (*hcloud.Server, *hcloud.Response, error),
) (*hcloud.Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry := lookup(); entry != nil && time.Now().Before(entry.expiresAt) {
		metrics.CacheRequests.WithLabelValues(c.subsystem, string(c.mode), "hit").Inc()
		klog.V(4).InfoS("per-server cache hit", "id", entry.server.ID, "name", entry.server.Name)
		return entry.server, nil
	}

	klog.V(4).InfoS("per-server cache miss, fetching from api")
	server, _, err := fetch()
	if err != nil {
		return nil, err
	}
	metrics.CacheRequests.WithLabelValues(c.subsystem, string(c.mode), "miss").Inc()
	if server != nil {
		klog.V(4).InfoS("per-server cache: fetched server from api", "id", server.ID, "name", server.Name)
		c.addToCache(server)
	} else {
		klog.V(4).InfoS("per-server cache: server not found via api")
	}

	return server, nil
}

// addToCache inserts (or refreshes) a server in the cache, evicting the
// expired entries.
// The caller must hold the mutex.
func (c *PerServerCache) addToCache(server *hcloud.Server) {
	if existing, ok := c.byID[server.ID]; ok {
		// Server name changed and needs updating.
		if existing.server.Name != server.Name {
			delete(c.byName, existing.server.Name)
		}
		existing.server = server
		existing.expiresAt = time.Now().Add(c.ttl)
		c.byName[server.Name] = existing
		return
	}

	// Create new server entry.
	entry := &perServerCacheEntry{
		server:    server,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.byID[server.ID] = entry
	c.byName[server.Name] = entry

	// Evict expired entries to avoid deleted Servers being around
	// until hccm restarts.
	for key := range c.byID {
		oldEntry := c.byID[key]
		if time.Now().After(oldEntry.expiresAt) {
			delete(c.byID, key)
			delete(c.byName, oldEntry.server.Name)
			klog.V(4).InfoS("per-server cache: evicted LRU entry", "id", key)
		}
	}
}
