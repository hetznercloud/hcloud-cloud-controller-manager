package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type CertificateClient struct {
	mock.Mock
}

func (m *CertificateClient) AllWithOpts(
	ctx context.Context, opts hcloud.CertificateListOpts,
) ([]*hcloud.Certificate, error) {
	args := m.Called(ctx, opts)
	return getCertificatePtrS(args, 0), args.Error(1)
}

func (m *CertificateClient) Get(ctx context.Context, idOrName string) (*hcloud.Certificate, *hcloud.Response, error) {
	args := m.Called(ctx, idOrName)
	return getCertificatePtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *CertificateClient) CreateCertificate(
	ctx context.Context, opts hcloud.CertificateCreateOpts,
) (hcloud.CertificateCreateResult, *hcloud.Response, error) {
	args := m.Called(ctx, opts)
	return getCertificateCreateResult(args, 0), getResponsePtr(args, 1), args.Error(2)
}
