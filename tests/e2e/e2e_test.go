package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
	"github.com/syself/hetzner-cloud-controller-manager/internal/annotation"
	"github.com/syself/hetzner-cloud-controller-manager/internal/hcops"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var testCluster TestCluster

func TestMain(m *testing.M) {
	fmt.Printf("the e2e tests seem to require a special setup. They are disabled in our fork.\n")
	os.Exit(1)
	if err := testCluster.Start(); err != nil {
		fmt.Printf("testCluster.Start failed: %v\n", err)
		os.Exit(1)
	}

	rc := m.Run()

	if err := testCluster.Stop(); err != nil {
		fmt.Printf("testCluster.Stop failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(rc)
}

func TestPodIsPresent(t *testing.T) {
	t.Parallel()

	t.Run("hcloud-cloud-controller-manager pod is present in kube-system", func(t *testing.T) {
		pods, err := testCluster.k8sClient.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{})
		assert.NoError(t, err)

		found := false
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "hcloud-cloud-controller-manager") {
				found = true
				break
			}
		}
		if !found {
			t.Error("kube-system does not contain a pod named hcloud-cloud-controller-manager")
		}
	})

	t.Run("pod with app=hcloud-cloud-controller-manager is present in kube-system", func(t *testing.T) {
		pods, err := testCluster.k8sClient.CoreV1().Pods("kube-system").
			List(context.Background(), metav1.ListOptions{
				LabelSelector: "app.kubernetes.io/name=hcloud-cloud-controller-manager",
			})
		assert.NoError(t, err)

		if len(pods.Items) == 0 {
			t.Fatal("kube-system does not contain a pod with label app=hcloud-cloud-controller-manager")
		}
	})
}

func TestNodeSetCorrectNodeLabelsAndIPAddresses(t *testing.T) {
	t.Parallel()

	node, err := testCluster.k8sClient.CoreV1().Nodes().Get(context.Background(), "hccm-"+testCluster.scope+"-1", metav1.GetOptions{})
	assert.NoError(t, err)

	server, _, err := testCluster.hcloud.Server.Get(context.TODO(), "hccm-"+testCluster.scope+"-1")
	if err != nil {
		return
	}

	labels := node.Labels
	expectedLabels := map[string]string{
		"node.kubernetes.io/instance-type": server.ServerType.Name,
		"topology.kubernetes.io/region":    server.Datacenter.Location.Name,
		"topology.kubernetes.io/zone":      server.Datacenter.Name,
		"kubernetes.io/hostname":           server.Name,
		"kubernetes.io/os":                 "linux",
		"kubernetes.io/arch":               "amd64",
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
		t:         t,
		K8sClient: testCluster.k8sClient,
		podName:   "loadbalancer-minimal",
	}

	pod := lbTest.DeployTestPod()

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation): "nbg1",
	})

	lbSvc, err := lbTest.CreateService(lbSvc)
	if err != nil {
		t.Fatalf("deploying test svc: %s", err)
	}

	WaitForHTTPAvailable(t, lbSvc.Status.LoadBalancer.Ingress[0].IP, false)

	lbTest.TearDown()
}

func TestServiceLoadBalancersHTTPS(t *testing.T) {
	t.Parallel()

	cert := testCluster.CreateTLSCertificate(t, "loadbalancer-https")
	lbTest := lbTestHelper{
		t:         t,
		K8sClient: testCluster.k8sClient,
		podName:   "loadbalancer-https",
		port:      443,
	}

	pod := lbTest.DeployTestPod()

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation):            "nbg1",
		string(annotation.LBSvcHTTPCertificates): cert.Name,
		string(annotation.LBSvcProtocol):         "https",
	})

	lbSvc, err := lbTest.CreateService(lbSvc)
	if err != nil {
		t.Fatalf("deploying test svc: %s", err)
	}

	WaitForHTTPAvailable(t, lbSvc.Status.LoadBalancer.Ingress[0].IP, true)

	lbTest.TearDown()
}

