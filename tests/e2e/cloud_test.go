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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestNodeSetCorrectNodeLabelsAndIPAddresses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	node, err := testCluster.k8sClient.CoreV1().Nodes().Get(ctx, testCluster.ControlNodeName(), metav1.GetOptions{})
	assert.NoError(t, err)

	server, _, err := testCluster.hcloud.Server.Get(ctx, testCluster.ControlNodeName())
	if err != nil {
		return
	}

	labels := node.Labels
	expectedLabels := map[string]string{
		"node.kubernetes.io/instance-type":   server.ServerType.Name,
		"topology.kubernetes.io/region":      server.Datacenter.Location.Name,
		"topology.kubernetes.io/zone":        server.Datacenter.Name,
		"kubernetes.io/hostname":             server.Name,
		"kubernetes.io/os":                   "linux",
		"kubernetes.io/arch":                 "amd64",
		"instance.hetzner.cloud/provided-by": "cloud",
	}
	for expectedLabel, expectedValue := range expectedLabels {
		if labelValue, ok := labels[expectedLabel]; !ok || labelValue != expectedValue {
			t.Errorf("node have a not expected label %s, ok: %v, given value %s, expected value %s", expectedLabel, ok, labelValue, expectedValue)
		}
	}

	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeExternalIP {
			expectedIP := server.PublicNet.IPv4.IP.String()
			if expectedIP != address.Address {
				t.Errorf("Got %s as NodeExternalIP but expected %s", address.Address, expectedIP)
			}
		}
	}

	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			expectedIP := server.PrivateNet[0].IP.String()
			if expectedIP != address.Address {
				t.Errorf("Got %s as NodeInternalIP but expected %s", address.Address, expectedIP)
			}
		}
	}
}

func TestServiceLoadBalancersMinimalSetup(t *testing.T) {
	t.Parallel()

	lbTest := lbTestHelper{
		t:       t,
		podName: "loadbalancer-minimal",
	}

	pod := lbTest.DeployTestPod()

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation): "nbg1",
	})

	lbSvc, err := lbTest.CreateService(lbSvc)
	if assert.NoError(t, err, "deploying test svc") {
		WaitForHTTPAvailable(t, lbSvc.Status.LoadBalancer.Ingress[0].IP, false)
	}

	lbTest.TearDown()
}

func TestServiceLoadBalancersHTTPS(t *testing.T) {
	t.Parallel()

	cert := testCluster.CreateTLSCertificate(t, "loadbalancer-https")
	lbTest := lbTestHelper{
		t:       t,
		podName: "loadbalancer-https",
		port:    443,
	}

	pod := lbTest.DeployTestPod()

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation):            "nbg1",
		string(annotation.LBSvcHTTPCertificates): cert.Name,
		string(annotation.LBSvcProtocol):         "https",
	})

	lbSvc, err := lbTest.CreateService(lbSvc)
	if assert.NoError(t, err, "deploying test svc") {
		WaitForHTTPAvailable(t, lbSvc.Status.LoadBalancer.Ingress[0].IP, true)
	}

	lbTest.TearDown()
}

func TestServiceLoadBalancersHTTPSWithManagedCertificate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if testCluster.certDomain == "" {
		t.Skip("Skipping because CERT_DOMAIN is not set")
	}

	domainName := fmt.Sprintf("%d-ccm-test.%s", rand.Int(), testCluster.certDomain)
	lbTest := lbTestHelper{
		t:       t,
		podName: "loadbalancer-https",
		port:    443,
	}

	pod := lbTest.DeployTestPod()

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation):                                "nbg1",
		string(annotation.LBSvcHTTPCertificateType):                  "managed",
		string(annotation.LBSvcHTTPManagedCertificateDomains):        domainName,
		string(annotation.LBSvcProtocol):                             "https",
		string(annotation.LBSvcHTTPManagedCertificateUseACMEStaging): "true",
	})

	lbSvc, err := lbTest.CreateService(lbSvc)
	if assert.NoError(t, err, "deploying test svc") {
		certs, err := testCluster.hcloud.Certificate.AllWithOpts(ctx, hcloud.CertificateListOpts{
			ListOpts: hcloud.ListOpts{
				LabelSelector: fmt.Sprintf("%s=%s", hcops.LabelServiceUID, lbSvc.ObjectMeta.UID),
			},
		})
		assert.NoError(t, err)
		if assert.Len(t, certs, 1) {
			testCluster.certificates.Add(certs[0].ID)
		}
	}

	lbTest.TearDown()
}

func TestServiceLoadBalancersWithPrivateNetwork(t *testing.T) {
	t.Parallel()

	lbTest := lbTestHelper{t: t, podName: "loadbalancer-private-network"}

	pod := lbTest.DeployTestPod()

	ipRange := &net.IPNet{
		IP:   net.IPv4(10, 0, 0, 0),
		Mask: net.CIDRMask(24, 32),
	}

	lbSvcDefinition := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation):           "nbg1",
		string(annotation.LBUsePrivateIP):       "true",
		string(annotation.PrivateSubnetIPRange): ipRange.String(),
	})

	lbSvc, err := lbTest.CreateService(lbSvcDefinition)
	if assert.NoError(t, err, "deploying test svc") {
		WaitForHTTPAvailable(t, lbSvc.Status.LoadBalancer.Ingress[0].IP, false)

		anyInIPRange := slices.ContainsFunc(lbSvc.Status.LoadBalancer.Ingress, func(ingress corev1.LoadBalancerIngress) bool {
			ip := net.ParseIP(ingress.IP)
			if ip == nil {
				return false
			}
			return ipRange.Contains(ip)
		})

		assert.True(t, anyInIPRange)
	}

	lbTest.TearDown()
}

func TestRouteNetworksPodIPsAreAccessible(t *testing.T) {
	t.Parallel()

	err := wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		node, err := testCluster.k8sClient.CoreV1().Nodes().Get(ctx, testCluster.ControlNodeName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		network, _, err := testCluster.hcloud.Network.Get(ctx, testCluster.NetworkName())
		if err != nil {
			return false, err
		}
		for _, route := range network.Routes {
			if route.Destination.String() == node.Spec.PodCIDR {
				for _, a := range node.Status.Addresses {
					if a.Type == corev1.NodeInternalIP {
						assert.Equal(t, a.Address, route.Gateway.String())
					}
				}
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
