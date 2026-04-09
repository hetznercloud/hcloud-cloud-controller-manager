package servercache

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// ErrRateLimited is returned by a [ServerCache] when a lookup would have
// required a refresh but the cache's internal rate limiter denied it.
// Callers must treat this as "result unknown" — not as "server does not exist".
var ErrRateLimited = errors.New("servercache: refresh rate limited")

type ServerCache interface {
	ByID(context.Context, int64) (*hcloud.Server, error)
	ByName(context.Context, string) (*hcloud.Server, error)
}

// ----- PerServerCache -----

var _ ServerCache = (*PerServerCache)(nil)

type PerServerCache struct {
	caller  string
	mode    string
	ttl     time.Duration
	maxSize int

	client *hcloud.Client

	byID    map[int64]*perServerCacheEntry
	byName  map[string]*perServerCacheEntry
	lruList *list.List // front = MRU, back = LRU; element values are int64 server IDs

	mu sync.Mutex
}

type perServerCacheEntry struct {
	server    *hcloud.Server
	expiredAt time.Time
	element   *list.Element
}

const (
	DefaultPerServerCacheMaxSize = 32
)

func NewPerServerCache(caller string, client *hcloud.Client, ttl time.Duration) *PerServerCache {
	return &PerServerCache{
		caller:  caller,
		mode:    "per-server",
		ttl:     ttl,
		client:  client,
		maxSize: DefaultPerServerCacheMaxSize,
		byID:    make(map[int64]*perServerCacheEntry, DefaultPerServerCacheMaxSize),
		byName:  make(map[string]*perServerCacheEntry, DefaultPerServerCacheMaxSize),
		lruList: list.New(),
	}
}

