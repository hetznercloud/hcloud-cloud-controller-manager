package servercache

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// EvalCache wraps a [PerServerCache], an [AllServerCache], and a [Passthrough] and
// runs each cache sequentially for every lookup.

var _ ServerCache = (*EvalCache)(nil)

type EvalCache struct {
	caches []ServerCache
}

func NewEvalCache(client *hcloud.Client, subsystem string, ttl time.Duration) *EvalCache {
	caches := []ServerCache{
		NewAllServerCache(client, subsystem, ttl),
		NewPerServerCache(client, subsystem, ttl),
		NewPassthrough(client),
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
