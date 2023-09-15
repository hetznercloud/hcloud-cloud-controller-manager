package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// ServerClient is a mock implementation of the hcloud.ServerClient.
type ServerClient struct {
	mock.Mock
	T *testing.T
}

// NewServerClient creates a new mock server client ready for use.
func NewServerClient(t *testing.T) *ServerClient {
	m := &ServerClient{T: t}
	m.Test(t)
	return m
}

// All registers a call to obtain all servers from the Hetzner Cloud API.
func (m *ServerClient) All(ctx context.Context) ([]*hcloud.Server, error) {
	args := m.Called(ctx)
	return serverPtrSlice(m.T, args.Get(0)), args.Error(1)
}

func serverPtrSlice(t *testing.T, v interface{}) []*hcloud.Server {
	const op = "mocks/serverPtrSlice"

	t.Helper()

	if v == nil {
		return nil
	}
	ss, ok := v.([]*hcloud.Server)
	if !ok {
		t.Fatalf("%s: not a []*Server: %t", op, v)
	}
	return ss
}
