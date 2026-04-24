//go:build e2e && robot

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Get a random Robot server from all Nodes in the cluster
	nodes, err := testCluster.k8sClient.CoreV1().Nodes().List(t.Context(), metav1.ListOptions{
		LabelSelector: "instance.hetzner.cloud/is-root-server=true",
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(nodes.Items), 1)
	node := nodes.Items[0]

	// Parse the server number from the ProviderID
	id, isCloudServer, err := providerid.ToServerID(node.Spec.ProviderID)
	require.NoError(t, err)
	assert.False(t, isCloudServer)

	// Get the server from the Robot API to cross-check Labels
	server, err := testCluster.hrobot.ServerGet(int(id))
	require.NoError(t, err)

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
	t.Cleanup(func() {
		lbTest.TearDown()
	})

	pod, err := lbTest.DeployTestPod()
	require.NoError(t, err)

	lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
		string(annotation.LBLocation): "nbg1",
		// Only add the Robot server as a Load Balancer target
		string(annotation.LBNodeSelector): "instance.hetzner.cloud/is-root-server=true",
	})

	lbSvc, err = lbTest.CreateService(lbSvc, 8*time.Minute)
	require.NoError(t, err)

	err = lbTest.WaitForHTTPAvailable(lbSvc.Status.LoadBalancer.Ingress[0].IP, false)
	require.NoError(t, err)
}
