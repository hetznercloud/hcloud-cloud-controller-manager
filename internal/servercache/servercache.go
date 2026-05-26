package servercache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// ServerCache defines a caching layer for retrieving Hetzner Cloud servers.
type ServerCache interface {
	// ByID retrieves a server by its unique numeric ID.
	// Returns the server if found, nil and no error if not found,
	// or nil and an error if the lookup fails.
	ByID(context.Context, int64) (*hcloud.Server, error)

	// ByName retrieves a server by its name.
	// Returns the server if found, nil and no error if not found,
	// or nil and an error if the lookup fails.
	ByName(context.Context, string) (*hcloud.Server, error)
}

// ErrRateLimited is returned by a [ServerCache] when a lookup would have
// required a refresh but the cache's internal rate limiter denied it.
var ErrRateLimited = errors.New("refresh_rate_limited")

type Mode string

const (
	ModeAllServers Mode = "all-server"
	ModePerServer  Mode = "per-server"
	ModeEval       Mode = "eval"
	ModeOff        Mode = "off"
)

func New(client *hcloud.Client, subsystem string, mode Mode, ttl time.Duration) (ServerCache, error) {
	if mode != ModeOff {
		klog.Warningf("instance caching is experimental, breaking changes may occur within minor releases; set HCLOUD_INSTANCES_CACHE_MODE=off to opt out (mode=%q)", mode)
	}
	switch mode {
	case ModeAllServers:
		return NewAllServerCache(client, subsystem, ttl), nil
	case ModePerServer:
		return NewPerServerCache(client, subsystem, ttl), nil
	case ModeEval:
		klog.Warningf("instance cache mode %q is for internal evaluation only and is not intended for production use", mode)
		return NewEvalCache(client, subsystem, ttl), nil
	case ModeOff:
		return NewPassthrough(client), nil
	}
	return nil, fmt.Errorf("invalid cache mode %q", mode)
}

// ----- Passthrough -----

// Passthrough is a [ServerCache] that always queries the API.

var _ ServerCache = (*Passthrough)(nil)

type Passthrough struct {
	client *hcloud.Client
}

func NewPassthrough(client *hcloud.Client) *Passthrough {
	return &Passthrough{client: client}
}

func (c *Passthrough) ByID(ctx context.Context, id int64) (*hcloud.Server, error) {
	server, _, err := c.client.Server.GetByID(ctx, id)
	return server, err
}

func (c *Passthrough) ByName(ctx context.Context, name string) (*hcloud.Server, error) {
	server, _, err := c.client.Server.GetByName(ctx, name)
	return server, err
}
