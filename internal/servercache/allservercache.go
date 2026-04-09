package servercache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var _ ServerCache = (*AllServerCache)(nil)

type AllServerCache struct {
	subsystem string
	mode      Mode
	ttl       time.Duration
	expiresAt time.Time

	client *hcloud.Client

	byID   map[int64]*hcloud.Server
	byName map[string]*hcloud.Server

	limiter *rate.Limiter
	mu      sync.Mutex
}

func NewAllServerCache(client *hcloud.Client, subsystem string, ttl time.Duration) *AllServerCache {
	return &AllServerCache{
		subsystem: subsystem,
		mode:      ModeAllServers,
		ttl:       ttl,
		client:    client,
		expiresAt: time.Now(),
		limiter:   rate.NewLimiter(rate.Every(ttl), 1),
	}
}

// ByID implements [ServerCache].
func (c *AllServerCache) ByID(ctx context.Context, id int64) (*hcloud.Server, error) {
	return c.getFromCache(ctx, func() *hcloud.Server { return c.byID[id] })
}

// ByName implements [ServerCache].
func (c *AllServerCache) ByName(ctx context.Context, name string) (*hcloud.Server, error) {
	return c.getFromCache(ctx, func() *hcloud.Server { return c.byName[name] })
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

	c.expiresAt = time.Now().Add(c.ttl)
	klog.V(4).InfoS("all-server cache: refresh complete", "count", len(servers), "expiresAt", c.expiresAt)

	return nil
}

func (c *AllServerCache) getFromCache(ctx context.Context, lookup func() *hcloud.Server) (*hcloud.Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheRefreshed := false
	if time.Now().After(c.expiresAt) {
		klog.V(4).InfoS("all-server cache: cache expired, refreshing", "expiresAt", c.expiresAt)
		if err := c.refresh(ctx); err != nil {
			return nil, err
		}
		cacheRefreshed = true
	}

	if server := lookup(); server != nil {
		metrics.CacheRequests.WithLabelValues(c.subsystem, string(c.mode), "hit").Inc()
		klog.V(4).InfoS("all-server cache hit", "id", server.ID, "name", server.Name)
		return server, nil
	}

	metrics.CacheRequests.WithLabelValues(c.subsystem, string(c.mode), "miss").Inc()

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
