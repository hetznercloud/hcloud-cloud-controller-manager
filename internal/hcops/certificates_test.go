package hcops_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestCertificateOps_GetCertificateByNameOrID(t *testing.T) {
	tests := []certificateOpsTestCase{
		{
			Name: "certificate not found",
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				tt.CertClient.
					On("Get", tt.Ctx, "15").
					Return(nil, nil, nil)
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				cert, err := tt.CertOps.GetCertificateByNameOrID(tt.Ctx, "15")
				assert.ErrorIs(t, err, hcops.ErrNotFound)
				assert.Nil(t, cert)
			},
		},
		{
			Name:      "error calling API",
			ClientErr: errors.New("some error"),
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				tt.CertClient.
					On("Get", tt.Ctx, "some-cert").
					Return(nil, nil, tt.ClientErr)
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				cert, err := tt.CertOps.GetCertificateByNameOrID(tt.Ctx, "some-cert")
				assert.ErrorIs(t, err, tt.ClientErr)
				assert.Nil(t, cert)
			},
		},
		{
			Name:        "certificate found",
			Certificate: &hcloud.Certificate{ID: 16},
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				tt.CertClient.
					On("Get", tt.Ctx, "16").
					Return(tt.Certificate, nil, nil)
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				cert, err := tt.CertOps.GetCertificateByNameOrID(tt.Ctx, "16")
				assert.NoError(t, err)
				assert.Equal(t, tt.Certificate, cert)
			},
		},
	}

	runCertificateOpsTestCases(t, tests)
}

func TestCertificateOps_GetCertificateByLabel(t *testing.T) {
	tests := []certificateOpsTestCase{
		{
			Name: "call to backend fails",
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				tt.CertClient.
					On("AllWithOpts", tt.Ctx, hcloud.CertificateListOpts{
						ListOpts: hcloud.ListOpts{LabelSelector: "key=value"},
					}).
					Return(nil, errors.New("test error"))
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				cert, err := tt.CertOps.GetCertificateByLabel(tt.Ctx, "key=value")
				assert.Nil(t, cert)
				assert.Error(t, err)
				assert.True(t, strings.HasSuffix(err.Error(), "test error"))
			},
		},
		{
			Name: "no matching certificate found",
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				tt.CertClient.
					On("AllWithOpts", tt.Ctx, hcloud.CertificateListOpts{
						ListOpts: hcloud.ListOpts{LabelSelector: "key=value"},
					}).
					Return(nil, nil)
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				cert, err := tt.CertOps.GetCertificateByLabel(tt.Ctx, "key=value")
				assert.ErrorIs(t, err, hcops.ErrNotFound)
				assert.Nil(t, cert)
			},
		},
		{
			Name: "more than one matching certificate found",
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				tt.CertClient.
					On("AllWithOpts", tt.Ctx, hcloud.CertificateListOpts{
						ListOpts: hcloud.ListOpts{LabelSelector: "key=value"},
					}).
					Return([]*hcloud.Certificate{{ID: 1}, {ID: 2}}, nil)
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				cert, err := tt.CertOps.GetCertificateByLabel(tt.Ctx, "key=value")
				assert.ErrorIs(t, err, hcops.ErrNonUniqueResult)
				assert.Nil(t, cert)
			},
		},
		{
			Name: "exactly one certificate found",
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				tt.CertClient.
					On("AllWithOpts", tt.Ctx, hcloud.CertificateListOpts{
						ListOpts: hcloud.ListOpts{LabelSelector: "key=value"},
					}).
					Return([]*hcloud.Certificate{{ID: 1}}, nil)
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				cert, err := tt.CertOps.GetCertificateByLabel(tt.Ctx, "key=value")
				assert.NoError(t, err)
				assert.Equal(t, &hcloud.Certificate{ID: 1}, cert)
			},
		},
	}

	runCertificateOpsTestCases(t, tests)
}

func TestCertificateOps_CreateManagedCertificate(t *testing.T) {
	tests := []certificateOpsTestCase{
		{
			Name: "certificate creation fails",
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				tt.CertClient.
					On("CreateCertificate", tt.Ctx, mock.AnythingOfType("hcloud.CertificateCreateOpts")).
					Return(nil, nil, errors.New("test error"))
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				err := tt.CertOps.CreateManagedCertificate(
					tt.Ctx,
					"test-cert",
					[]string{"example.com", "*.example.com"},
					map[string]string{"key": "value"},
				)
				assert.Error(t, err)
				assert.True(t, strings.HasSuffix(err.Error(), "test error"))
			},
		},
		{
			Name: "certificate already exists",
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				err := hcloud.Error{Code: hcloud.ErrorCodeUniquenessError}
				tt.CertClient.
					On("CreateCertificate", tt.Ctx, mock.AnythingOfType("hcloud.CertificateCreateOpts")).
					Return(nil, nil, err)
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				err := tt.CertOps.CreateManagedCertificate(
					tt.Ctx,
					"test-cert",
					[]string{"example.com", "*.example.com"},
					map[string]string{"key": "value"},
				)
				assert.ErrorIs(t, err, hcops.ErrAlreadyExists)
			},
		},
		{
			Name: "certificate creation successful",
			Mock: func(_ *testing.T, tt *certificateOpsTestCase) {
				res := hcloud.CertificateCreateResult{Certificate: &hcloud.Certificate{ID: 1}}
				tt.CertClient.
					On("CreateCertificate", tt.Ctx, hcloud.CertificateCreateOpts{
						Name:        "test-cert",
						Type:        hcloud.CertificateTypeManaged,
						DomainNames: []string{"example.com", "*.example.com"},
						Labels:      map[string]string{"key": "value"},
					}).
					Return(res, nil, nil)
			},
			Perform: func(t *testing.T, tt *certificateOpsTestCase) {
				err := tt.CertOps.CreateManagedCertificate(
					tt.Ctx,
					"test-cert",
					[]string{"example.com", "*.example.com"},
					map[string]string{"key": "value"},
				)
				assert.NoError(t, err)
			},
		},
	}

	runCertificateOpsTestCases(t, tests)
}

type certificateOpsTestCase struct {
	Name        string
	Mock        func(t *testing.T, tt *certificateOpsTestCase)
	Perform     func(t *testing.T, tt *certificateOpsTestCase)
	Certificate *hcloud.Certificate
	ClientErr   error

	// Set in run before actual test execution
	Ctx        context.Context
	CertOps    *hcops.CertificateOps
	CertClient *mocks.CertificateClient
}

func (tt *certificateOpsTestCase) run(t *testing.T) {
	t.Helper()

	tt.Ctx = context.Background()
	tt.CertClient = &mocks.CertificateClient{}
	tt.CertClient.Test(t)
	tt.CertOps = &hcops.CertificateOps{CertClient: tt.CertClient}

	if tt.Mock != nil {
		tt.Mock(t, tt)
	}
	tt.Perform(t, tt)

	tt.CertClient.AssertExpectations(t)
}

func runCertificateOpsTestCases(t *testing.T, tests []certificateOpsTestCase) {
	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, tt.run)
	}
}
