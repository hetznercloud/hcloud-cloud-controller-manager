package cache

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var lbTypeCacheRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "cloud_controller_manager",
	Subsystem: "load_balancer_type",
	Name: "cache_requests_total",
	Help: "Total cache requests to the Load Balancer Types API partitioned by subsystem, mode and result.",
}, []string{"subsystem", "mode", "result"})

func init() {
	metrics.GetRegistry().MustRegister(lbTypeCacheRequests)
}

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
		lbTypeCacheRequests,
		defaultMode,
		defaultMaxAge,
	)
}
