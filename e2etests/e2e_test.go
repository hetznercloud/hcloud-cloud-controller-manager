package e2etests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/stretchr/testify/assert"
	typesv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCluster TestCluster

func TestMain(m *testing.M) {
	if err := testCluster.Start(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	rc := m.Run()

	if err := testCluster.Stop(); err != nil {
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
		pods, err := testCluster.k8sClient.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{LabelSelector: "app=hcloud-cloud-controller-manager"})
		assert.NoError(t, err)

		if len(pods.Items) == 0 {
			t.Fatal("kube-system does not contain a pod with label app=hcloud-cloud-controller-manager")
		}
	})
}

func TestCloudControllerManagerSetCorrectNodeLabelsAndIPAddresses(t *testing.T) {
	node, err := testCluster.k8sClient.CoreV1().Nodes().Get(context.Background(), testCluster.setup.server.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	labels := node.Labels
	expectedLabels := map[string]string{
		"node.kubernetes.io/instance-type": testCluster.setup.server.ServerType.Name,
		"topology.kubernetes.io/region":    testCluster.setup.server.Datacenter.Location.Name,
		"topology.kubernetes.io/zone":      testCluster.setup.server.Datacenter.Name,
		"kubernetes.io/hostname":           testCluster.setup.server.Name,
		"kubernetes.io/os":                 "linux",
		"kubernetes.io/arch":               "amd64",
	}
	for expectedLabel, expectedValue := range expectedLabels {
		if labelValue, ok := labels[expectedLabel]; !ok || labelValue != expectedValue {
			t.Errorf("node have a not expected label %s, ok: %v, given value %s, expected value %s", expectedLabel, ok, labelValue, expectedValue)
		}
	}

	for _, address := range node.Status.Addresses {
		switch address.Type {
		case typesv1.NodeExternalIP:
			expectedIP := testCluster.setup.server.PublicNet.IPv4.IP.String()
			if expectedIP != address.Address {
				t.Errorf("Got %s as NodeExternalIP but expected %s", address.Address, expectedIP)
			}
		}
	}
	if testCluster.useNetworks {
		for _, address := range node.Status.Addresses {
			switch address.Type {
			case typesv1.NodeInternalIP:
				expectedIP := testCluster.setup.server.PrivateNet[0].IP.String()
				if expectedIP != address.Address {
					t.Errorf("Got %s as NodeInternalIP but expected %s", address.Address, expectedIP)
				}
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

	ingressIP := lbSvc.Status.LoadBalancer.Ingress[0].IP // Index 0 is always the public IP of the LB
	lbTest.WaitForHttpAvailable(ingressIP)

	lbTest.TearDown()
}

func TestCloudControllerManagerLoadBalancersWithPrivateNetwork(t *testing.T) {
	if testCluster.useNetworks == false {
		t.Skipf("Private Networks test is disabled")
	}

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
	lbTest.WaitForHttpAvailable(ingressIP)

	lbTest.TearDown()
}

func TestCloudControllerManagerNetworksPodIPsAreAccessible(t *testing.T) {
	if testCluster.useNetworks == false {
		t.Skipf("Private Networks test is disabled")
	}

	nwTest := nwTestHelper{t: t, K8sClient: testCluster.k8sClient, server: testCluster.setup.server, privateKey: testCluster.setup.privKey, podName: "network-routes-accessible"}

	pod := nwTest.DeployTestPod()

	nwTest.WaitForHttpAvailable(pod.Status.PodIP)

	nwTest.TearDown()
}
