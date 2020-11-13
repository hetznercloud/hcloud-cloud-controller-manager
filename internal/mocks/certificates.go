package mocks

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/mock"
)

type CertificateClient struct {
	mock.Mock
}

func (m *CertificateClient) Get(ctx context.Context, idOrName string) (*hcloud.Certificate, *hcloud.Response, error) {
	args := m.Called(ctx, idOrName)
	return getCertificatePtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}
