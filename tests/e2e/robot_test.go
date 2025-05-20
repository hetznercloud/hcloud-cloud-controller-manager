//go:build e2e && robot

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/providerid"
)

func TestRobotClientIsAvailable(t *testing.T) {
	assert.NotNil(t, testCluster.hrobot)
}

func TestNodeSetCorrectNodeLabelsAndIPAddressesRobot(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Get a random Robot server from all Nodes in the cluster
	nodes, err := testCluster.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "instance.hetzner.cloud/is-root-server=true",
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(nodes.Items), 1)
	node := nodes.Items[0]

	// Parse the server number from the ProviderID
	id, isCloudServer, err := providerid.ToServerID(node.Spec.ProviderID)
	assert.NoError(t, err)
	assert.False(t, isCloudServer)

	// Get the server from the Robot API to cross-check Labels
	server, err := testCluster.hrobot.ServerGet(int(id))
	assert.NoError(t, err)

	labels := node.Labels
	expectedLabels := map[string]string{
		"kubernetes.io/hostname":             server.Name,
		"kubernetes.io/os":                   "linux",
		"kubernetes.io/arch":                 "amd64",
		"instance.hetzner.cloud/provided-by": "robot",
	}
	for expectedLabel, expectedValue := range expectedLabels {
		assert.Equal(t, expectedValue, labels[expectedLabel], "node does not have expected label %s", expectedLabel)
	}

	expectedLabelsSet := []string{
		"node.kubernetes.io/instance-type",
		"topology.kubernetes.io/region",
		"topology.kubernetes.io/zone",
	}
	for _, expectedLabel := range expectedLabelsSet {
		_, ok := labels[expectedLabel]
		assert.True(t, ok, "node is missing expected label %s", expectedLabel)
	}

	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeExternalIP {
			expectedIP := server.ServerIP
			assert.Equal(t, expectedIP, address.Address, "node has unexpected external ip")
		}
	}
}

func TestServiceLoadBalancersRobot(t *testing.T) {
	t.Parallel()

	lbTest := lbTestHelper{
		t:       t,
		podName: "loadbalancer-robot-only",
	}

	pod := lbTest.DeployTestPod()

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation): "nbg1",
		// Only add the Robot server as a Load Balancer target
		string(annotation.LBNodeSelector): "instance.hetzner.cloud/is-root-server=true",
	})

	lbSvc, err := lbTest.CreateService(lbSvc)
	if assert.NoError(t, err, "deploying test svc") {
		WaitForHTTPAvailable(t, lbSvc.Status.LoadBalancer.Ingress[0].IP, false)
	}

	lbTest.TearDown()
}
