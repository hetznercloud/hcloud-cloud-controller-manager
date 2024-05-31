package hcops

import (
	"context"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type HCloudActionClient interface {
	WaitFor(ctx context.Context, actions ...*hcloud.Action) error
}
