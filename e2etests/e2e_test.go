package e2etests

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}
func TestCloudController(t *testing.T) {
	ctx := context.Background()

	networksSupport := os.Getenv("USE_NETWORKS")
	useNetworks := false
	if networksSupport == "yes" {
		useNetworks = true
	}

	var kubeconfigPath string
	kubeconfigPath = os.Getenv("USE_KUBE_CONFIG")

	var setup *hcloudK8sSetup
	if kubeconfigPath == "" {
		path, s, err := prepareTestCluster(ctx, useNetworks)
		assert.NoError(t, err)

		kubeconfigPath = path
		setup = s
	}

	defer func() {
		fmt.Println("Tear Down")
		err := setup.TearDown(context.Background())
		assert.NoError(t, err)
	}()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	assert.NoError(t, err)

	k8sClient, err := kubernetes.NewForConfig(config)
	assert.NoError(t, err)

	t.Run("hcloud-cloud-controller-manager pod is present in kube-system", func(t *testing.T) {
		err = ensureNodesAreReady(ctx, k8sClient)
		assert.NoError(t, err)

		pods, err := k8sClient.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
		assert.NoError(t, err)

		var found = false
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
		err = ensureNodesAreReady(ctx, k8sClient)
		assert.NoError(t, err)

		pods, err := k8sClient.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{LabelSelector: "app=hcloud-cloud-controller-manager"})
		assert.NoError(t, err)

		if len(pods.Items) == 0 {
			t.Fatal("kube-system does not contain a pod with label app=hcloud-cloud-controller-manager")
		}
	})

	t.Run("the node has the correct labels set", func(t *testing.T) {
		err = ensureNodesAreReady(ctx, k8sClient)
		assert.NoError(t, err)

		node, err := k8sClient.CoreV1().Nodes().Get(ctx, setup.server.Name, metav1.GetOptions{})
		assert.NoError(t, err)

		labels := node.Labels
		expectedLabels := map[string]string{
			"node.kubernetes.io/instance-type": setup.server.ServerType.Name,
			"topology.kubernetes.io/region":    setup.server.Datacenter.Location.Name,
			"topology.kubernetes.io/zone":      setup.server.Datacenter.Name,
		}
		for expectedLabel, expectedValue := range expectedLabels {
			if labelValue, ok := labels[expectedLabel]; !ok || labelValue != expectedValue {
				t.Errorf("node have a not expected label %s, ok: %v, given value %s, expected value %s", expectedLabel, ok, labelValue, expectedValue)
			}
		}
	})

	t.Run("load balancers work properly with public interface", func(t *testing.T) {
		err = ensureNodesAreReady(ctx, k8sClient)
		assert.NoError(t, err)
		lbTest := lbTests{TestName: "lb-public", K8sClient: k8sClient}

		pod, err := lbTest.DeployTestPod(ctx)
		assert.NoError(t, err)
		if err != nil {
			t.Fatalf("deploying test pod: %s", err)
		}

		lbSvc := lbTest.ServiceDefinition(pod, map[string]string{
			string(annotation.LBLocation): "nbg1",
		})

		lbSvc, err = lbTest.CreateService(ctx, lbSvc)
		if err != nil {
			t.Fatalf("deploying test svc: %s", err)
		}

		ingressIP := lbSvc.Status.LoadBalancer.Ingress[0].IP // Index 0 is always the public IP of the LB
		err = lbTest.WaitForHttpAvailable(ingressIP)
		assert.NoError(t, err)

		err = lbTest.TearDown(ctx)
		assert.NoError(t, err)
	})

	t.Run("load balancers work properly with private interface", func(t *testing.T) {
		if useNetworks == false {
			t.Skipf("Private Networks test is disabled")
		}
		err = ensureNodesAreReady(ctx, k8sClient)
		assert.NoError(t, err)

		lbTest := lbTests{TestName: "lb-private", K8sClient: k8sClient}

		pod, err := lbTest.DeployTestPod(ctx)
		if err != nil {
			t.Fatalf("deploying test pod: %s", err)
		}

		lbSvcDefinition := lbTest.ServiceDefinition(pod, map[string]string{
			string(annotation.LBLocation):     "nbg1",
			string(annotation.LBUsePrivateIP): "true",
		})

		lbSvc, err := lbTest.CreateService(ctx, lbSvcDefinition)
		if err != nil {
			t.Fatalf("deploying test svc: %s", err)
		}

		ingressIP := lbSvc.Status.LoadBalancer.Ingress[0].IP // Index 0 is always the public IP of the LB
		err = lbTest.WaitForHttpAvailable(ingressIP)
		assert.NoError(t, err)

		err = lbTest.TearDown(ctx)
		assert.NoError(t, err)
	})

	t.Run("private network route advertising works correctly", func(t *testing.T) {
		if useNetworks == false {
			t.Skipf("Private Networks test is disabled")
		}
		err = ensureNodesAreReady(ctx, k8sClient)
		assert.NoError(t, err)

		nwTest := nwTests{TestName: "routes-accessable", K8sClient: k8sClient, server: setup.server, privateKey: setup.privKey}

		pod, err := nwTest.DeployTestPod(ctx)
		if err != nil {
			t.Fatalf("deploying test pod: %s", err)
		}

		err = nwTest.WaitForHttpAvailable(pod.Status.PodIP)
		assert.NoError(t, err)

		err = nwTest.TearDown(ctx)
		assert.NoError(t, err)
	})

}

