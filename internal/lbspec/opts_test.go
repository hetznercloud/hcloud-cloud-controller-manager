package lbspec_test

import (
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/lbspec"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestSpecCreateOpts(t *testing.T) {
	lbType := &hcloud.LoadBalancerType{ID: 2, Name: "lb21"}
	labels := map[string]string{lbspec.LabelServiceUID: "some-uid"}

	t.Run("in a location", func(t *testing.T) {
		spec := lbspec.Spec{Name: "some-lb", Labels: labels, Location: "fsn1"}

		opts := spec.CreateOpts(lbType)

		assert.Equal(t, hcloud.LoadBalancerCreateOpts{
			Name:             "some-lb",
			LoadBalancerType: lbType,
			Labels:           labels,
			Location:         &hcloud.Location{Name: "fsn1"},
		}, opts)
	})

	t.Run("in a network zone", func(t *testing.T) {
		spec := lbspec.Spec{Name: "some-lb", Labels: labels, NetworkZone: hcloud.NetworkZoneEUCentral}

		opts := spec.CreateOpts(lbType)

		assert.Nil(t, opts.Location)
		assert.Equal(t, hcloud.NetworkZoneEUCentral, opts.NetworkZone)
	})

	t.Run("with an algorithm and a disabled public interface", func(t *testing.T) {
		spec := lbspec.Spec{
			Name:            "some-lb",
			Labels:          labels,
			Location:        "fsn1",
			Algorithm:       hcloud.LoadBalancerAlgorithmTypeLeastConnections,
			PublicInterface: new(false),
		}

		opts := spec.CreateOpts(lbType)

		assert.Equal(t, &hcloud.LoadBalancerAlgorithm{
			Type: hcloud.LoadBalancerAlgorithmTypeLeastConnections,
		}, opts.Algorithm)
		assert.Equal(t, new(false), opts.PublicInterface)
	})

	t.Run("unconfigured settings are left to the API", func(t *testing.T) {
		spec := lbspec.Spec{Name: "some-lb", Labels: labels, Location: "fsn1"}

		opts := spec.CreateOpts(lbType)

		assert.Nil(t, opts.Algorithm, "an unconfigured algorithm is not sent")
		assert.Nil(t, opts.PublicInterface, "an unconfigured public interface is not sent")
	})
}

func TestSpecUpdateOpts(t *testing.T) {
	required := map[string]string{lbspec.LabelServiceUID: "new-uid"}

	t.Run("nothing to update", func(t *testing.T) {
		lb := &hcloud.LoadBalancer{
			Name:   "some-lb",
			Labels: map[string]string{lbspec.LabelServiceUID: "new-uid"},
		}

		_, update := lbspec.Spec{Name: "some-lb", Labels: required}.UpdateOpts(lb)

		assert.False(t, update)
	})

	t.Run("an unset name leaves the name alone", func(t *testing.T) {
		lb := &hcloud.LoadBalancer{
			Name:   "imported-lb",
			Labels: map[string]string{lbspec.LabelServiceUID: "new-uid"},
		}

		opts, update := lbspec.Spec{Name: "derived-name", NameUnset: true, Labels: required}.UpdateOpts(lb)

		assert.False(t, update)
		assert.Empty(t, opts.Name)
	})

	t.Run("renames the Load Balancer", func(t *testing.T) {
		lb := &hcloud.LoadBalancer{
			Name:   "old-name",
			Labels: map[string]string{lbspec.LabelServiceUID: "new-uid"},
		}

		opts, update := lbspec.Spec{Name: "new-name", Labels: required}.UpdateOpts(lb)

		require.True(t, update)
		assert.Equal(t, "new-name", opts.Name)
		assert.Nil(t, opts.Labels, "the labels are already correct")
	})

	t.Run("adds the required labels to a Load Balancer without any", func(t *testing.T) {
		lb := &hcloud.LoadBalancer{Name: "some-lb"}

		opts, update := lbspec.Spec{Name: "some-lb", Labels: required}.UpdateOpts(lb)

		require.True(t, update)
		assert.Equal(t, map[string]string{lbspec.LabelServiceUID: "new-uid"}, opts.Labels)
	})

	t.Run("replaces a stale label and keeps the others", func(t *testing.T) {
		// A Service that was deleted and recreated keeps its Load Balancer, which
		// then carries the UID of the previous Service. The required label has to
		// win, or every reconcile writes the stale value back.
		lb := &hcloud.LoadBalancer{
			Name: "some-lb",
			Labels: map[string]string{
				lbspec.LabelServiceUID: "old-uid",
				"team":                 "keep-me",
			},
		}

		opts, update := lbspec.Spec{Name: "some-lb", Labels: required}.UpdateOpts(lb)

		require.True(t, update)
		assert.Equal(t, map[string]string{
			lbspec.LabelServiceUID: "new-uid",
			"team":                 "keep-me",
		}, opts.Labels)
		assert.Equal(t, "old-uid", lb.Labels[lbspec.LabelServiceUID],
			"the Load Balancer is only updated once the API call succeeds")
	})
}

