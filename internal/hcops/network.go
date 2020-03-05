package hcops

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

type HCloudNetworkClient interface {
	GetByID(ctx context.Context, id int) (*hcloud.Network, *hcloud.Response, error)
}