func TestServiceLoadBalancersHTTPSWithManagedCertificate(t *testing.T) {
	t.Parallel()

	if testCluster.certDomain == "" {
		t.Skip("Skipping because CERT_DOMAIN is not set")
	}

	domainName := fmt.Sprintf("%d-ccm-test.%s", rand.Int(), testCluster.certDomain)
	lbTest := lbTestHelper{
		t:         t,
		K8sClient: testCluster.k8sClient,
		podName:   "loadbalancer-https",
		port:      443,
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
	if err != nil {
		t.Fatalf("deploying test svc: %s", err)
	}
	certs, err := testCluster.hcloud.Certificate.AllWithOpts(context.Background(), hcloud.CertificateListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: fmt.Sprintf("%s=%s", hcops.LabelServiceUID, lbSvc.ObjectMeta.UID),
		},
	})
	assert.NoError(t, err)
	assert.Len(t, certs, 1)

	lbTest.TearDown()
	_, err = testCluster.hcloud.Certificate.Delete(context.Background(), certs[0])
	assert.NoError(t, err)
}

func TestServiceLoadBalancersWithPrivateNetwork(t *testing.T) {
	t.Parallel()

	lbTest := lbTestHelper{t: t, K8sClient: testCluster.k8sClient, podName: "loadbalancer-private-network"}

	pod := lbTest.DeployTestPod()

	lbSvcDefinition := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation):     "nbg1",
		string(annotation.LBUsePrivateIP): "true",
	})

	lbSvc, err := lbTest.CreateService(lbSvcDefinition)
	if err != nil {
		t.Fatalf("deploying test svc: %s", err)
	}

	WaitForHTTPAvailable(t, lbSvc.Status.LoadBalancer.Ingress[0].IP, false)

	lbTest.TearDown()
}

func TestRouteNetworksPodIPsAreAccessible(t *testing.T) {
	t.Parallel()

	err := wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		node, err := testCluster.k8sClient.CoreV1().Nodes().Get(ctx, "hccm-"+testCluster.scope+"-1", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		network, _, err := testCluster.hcloud.Network.Get(context.TODO(), "hccm-"+testCluster.scope)
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

func TestRouteDeleteCorrectRoutes(t *testing.T) {
	t.Parallel()

	// This test tests for:
	// - hccm is keeping routes that are outside of its scope (e.g. 0.0.0.0/0)
	// - hccm is deleting routes that are inside the cluster cidr, but for which no server exists (10.254.0.0/24)
	//
	// Testing for no-op (keep route) is hard, but by combining the test with a deletion test we can be reasonably sure
	// that it was not tried in the same reconcile loop.

	ctx := context.Background()

	testGateway := net.ParseIP("10.0.0.254")
	_, defaultDestination, err := net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		t.Fatal(err)
	}
	_, outdatedDestination, err := net.ParseCIDR("10.244.254.0/24")
	if err != nil {
		t.Fatal(err)
	}

	network, _, err := testCluster.hcloud.Network.Get(ctx, "hccm-"+testCluster.scope)
	if err != nil {
		t.Fatal(err)
	}

	// Remove routes from previous tests
	for _, route := range network.Routes {
		if route.Gateway.Equal(testGateway) {
			action, _, err := testCluster.hcloud.Network.DeleteRoute(ctx, network, hcloud.NetworkDeleteRouteOpts{Route: route})
			if err != nil {
				t.Fatal(err)
			}
			if err := hcops.WatchAction(ctx, &testCluster.hcloud.Action, action); err != nil {
				t.Fatal(err)
			}
		}
	}

	action, _, err := testCluster.hcloud.Network.AddRoute(ctx, network, hcloud.NetworkAddRouteOpts{Route: hcloud.NetworkRoute{
		Destination: defaultDestination,
		Gateway:     testGateway,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := hcops.WatchAction(ctx, &testCluster.hcloud.Action, action); err != nil {
		t.Fatal(err)
	}

	action, _, err = testCluster.hcloud.Network.AddRoute(ctx, network, hcloud.NetworkAddRouteOpts{Route: hcloud.NetworkRoute{
		Destination: outdatedDestination,
		Gateway:     testGateway,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := hcops.WatchAction(ctx, &testCluster.hcloud.Action, action); err != nil {
		t.Fatal(err)
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		network, _, err = testCluster.hcloud.Network.Get(ctx, "hccm-"+testCluster.scope)
		if err != nil {
			return false, err
		}

		hasDefaultRoute := false
		for _, route := range network.Routes {
			switch route.Destination.String() {
			case defaultDestination.String():
				hasDefaultRoute = true
			case outdatedDestination.String():
				// Route for outdated destination still exists
				return false, nil
			}
		}
		return hasDefaultRoute, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
