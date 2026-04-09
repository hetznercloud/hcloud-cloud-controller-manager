package servercache

import (
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

// newCacheTestClient spins up a [mockutil.Server] and returns a client pointed
// at it.
func newCacheTestClient(t *testing.T, requests []mockutil.Request) *hcloud.Client {
	t.Helper()
	server := mockutil.NewServer(t, requests)
	return hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithPollOpts(hcloud.PollOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
	)
}

var notFoundResponse = schema.ErrorResponse{Error: schema.Error{Code: string(hcloud.ErrorCodeNotFound), Message: "not found"}}
