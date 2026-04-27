//go:build e2e && !robot

package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/legacydatacenter"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestNodeSetCorrectNodeLabelsAndIPAddresses(t *testing.T) {
	t.Parallel()

	node, err := testCluster.k8sClient.CoreV1().Nodes().Get(t.Context(), testCluster.ControlNodeName(), metav1.GetOptions{})
	require.NoError(t, err)

	server, _, err := testCluster.hcloud.Server.Get(t.Context(), testCluster.ControlNodeName())
	require.NoError(t, err)

	expectedLabels := map[string]string{
		"node.kubernetes.io/instance-type":   server.ServerType.Name,
		"topology.kubernetes.io/region":      server.Location.Name,
		"topology.kubernetes.io/zone":        legacydatacenter.NameFromLocation(server.Location.Name),
		"kubernetes.io/hostname":             server.Name,
		"kubernetes.io/os":                   "linux",
		"kubernetes.io/arch":                 "amd64",
		"instance.hetzner.cloud/provided-by": "cloud",
	}
	for expectedLabel, expectedValue := range expectedLabels {
		assert.Equal(t, expectedValue, node.Labels[expectedLabel], "unexpected value for label %s", expectedLabel)
	}

	for _, address := range node.Status.Addresses {
		switch address.Type {
		case corev1.NodeExternalIP:
			assert.Equal(t, server.PublicNet.IPv4.IP.String(), address.Address, "unexpected NodeExternalIP")
		case corev1.NodeInternalIP:
			assert.Equal(t, server.PrivateNet[0].IP.String(), address.Address, "unexpected NodeInternalIP")
		}
	}
}

func TestServiceLoadBalancersMinimalSetup(t *testing.T) {
	t.Parallel()

	lbTest := lbTestHelper{
		t:       t,
		podName: "loadbalancer-minimal",
	}
	t.Cleanup(func() {
		lbTest.TearDown()
	})

	pod, err := lbTest.DeployTestPod()
	require.NoError(t, err)

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation): "nbg1",
	})

	lbSvc, err = lbTest.CreateService(lbSvc)
	require.NoError(t, err)

	err = lbTest.WaitForHTTPAvailable(lbSvc.Status.LoadBalancer.Ingress[0].IP, false)
	require.NoError(t, err)
}

func TestServiceLoadBalancersHTTPS(t *testing.T) {
	lbTest := lbTestHelper{
		t:       t,
		podName: "loadbalancer-https",
		port:    443,
	}
	t.Cleanup(func() {
		lbTest.TearDown()
	})

	cert, err := testCluster.CreateTLSCertificate(t, "loadbalancer-https")
	require.NoError(t, err)

	pod, err := lbTest.DeployTestPod()
	require.NoError(t, err)

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation):            "nbg1",
		string(annotation.LBSvcHTTPCertificates): cert.Name,
		string(annotation.LBSvcProtocol):         "https",
	})

	lbSvc, err = lbTest.CreateService(lbSvc)
	require.NoError(t, err)

	err = lbTest.WaitForHTTPAvailable(lbSvc.Status.LoadBalancer.Ingress[0].IP, true)
	require.NoError(t, err)
}

func TestServiceLoadBalancersHTTPSWithManagedCertificate(t *testing.T) {
	if testCluster.certDomain == "" {
		t.Skip("Skipping because CERT_DOMAIN is not set")
	}

	domainName := fmt.Sprintf("%d-ccm-test.%s", rand.Int(), testCluster.certDomain)
	lbTest := lbTestHelper{
		t:       t,
		podName: "loadbalancer-https",
		port:    443,
	}
	t.Cleanup(func() {
		lbTest.TearDown()
	})

	pod, err := lbTest.DeployTestPod()
	require.NoError(t, err)

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation):                                "nbg1",
		string(annotation.LBSvcHTTPCertificateType):                  "managed",
		string(annotation.LBSvcHTTPManagedCertificateDomains):        domainName,
		string(annotation.LBSvcProtocol):                             "https",
		string(annotation.LBSvcHTTPManagedCertificateUseACMEStaging): "true",
	})

	lbSvc, err = lbTest.CreateService(lbSvc)
	require.NoError(t, err)

	certs, err := testCluster.hcloud.Certificate.AllWithOpts(t.Context(), hcloud.CertificateListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: fmt.Sprintf("%s=%s", hcops.LabelServiceUID, lbSvc.ObjectMeta.UID),
		},
	})
	assert.NoError(t, err)
	if assert.Len(t, certs, 1) {
		testCluster.certificates.Add(certs[0].ID)
	}
}

func TestServiceLoadBalancersWithPrivateNetwork(t *testing.T) {
	t.Parallel()

	lbTest := lbTestHelper{t: t, podName: "loadbalancer-private-network"}
	t.Cleanup(func() {
		lbTest.TearDown()
	})

	pod, err := lbTest.DeployTestPod()
	require.NoError(t, err)

	ipRange := &net.IPNet{
		IP:   net.IPv4(10, 0, 0, 0),
		Mask: net.CIDRMask(24, 32),
	}

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation):           "nbg1",
		string(annotation.LBUsePrivateIP):       "true",
		string(annotation.PrivateSubnetIPRange): ipRange.String(),
	})

	lbSvc, err = lbTest.CreateService(lbSvc)
	require.NoError(t, err)

	err = lbTest.WaitForHTTPAvailable(lbSvc.Status.LoadBalancer.Ingress[0].IP, false)
	require.NoError(t, err)

	anyInIPRange := slices.ContainsFunc(lbSvc.Status.LoadBalancer.Ingress, func(ingress corev1.LoadBalancerIngress) bool {
		ip := net.ParseIP(ingress.IP)
		if ip == nil {
			return false
		}
		return ipRange.Contains(ip)
	})
	assert.True(t, anyInIPRange)
}

func TestRouteNetworksPodIPsAreAccessible(t *testing.T) {
	t.Parallel()

	var (
		nodeInternalIP string
		routeGateway   string
	)
	err := wait.PollUntilContextTimeout(t.Context(), 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		node, err := testCluster.k8sClient.CoreV1().Nodes().Get(ctx, testCluster.ControlNodeName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		network, _, err := testCluster.hcloud.Network.Get(ctx, testCluster.NetworkName())
		if err != nil {
			return false, err
		}
		for _, route := range network.Routes {
			if route.Destination.String() != node.Spec.PodCIDR {
				continue
			}
			routeGateway = route.Gateway.String()
			for _, a := range node.Status.Addresses {
				if a.Type == corev1.NodeInternalIP {
					nodeInternalIP = a.Address
					break
				}
			}
			return true, nil
		}
		return false, nil
	})
	require.NoError(t, err, "error waiting for pod IPs being accessible")
	assert.Equal(t, nodeInternalIP, routeGateway, "route gateway should match node internal IP")
}
