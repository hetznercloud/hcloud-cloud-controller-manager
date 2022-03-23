package hcops

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/syself/hetzner-cloud-controller-manager/internal/metrics"
)

// HCloudCertificateClient defines the hcloud-go function related to
// certificate management.
type HCloudCertificateClient interface {
	AllWithOpts(context.Context, hcloud.CertificateListOpts) ([]*hcloud.Certificate, error)
	Get(ctx context.Context, idOrName string) (*hcloud.Certificate, *hcloud.Response, error)
	CreateCertificate(
		ctx context.Context, opts hcloud.CertificateCreateOpts,
	) (hcloud.CertificateCreateResult, *hcloud.Response, error)
}

// CertificateOps implements all operations regarding Hetzner Cloud Certificates.
type CertificateOps struct {
	CertClient HCloudCertificateClient
}

// GetCertificateByNameOrID obtains a certificate from the Hetzner Cloud
// backend using its ID or Name.
//
// If a certificate could not be found the returned error wraps ErrNotFound.
func (co *CertificateOps) GetCertificateByNameOrID(ctx context.Context, idOrName string) (*hcloud.Certificate, error) {
	const op = "hcops/CertificateOps.GetCertificateByNameOrID"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	cert, _, err := co.CertClient.Get(ctx, idOrName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if cert == nil {
		return nil, fmt.Errorf("%s: %w", op, ErrNotFound)
	}
	return cert, nil
}

// GetCertificateByLabel obtains a single certificate by the passed label.
//
// If the label matches more than one certificate a wrapped ErrNonUniqueResult
// is returned. If no certificate could be found a wrapped ErrNotFound is
// returned.
func (co *CertificateOps) GetCertificateByLabel(ctx context.Context, label string) (*hcloud.Certificate, error) {
	const op = "hcops/CertificateOps.GetCertificateByLabel"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	opts := hcloud.CertificateListOpts{ListOpts: hcloud.ListOpts{LabelSelector: label}}
	certs, err := co.CertClient.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", op, err)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrNotFound)
	}
	if len(certs) > 1 {
		return nil, fmt.Errorf("%s: %w", op, ErrNonUniqueResult)
	}
	return certs[0], nil
}

// CreateManagedCertificate creates a managed certificate for domains labeled
// with label.
//
// CreateManagedCertificate returns a wrapped ErrAlreadyExists if the
// certificate already exists.
func (co *CertificateOps) CreateManagedCertificate(
	ctx context.Context, name string, domains []string, labels map[string]string,
) error {
	const op = "hcops/CertificateOps.CreateManagedCertificate"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	opts := hcloud.CertificateCreateOpts{
		Name:        name,
		Type:        hcloud.CertificateTypeManaged,
		DomainNames: domains,
		Labels:      labels,
	}
	_, _, err := co.CertClient.CreateCertificate(ctx, opts)
	if hcloud.IsError(err, hcloud.ErrorCodeUniquenessError) {
		return fmt.Errorf("%s: %w", op, ErrAlreadyExists)
	}
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	return nil
}
