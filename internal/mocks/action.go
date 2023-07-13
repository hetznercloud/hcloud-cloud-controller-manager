package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type ActionClient struct {
	mock.Mock
}

func (m *ActionClient) WatchProgress(ctx context.Context, a *hcloud.Action) (<-chan int, <-chan error) {
	args := m.Called(ctx, a)
	return getIntChan(args, 0), getErrChan(args, 1)
}

func (m *ActionClient) MockWatchProgress(ctx context.Context, a *hcloud.Action, err error) {
	resC := make(chan int)
	errC := make(chan error, 1)
	if err != nil {
		errC <- err
	}
	close(resC)
	close(errC)
	m.On("WatchProgress", ctx, a).Return(resC, errC)
}
