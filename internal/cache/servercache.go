package cache

import (
	"context"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func NewServerCache(client *hcloud.Client, defaultMode Mode, defaultMaxAge time.Duration) *Cache[hcloud.Server] {
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
		defaultMode,
		defaultMaxAge,
	)
}
