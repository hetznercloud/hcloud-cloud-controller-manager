package cache

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var serverCacheRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "cloud_controller_manager_server_cache_requests_total",
	Help: "Total cache requests to the Servers API partitioned by subsystem, mode and result.",
}, []string{"subsystem", "mode", "result"})

func init() {
	metrics.GetRegistry().MustRegister(serverCacheRequests)
}

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
		serverCacheRequests,
		defaultMode,
		defaultMaxAge,
	)
}