func TestSpecChangeAlgorithmOpts(t *testing.T) {
	spec := lbspec.Spec{Algorithm: hcloud.LoadBalancerAlgorithmTypeLeastConnections}

	assert.Equal(t, hcloud.LoadBalancerChangeAlgorithmOpts{
		Type: hcloud.LoadBalancerAlgorithmTypeLeastConnections,
	}, spec.ChangeAlgorithmOpts())
}

func TestSpecAttachToNetworkOpts(t *testing.T) {
	network := &hcloud.Network{ID: 4711}

	t.Run("without an address or subnet", func(t *testing.T) {
		opts := lbspec.Spec{}.AttachToNetworkOpts(network)

		assert.Equal(t, network, opts.Network)
		assert.Nil(t, opts.IP, "the API picks an address")
		assert.Nil(t, opts.IPRange)
	})

	t.Run("with an address and a subnet", func(t *testing.T) {
		spec := lbspec.Spec{
			PrivateIPv4:          netip.MustParseAddr("10.0.1.5"),
			PrivateSubnetIPRange: netip.MustParsePrefix("10.0.1.0/24"),
		}

		opts := spec.AttachToNetworkOpts(network)

		assert.True(t, net.ParseIP("10.0.1.5").Equal(opts.IP))
		require.NotNil(t, opts.IPRange)
		assert.Equal(t, "10.0.1.0/24", opts.IPRange.String())
	})
}

func TestSpecAddServerTargetOpts(t *testing.T) {
	for _, usePrivateIP := range []bool{true, false} {
		spec := lbspec.Spec{UsePrivateIP: usePrivateIP}

		opts := spec.AddServerTargetOpts(42)

		assert.Equal(t, &hcloud.Server{ID: 42}, opts.Server)
		assert.Equal(t, new(usePrivateIP), opts.UsePrivateIP)
	}
}

func TestManagedCertificateCreateOpts(t *testing.T) {
	t.Run("certificate for the configured domains", func(t *testing.T) {
		cert := lbspec.ManagedCertificate{
			Name:    "some-cert",
			Labels:  map[string]string{lbspec.LabelServiceUID: "some-uid"},
			Domains: []string{"example.com", "*.example.com"},
		}

		opts := cert.CreateOpts()

		assert.Equal(t, hcloud.CertificateCreateOpts{
			Name:        "some-cert",
			Type:        hcloud.CertificateTypeManaged,
			DomainNames: []string{"example.com", "*.example.com"},
			Labels:      map[string]string{lbspec.LabelServiceUID: "some-uid"},
		}, opts)
	})

	t.Run("ACME staging is requested through a label", func(t *testing.T) {
		cert := lbspec.ManagedCertificate{
			Name:           "some-cert",
			Labels:         map[string]string{lbspec.LabelServiceUID: "some-uid"},
			Domains:        []string{"example.com"},
			UseACMEStaging: true,
		}

		opts := cert.CreateOpts()

		assert.Equal(t, map[string]string{
			lbspec.LabelServiceUID: "some-uid",
			"HC-Use-Staging-CA":    "true",
		}, opts.Labels)
		assert.Equal(t, map[string]string{lbspec.LabelServiceUID: "some-uid"}, cert.Labels,
			"the labels of the spec are not modified")
	})
}
