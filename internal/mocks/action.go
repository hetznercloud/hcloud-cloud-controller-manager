package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type ActionClient struct {
	mock.Mock
	hcloud.IActionClient // embedded for compile-time interface satisfaction
}

func (m *ActionClient) WaitFor(ctx context.Context, actions ...*hcloud.Action) error {
	// The mock library does not support variadic arguments, ignore for now
	args := m.Called(ctx, mock.Anything)
	return args.Error(0)
}