func (c *PerServerCache) getOrFetch(
	lookup func() *perServerCacheEntry,
	fetch func() (*hcloud.Server, *hcloud.Response, error),
) (*hcloud.Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry := lookup(); entry != nil && time.Now().Before(entry.expiredAt) {
		c.lruList.MoveToFront(entry.element)
		metrics.CacheRequests.WithLabelValues(c.caller, c.mode, "hit").Inc()
		klog.V(4).InfoS("per-server cache hit", "id", entry.server.ID, "name", entry.server.Name)
		return entry.server, nil
	}

	klog.V(4).InfoS("per-server cache miss, fetching from api")
	server, _, err := fetch()
	if err != nil {
		return nil, err
	}
	metrics.CacheRequests.WithLabelValues(c.caller, c.mode, "miss").Inc()
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

// addToCache inserts (or refreshes) a server in the cache, evicting the
// least-recently-used entry if the cache is at capacity.
// The caller must hold the mutex.
func (c *PerServerCache) addToCache(server *hcloud.Server) {
	if existing, ok := c.byID[server.ID]; ok {
		if existing.server.Name != server.Name {
			delete(c.byName, existing.server.Name)
		}
		existing.server = server
		existing.expiredAt = time.Now().Add(c.ttl)
		c.byName[server.Name] = existing
		c.lruList.MoveToFront(existing.element)
		return
	}

	entry := &perServerCacheEntry{
		server:    server,
		expiredAt: time.Now().Add(c.ttl),
	}
	entry.element = c.lruList.PushFront(server.ID)
	c.byID[server.ID] = entry
	c.byName[server.Name] = entry

	for c.lruList.Len() > c.maxSize {
		oldest := c.lruList.Back()
		oldID := oldest.Value.(int64)
		oldEntry := c.byID[oldID]
		delete(c.byID, oldID)
		delete(c.byName, oldEntry.server.Name)
		c.lruList.Remove(oldest)
		klog.V(4).InfoS("per-server cache: evicted LRU entry", "id", oldID)
	}
}

// ----- AllServerCache -----

var _ ServerCache = (*AllServerCache)(nil)

type AllServerCache struct {
	caller    string
	mode      string
	ttl       time.Duration
	expiredAt time.Time

	client *hcloud.Client

	byID   map[int64]*hcloud.Server
	byName map[string]*hcloud.Server

	limiter *rate.Limiter
	mu      sync.Mutex
}

func NewAllServerCache(caller string, client *hcloud.Client, ttl time.Duration) *AllServerCache {
	return &AllServerCache{
		caller:    caller,
		mode:      "all-server",
		ttl:       ttl,
		client:    client,
		expiredAt: time.Now(),
		limiter:   rate.NewLimiter(rate.Every(ttl), 1),
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

	cacheRefreshed := false
	if time.Now().After(c.expiredAt) {
		klog.V(4).InfoS("all-server cache: cache expired, refreshing", "expiredAt", c.expiredAt)
		if err := c.refresh(ctx); err != nil {
			return nil, err
		}
		cacheRefreshed = true
	}

	if server := lookup(); server != nil {
		metrics.CacheRequests.WithLabelValues(c.caller, c.mode, "hit").Inc()
		klog.V(4).InfoS("all-server cache hit", "id", server.ID, "name", server.Name)
		return server, nil
	}

	metrics.CacheRequests.WithLabelValues(c.caller, c.mode, "miss").Inc()

	// Server not found on fresh cache so return early.
	if cacheRefreshed {
		klog.V(4).InfoS("all-server cache: server not found in fresh snapshot")
		return nil, nil
	}

	// Cache was not refreshed and rate limiter does not allow refreshing right now.
	if !c.limiter.Allow() {
		klog.V(4).InfoS("all-server cache: miss-driven refresh denied by rate limiter")
		return nil, ErrRateLimited
	}

	klog.V(4).InfoS("all-server cache miss: refreshing to catch newly-created server")
	if err := c.refresh(ctx); err != nil {
		return nil, err
	}

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

// ----- NoCache -----

// NoCache is a pass-through [ServerCache] that always queries the API.

var _ ServerCache = (*NoCache)(nil)

type NoCache struct {
	client *hcloud.Client
}

func NewNoCache(client *hcloud.Client) *NoCache {
	return &NoCache{client: client}
}

func (c *NoCache) ByID(ctx context.Context, id int64) (*hcloud.Server, error) {
	server, _, err := c.client.Server.GetByID(ctx, id)
	return server, err
}

func (c *NoCache) ByName(ctx context.Context, name string) (*hcloud.Server, error) {
	server, _, err := c.client.Server.GetByName(ctx, name)
	return server, err
}

// ----- EvalCache -----

// EvalCache wraps a [PerServerCache], an [AllServerCache], and a [NoCache] and
// runs each cache sequentially for every lookup.

var _ ServerCache = (*EvalCache)(nil)

type EvalCache struct {
	caches []ServerCache
}

func NewEvalCache(caller string, client *hcloud.Client, ttl time.Duration) *EvalCache {
	caches := []ServerCache{
		NewAllServerCache(caller, client, ttl),
		NewPerServerCache(caller, client, ttl),
		NewNoCache(client),
	}

	return &EvalCache{
		caches: caches,
	}
}

func (c *EvalCache) ByID(ctx context.Context, id int64) (*hcloud.Server, error) {
	return c.run(func(s ServerCache) (*hcloud.Server, error) { return s.ByID(ctx, id) })
}

func (c *EvalCache) ByName(ctx context.Context, name string) (*hcloud.Server, error) {
	return c.run(func(s ServerCache) (*hcloud.Server, error) { return s.ByName(ctx, name) })
}

// run invokes every underlying cache so each one records its own metrics for
// the same request, then returns the first cache's result as the authoritative
// answer. Errors from secondary caches are logged but not propagated, so a
// transient failure in a comparison cache cannot affect the controller.
func (c *EvalCache) run(lookup func(ServerCache) (*hcloud.Server, error)) (*hcloud.Server, error) {
	var (
		firstServer *hcloud.Server
		firstErr    error
	)

	for i, cache := range c.caches {
		server, err := lookup(cache)
		if i == 0 {
			firstServer, firstErr = server, err
			continue
		}
		if err != nil {
			klog.V(4).InfoS("eval cache: secondary cache returned error", "index", i, "err", err)
		}
	}

	return firstServer, firstErr
}
