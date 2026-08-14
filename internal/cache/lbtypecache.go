package cache

import (
	"context"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func NewLoadBalancerTypeCache(client *hcloud.Client, defaultMode Mode, defaultMaxAge time.Duration) *Cache[hcloud.LoadBalancerType] {
	return newCache[hcloud.LoadBalancerType](
		func(ctx context.Context, id int64) (*hcloud.LoadBalancerType, error) {
			value, _, err := client.LoadBalancerType.GetByID(ctx, id)
			return value, err
		},
		func(ctx context.Context, name string) (*hcloud.LoadBalancerType, error) {
			value, _, err := client.LoadBalancerType.GetByName(ctx, name)
			return value, err
		},
		func(ctx context.Context) ([]*hcloud.LoadBalancerType, error) {
			values, err := client.LoadBalancerType.All(ctx)
			return values, err
		},
		func(value *hcloud.LoadBalancerType) int64 { return value.ID },
		func(value *hcloud.LoadBalancerType) string { return value.Name },
		defaultMode,
		defaultMaxAge,
	)
}
