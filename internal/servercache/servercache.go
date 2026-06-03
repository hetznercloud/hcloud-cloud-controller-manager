package servercache

import (
	"context"
	"maps"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type Mode string

const (
	ModeAllServers Mode = "all-server"
	ModePerServer  Mode = "per-server"
	ModeOff        Mode = "off"
)

type RefreshOpts struct {
	ttl  time.Duration
	mode Mode
}

func newCacheRefreshOpts[T any](cache *Cache[T], opts ...RefreshOption) *RefreshOpts {
	refreshOpts := &RefreshOpts{
		ttl:  cache.defaultTTL,
		mode: cache.defaultMode,
	}
	for _, opt := range opts {
		opt(refreshOpts)
	}
	return refreshOpts
}

type RefreshOption func(cro *RefreshOpts)

func WithTTL(ttl time.Duration) func(*RefreshOpts) {
	return func(ro *RefreshOpts) {
		ro.ttl = ttl
	}
}

func WithMode(mode Mode) func(*RefreshOpts) {
	return func(ro *RefreshOpts) {
		ro.mode = mode
	}
}

type entry[T any] struct {
	expiresAt time.Time
	value     *T
}

type Cache[T any] struct {
	fetchOneByID   func(ctx context.Context, id int64) (*T, error)
	fetchOneByName func(ctx context.Context, name string) (*T, error)
	fetchAll       func(ctx context.Context) ([]*T, error)
	getID          func(value *T) int64
	getName        func(value *T) string

	defaultTTL  time.Duration
	defaultMode Mode

	subsystem string

	byID   map[int64]*entry[T]
	byName map[string]*entry[T]

	mu sync.Mutex
}

func NewServerCache(client *hcloud.Client, subsystem string, defaultMode Mode, defaultTTL time.Duration) *Cache[hcloud.Server] {
	return newCache[hcloud.Server](
		func(ctx context.Context, id int64) (*hcloud.Server, error) {
			value, _, err := client.Server.GetByID(ctx, id)
			return value, err
		},
		func(ctx context.Context, name string) (*hcloud.Server, error) {
			value, _, err := client.Server.GetByName(ctx, name)
			return value, err
		},
		func(ctx context.Context) ([]*hcloud.Server, error) {
			values, err := client.Server.All(ctx)
			return values, err
		},
		func(value *hcloud.Server) int64 { return value.ID },
		func(value *hcloud.Server) string { return value.Name },
		subsystem,
		defaultMode,
		defaultTTL,
	)
}

func newCache[T any](
	fetchOneByID func(ctx context.Context, id int64) (*T, error),
	fetchOneByName func(ctx context.Context, name string) (*T, error),
	fetchAll func(ctx context.Context) ([]*T, error),
	getID func(value *T) int64,
	getName func(value *T) string,
	subsystem string,
	defaultMode Mode,
	defaultTTL time.Duration,
) *Cache[T] {
	return &Cache[T]{
		fetchOneByID:   fetchOneByID,
		fetchOneByName: fetchOneByName,
		fetchAll:       fetchAll,
		getID:          getID,
		getName:        getName,

		subsystem:   subsystem,
		defaultMode: defaultMode,
		defaultTTL:  defaultTTL,

		byID:   make(map[int64]*entry[T]),
		byName: make(map[string]*entry[T]),
	}
}

func (c *Cache[T]) ByID(ctx context.Context, id int64, opts ...RefreshOption) (*T, error) {
	return c.getFromCache(
		ctx,
		func() *entry[T] {
			return c.byID[id]
		},
		func() (*T, error) {
			return c.fetchOneByID(ctx, id)
		},
		opts...,
	)
}

func (c *Cache[T]) ByName(ctx context.Context, name string, opts ...RefreshOption) (*T, error) {
	return c.getFromCache(
		ctx,
		func() *entry[T] {
			return c.byName[name]
		},
		func() (*T, error) {
			return c.fetchOneByName(ctx, name)
		},
		opts...,
	)
}

func (c *Cache[T]) getFromCache(
	ctx context.Context,
	lookup func() *entry[T],
	fetch func() (*T, error),
	opts ...RefreshOption,
) (*T, error) {
	refreshOpts := newCacheRefreshOpts(c, opts...)

	if refreshOpts.mode == ModeOff {
		metrics.CacheRequests.WithLabelValues(c.subsystem, string(refreshOpts.mode), "miss").Inc()
		klog.V(4).InfoS("cache mode is off")
		return fetch()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if e := lookup(); e != nil && time.Now().Before(e.expiresAt) {
		metrics.CacheRequests.WithLabelValues(c.subsystem, string(refreshOpts.mode), "hit").Inc()
		klog.V(4).InfoS("cache hit", "id", c.getID(e.value), "name", c.getName(e.value), "expiresAt", e.expiresAt.Format(time.RFC3339))
		return e.value, nil
	}

	switch refreshOpts.mode {
	case ModePerServer:
		if err := c.refreshPerServer(fetch, refreshOpts.ttl); err != nil {
			return nil, err
		}
	case ModeAllServers:
		if err := c.refreshAllServer(ctx, refreshOpts.ttl); err != nil {
			return nil, err
		}
	case ModeOff:
		// Handled above -> early return
	}

	metrics.CacheRequests.WithLabelValues(c.subsystem, string(refreshOpts.mode), "miss").Inc()

	if e := lookup(); e != nil {
		klog.V(4).InfoS("entry found after refresh", "id", c.getID(e.value), "name", c.getName(e.value))
		return e.value, nil
	}

	klog.V(4).InfoS("entry not found after refresh")
	return nil, nil
}

func (c *Cache[T]) refreshPerServer(
	fetch func() (*T, error),
	ttl time.Duration,
) error {
	klog.V(4).InfoS("refreshing server from api")
	value, err := fetch()
	if err != nil {
		return err
	}

	if value == nil {
		return nil
	}

	e := &entry[T]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	klog.V(4).InfoS("refreshed entry from api", "id", c.getID(e.value), "name", c.getName(e.value), "expiresAt", e.expiresAt.Format(time.RFC3339))

	c.byID[c.getID(value)] = e
	c.byName[c.getName(value)] = e

	// Evict expired entries so the cache does not grow indefinitely. This ensures deleted
	// Nodes or renamed Servers are cleaned from the cache.
	maps.DeleteFunc(c.byID, func(_ int64, ev *entry[T]) bool {
		if time.Now().After(ev.expiresAt) {
			klog.V(4).InfoS("evicting entry from cache by id", "id", c.getID(ev.value), "name", c.getName(ev.value), "expiresAt", ev.expiresAt.Format(time.RFC3339))
			return true
		}
		return false
	})
	maps.DeleteFunc(c.byName, func(_ string, ev *entry[T]) bool {
		if time.Now().After(ev.expiresAt) {
			klog.V(4).InfoS("evicting entry from cache by name", "id", c.getID(ev.value), "name", c.getName(ev.value), "expiresAt", ev.expiresAt.Format(time.RFC3339))
			return true
		}
		return false
	})

	return nil
}

func (c *Cache[T]) refreshAllServer(ctx context.Context, ttl time.Duration) error {
	klog.V(4).InfoS("refreshing all entries from api")

	values, err := c.fetchAll(ctx)
	if err != nil {
		return err
	}

	c.byID = make(map[int64]*entry[T], len(values))
	c.byName = make(map[string]*entry[T], len(values))

	expiresAt := time.Now().Add(ttl)

	for _, value := range values {
		e := &entry[T]{
			value:     value,
			expiresAt: expiresAt,
		}

		c.byID[c.getID(value)] = e
		c.byName[c.getName(value)] = e
	}

	klog.V(4).InfoS("refreshed all entries from api", "count", len(values), "expiresAt", expiresAt.Format(time.RFC3339))
	return nil
}
