package cache

import (
	"context"
	"maps"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
)

type Mode string

const (
	// ModeAll fetches and caches all entries.
	ModeAll Mode = "all"
	// ModeOne fetches and caches one entry.
	ModeOne Mode = "one"
	// ModeOff disables caching.
	ModeOff Mode = "off"
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

type RefreshOption func(ro *RefreshOpts)

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

	byID   map[int64]*entry[T]
	byName map[string]*entry[T]

	mu sync.Mutex
}

func newCache[T any](
	fetchOneByID func(ctx context.Context, id int64) (*T, error),
	fetchOneByName func(ctx context.Context, name string) (*T, error),
	fetchAll func(ctx context.Context) ([]*T, error),
	getID func(value *T) int64,
	getName func(value *T) string,
	defaultMode Mode,
	defaultTTL time.Duration,
) *Cache[T] {
	return &Cache[T]{
		fetchOneByID:   fetchOneByID,
		fetchOneByName: fetchOneByName,
		fetchAll:       fetchAll,
		getID:          getID,
		getName:        getName,

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
	subsystem := GetSubsystem(ctx)
	refreshOpts := newCacheRefreshOpts(c, opts...)

	if refreshOpts.mode == ModeOff {
		metrics.CacheRequests.WithLabelValues(subsystem, string(refreshOpts.mode), "miss").Inc()
		klog.V(4).InfoS("cache mode is off: fetching entry from api", "subsystem", subsystem)
		return fetch()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if e := lookup(); e != nil && time.Now().Before(e.expiresAt) {
		metrics.CacheRequests.WithLabelValues(subsystem, string(refreshOpts.mode), "hit").Inc()
		klog.V(4).InfoS(
			"cache hit",
			"subsystem", subsystem,
			"id", c.getID(e.value),
			"name", c.getName(e.value),
			"expiresAt", e.expiresAt.Format(time.RFC3339),
		)
		return e.value, nil
	}

	switch refreshOpts.mode {
	case ModeOne:
		if err := c.refreshOne(ctx, fetch, refreshOpts.ttl); err != nil {
			return nil, err
		}
	case ModeAll:
		if err := c.refreshAll(ctx, refreshOpts.ttl); err != nil {
			return nil, err
		}
	case ModeOff:
		// Handled above through early return
	}

	metrics.CacheRequests.WithLabelValues(subsystem, string(refreshOpts.mode), "miss").Inc()

	// When the value is not found in the API, the entry is not removed from the cache.
	// Expired entries are only evicted after an hour and when a value is found.
	// Make sure not to return an expired entry.
	if e := lookup(); e != nil && time.Now().Before(e.expiresAt) {
		klog.V(4).InfoS(
			"entry found after refresh",
			"subsystem", subsystem,
			"id", c.getID(e.value), "name", c.getName(e.value),
		)
		return e.value, nil
	}

	klog.V(4).InfoS("entry not found after refresh", "subsystem", subsystem)
	return nil, nil
}

func (c *Cache[T]) refreshOne(
	ctx context.Context,
	fetch func() (*T, error),
	ttl time.Duration,
) error {
	subsystem := GetSubsystem(ctx)
	klog.V(4).InfoS("refreshing entry from api", "subsystem", subsystem)
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
	klog.V(4).InfoS(
		"refreshed entry from api",
		"subsystem", subsystem,
		"id", c.getID(e.value),
		"name", c.getName(e.value),
		"expiresAt", e.expiresAt.Format(time.RFC3339),
	)

	c.byID[c.getID(value)] = e
	c.byName[c.getName(value)] = e

	// Evict expired entries so the cache does not grow indefinitely. This ensures deleted
	// or updated entries are cleaned from the cache. It only acts on entries that
	// expired some time ago, which keeps log output from being spammed.
	evictFunc := func(ev *entry[T]) bool {
		if time.Now().After(ev.expiresAt.Add(time.Hour)) {
			klog.V(4).InfoS(
				"evicting entry from cache",
				"subsystem", subsystem,
				"id", c.getID(ev.value),
				"name", c.getName(ev.value),
				"expiresAt", ev.expiresAt.Format(time.RFC3339),
			)
			return true
		}
		return false
	}

	maps.DeleteFunc(c.byID, func(_ int64, ev *entry[T]) bool {
		return evictFunc(ev)
	})
	maps.DeleteFunc(c.byName, func(_ string, ev *entry[T]) bool {
		return evictFunc(ev)
	})

	return nil
}

func (c *Cache[T]) refreshAll(ctx context.Context, ttl time.Duration) error {
	subsystem := GetSubsystem(ctx)
	klog.V(4).InfoS("refreshing all entries from api", "subsystem", subsystem)

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

		klog.V(4).InfoS(
			"refreshed entry from api",
			"subsystem", subsystem,
			"id", c.getID(e.value),
			"name", c.getName(e.value),
			"expiresAt", expiresAt.Format(time.RFC3339),
		)
	}

	return nil
}
