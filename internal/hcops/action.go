package hcops

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

type HCloudActionClient interface {
	WatchProgress(ctx context.Context, a *hcloud.Action) (<-chan int, <-chan error)
}

func WatchAction(ctx context.Context, ac HCloudActionClient, a *hcloud.Action) error {
	_, errCh := ac.WatchProgress(ctx, a)
	err := <-errCh
	return err
}
