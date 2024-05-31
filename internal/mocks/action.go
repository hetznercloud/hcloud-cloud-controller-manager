package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type ActionClient struct {
	mock.Mock
}

func (m *ActionClient) WaitFor(ctx context.Context, actions ...*hcloud.Action) error {
	args := m.Called(ctx, actions)
	return args.Error(0)
}
