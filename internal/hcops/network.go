package hcops

import (
	"context"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type HCloudNetworkClient interface {
	GetByID(ctx context.Context, id int64) (*hcloud.Network, *hcloud.Response, error)
}
