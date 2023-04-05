package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

var testCluster TestCluster

func TestMain(m *testing.M) {
	if err := testCluster.Start(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	rc := m.Run()

	if err := testCluster.Stop(rc > 0); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	os.Exit(rc)
}

func TestCloudControllerManagerPodIsPresent(t *testing.T) {
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

func TestCloudControllerManagerSetCorrectNodeLabelsAndIPAddresses(t *testing.T) {
	node, err := testCluster.k8sClient.CoreV1().Nodes().Get(context.Background(), testCluster.scope+"-1", metav1.GetOptions{})
	assert.NoError(t, err)

	server, _, err := testCluster.hcloud.Server.Get(context.TODO(), testCluster.scope+"-1")
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

func TestCloudControllerManagerLoadBalancersMinimalSetup(t *testing.T) {
	lbTest := lbTestHelper{t: t, K8sClient: testCluster.k8sClient, podName: "loadbalancer-minimal"}

	pod := lbTest.DeployTestPod()

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation): "nbg1",
	})

	lbSvc, err := lbTest.CreateService(lbSvc)
	if err != nil {
		t.Fatalf("deploying test svc: %s", err)
	}

	for _, ing := range lbSvc.Status.LoadBalancer.Ingress {
		fmt.Println(ing.IP)
		WaitForHTTPAvailable(t, ing.IP, false)
	}

	lbTest.TearDown()
}

func TestCloudControllerManagerLoadBalancersHTTPS(t *testing.T) {
	cert := testCluster.CreateTLSCertificate(t, "loadbalancer-https")
	lbTest := lbTestHelper{
		t:             t,
		K8sClient:     testCluster.k8sClient,
		KeepOnFailure: testCluster.KeepOnFailure,
		podName:       "loadbalancer-https",
		port:          443,
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

	ingressIP := lbSvc.Status.LoadBalancer.Ingress[0].IP // Index 0 is always the public IP of the LB
	WaitForHTTPAvailable(t, ingressIP, true)

	// TODO:
	//for _, ing := range lbSvc.Status.LoadBalancer.Ingress {
	//	WaitForHTTPOnServer(t, testCluster.setup.ExtServer, testCluster.setup.privKey, ing.IP, true)
	//}

	lbTest.TearDown()
}

func TestCloudControllerManagerLoadBalancersHTTPSWithManagedCertificate(t *testing.T) {
	domainName := fmt.Sprintf("%d-ccm-test.hc-certs.de", rand.Int())
	lbTest := lbTestHelper{
		t:             t,
		K8sClient:     testCluster.k8sClient,
		KeepOnFailure: testCluster.KeepOnFailure,
		podName:       "loadbalancer-https",
		port:          443,
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

func TestCloudControllerManagerLoadBalancersWithPrivateNetwork(t *testing.T) {
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

	ingressIP := lbSvc.Status.LoadBalancer.Ingress[0].IP // Index 0 is always the public IP of the LB
	WaitForHTTPAvailable(t, ingressIP, false)

	lbTest.TearDown()
}

func TestCloudControllerManagerNetworksPodIPsAreAccessible(t *testing.T) {
	node, err := testCluster.k8sClient.CoreV1().Nodes().Get(context.Background(), testCluster.scope+"-1", metav1.GetOptions{})

	network, _, err := testCluster.hcloud.Network.Get(context.TODO(), testCluster.scope)
	if err != nil {
		t.Error(err)
	}
	for _, subnet := range network.Subnets {
		if subnet.IPRange.String() == node.Spec.PodCIDR {
			return
		}
	}
	t.Fatal("did not find expected route table entry")
}