func ensureNodesAreReady(ctx context.Context, k8sClient *kubernetes.Clientset) error {
	const op = "ensureNodesAreReady"
	err := wait.Poll(1*time.Second, 5*time.Minute, func() (bool, error) {
		nodes, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		available := false
		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					available = true
				}
			}
		}
		return available, nil
	})
	if err != nil {
		return fmt.Errorf("%s: Nodes did not be ready after at least 5 minutes: %s", op, err)
	}
	return nil
}

func prepareTestCluster(ctx context.Context, withNetworks bool) (string, *hcloudK8sSetup, error) {
	const op = "prepareTestCluster"
	fmt.Printf("%s: Starting CCM Testsuite\n", op)
	isUsingGithubActions := os.Getenv("GITHUB_ACTIONS")
	isUsingGitlabCI := os.Getenv("CI_JOB_ID")
	testIdentifier := ""
	if isUsingGithubActions == "true" {
		testIdentifier = fmt.Sprintf("gh-%s-%d", os.Getenv("GITHUB_RUN_ID"), rng.Int())
		fmt.Printf("%s: Running in Github Action\n", op)
	}
	if isUsingGitlabCI != "" {
		testIdentifier = fmt.Sprintf("gl-%s", isUsingGitlabCI)
		fmt.Printf("%s: Running in Gitlab CI\n", op)
	}
	if testIdentifier == "" {
		testIdentifier = fmt.Sprintf("local-%d", rng.Int())
		fmt.Printf("%s: Running local\n", op)
	}

	k8sVersion := os.Getenv("K8S_VERSION")
	if k8sVersion == "" {
		k8sVersion = "1.18.9"
	}
	token := os.Getenv("HCLOUD_TOKEN")
	if len(token) != 64 {
		return "", nil, fmt.Errorf("%s: No valid HCLOUD_TOKEN found\n", op)
	}

	var additionalSSHKeys []*hcloud.SSHKey

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-ccm-testsuite", "1.0"),
	}
	hcloudClient := hcloud.NewClient(opts...)
	additionalSSHKeysIdOrName := os.Getenv("USE_SSH_KEYS")
	if additionalSSHKeysIdOrName != "" {
		idsOrNames := strings.Split(additionalSSHKeysIdOrName, ",")
		for _, idOrName := range idsOrNames {
			additionalSSHKey, _, err := hcloudClient.SSHKey.Get(ctx, idOrName)
			if err != nil {
				return "", nil, fmt.Errorf("%s:%s\n", op, err)
			}
			additionalSSHKeys = append(additionalSSHKeys, additionalSSHKey)
		}

	}

	fmt.Printf("Test against k8s %s\n", k8sVersion)

	fmt.Println("Building ccm image")
	cmd := exec.Command("docker", "build", "-t", fmt.Sprintf("hcloud-ccm:ci_%s", testIdentifier), "../")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return "", nil, fmt.Errorf("%s:%s\n", op, err)
	}

	fmt.Println("Saving ccm image to disk")
	cmd = exec.Command("docker", "save", "--output", "ci-hcloud-ccm.tar", fmt.Sprintf("hcloud-ccm:ci_%s", testIdentifier))
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return "", nil, fmt.Errorf("%s:%s\n", op, err)
	}

	setup := hcloudK8sSetup{Hcloud: hcloudClient, K8sVersion: k8sVersion, TestIdentifier: testIdentifier, HcloudToken: token}
	fmt.Println("Setting up test env")

	err = setup.PrepareTestEnv(ctx, additionalSSHKeys)
	if err != nil {
		return "", nil, fmt.Errorf("%s:%s\n", op, err)
	}

	fmt.Println("Prepare k8s")

	kubeconfigPath, err := setup.PrepareK8s(withNetworks)
	if err != nil {
		return "", nil, fmt.Errorf("%s:%s\n", op, err)
	}
	return kubeconfigPath, &setup, nil
}
