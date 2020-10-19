package e2etests

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/hetznercloud/hcloud-go/hcloud"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type TestCluster struct {
	once        sync.Once
	useNetworks bool
	setup       *hcloudK8sSetup
	k8sClient   *kubernetes.Clientset
	started     bool
	err         error
}

func (tc *TestCluster) initialize() error {
	const op = "TestCluster/initialize"
	tc.once.Do(func() {
		fmt.Printf("%s: Starting CCM Testsuite\n", op)
		networksSupport := os.Getenv("USE_NETWORKS")
		if networksSupport == "yes" {
			tc.useNetworks = true
		}
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
			tc.err = fmt.Errorf("%s: No valid HCLOUD_TOKEN found\n", op)
			return
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
				additionalSSHKey, _, err := hcloudClient.SSHKey.Get(context.Background(), idOrName)
				if err != nil {
					tc.err = fmt.Errorf("%s:%s\n", op, err)
					return
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
			tc.err = fmt.Errorf("%s:%s\n", op, err)
			return
		}

		fmt.Println("Saving ccm image to disk")
		cmd = exec.Command("docker", "save", "--output", "ci-hcloud-ccm.tar", fmt.Sprintf("hcloud-ccm:ci_%s", testIdentifier))
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			tc.err = fmt.Errorf("%s:%s\n", op, err)
			return
		}

		tc.setup = &hcloudK8sSetup{Hcloud: hcloudClient, K8sVersion: k8sVersion, TestIdentifier: testIdentifier, HcloudToken: token}
		fmt.Println("Setting up test env")

		err = tc.setup.PrepareTestEnv(context.Background(), additionalSSHKeys)
		if err != nil {
			tc.err = fmt.Errorf("%s:%s\n", op, err)
			return
		}

		kubeconfigPath, err := tc.setup.PrepareK8s(tc.useNetworks)
		if err != nil {
			tc.err = fmt.Errorf("%s:%s\n", op, err)
			return
		}

		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			fmt.Printf("%s: clientcmd.BuildConfigFromFlags: %s", op, err)
		}

		tc.k8sClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Printf("%s: kubernetes.NewForConfig: %s", op, err)
		}

		tc.started = true
	})
	return tc.err
}

func (tc *TestCluster) Start() error {
	if err := tc.initialize(); err != nil {
		return fmt.Errorf("start test cluster: %v", err)
	}
	if err := tc.ensureNodesAreReady(); err != nil {
		return fmt.Errorf("start test cluster: %v", err)
	}
	return nil
}

func (tc *TestCluster) Stop() error {
	const op = "TestCluster/Stop"
	if tc.err != nil {
		return fmt.Errorf("%s: %s", op, tc.err)
	}
	if !tc.started {
		return nil
	}
	err := tc.setup.TearDown(context.Background())
	if err != nil {
		fmt.Printf("%s: Tear Down: %s", op, err)
	}
	return nil
}

func (tc *TestCluster) ensureNodesAreReady() error {
	const op = "ensureNodesAreReady"
	err := wait.Poll(1*time.Second, 5*time.Minute, func() (bool, error) {
		nodes, err := tc.k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
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

type lbTestHelper struct {
	podName   string
	K8sClient *kubernetes.Clientset
	t         *testing.T
}

// DeployTestPod deploys a basic nginx pod within the k8s cluster
// and waits until it is "ready"
func (l *lbTestHelper) DeployTestPod() *corev1.Pod {
	const op = "lbTestHelper/DeployTestPod"
	podName := fmt.Sprintf("pod-%s", l.podName)
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"app": podName,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx-hello-world",
					Image: "nginxdemos/hello:plain-text",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
							Name:          "http",
						},
					},
				},
			},
		},
	}

	pod, err := l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Create(context.Background(), &testPod, metav1.CreateOptions{})
	if err != nil {
		l.t.Fatalf("%s: could not create test pod: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		p, err := l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, condition := range p.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		pod = p
		return false, nil
	})
	if err != nil {
		l.t.Fatalf("%s: pod %s did not come up after 1 minute: %s", op, podName, err)
	}
	return pod
}

// ServiceDefinition returns a service definition for a Hetzner Cloud Load Balancer (k8s service)
func (l *lbTestHelper) ServiceDefinition(pod *corev1.Pod, annotations map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("svc-%s", l.podName),
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": pod.Name,
			},
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Port: 80,
					Name: "http",
				},
			},
			ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal,
		},
	}
}

// CreateService creates a k8s service based on the given service definition
// and waits until it is "ready"
func (l *lbTestHelper) CreateService(lbSvc *corev1.Service) (*corev1.Service, error) {
	const op = "lbTestHelper/CreateService"
	_, err := l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Create(context.Background(), lbSvc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: could not create service: %s", op, err)
	}

	err = wait.Poll(1*time.Second, 5*time.Minute, func() (done bool, err error) {
		svc, err := l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Get(context.Background(), lbSvc.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		ingressIPs := svc.Status.LoadBalancer.Ingress
		if len(ingressIPs) > 0 {
			lbSvc = svc
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: test service (load balancer) did not come up after 1 minute: %s", op, err)
	}
	return lbSvc, nil
}

// TearDown deletes the created pod and service
func (l *lbTestHelper) TearDown() {
	const op = "lbTestHelper/TearDown"
	svcName := fmt.Sprintf("svc-%s", l.podName)
	err := l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Delete(context.Background(), svcName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		l.t.Errorf("%s: deleting test svc failed: %s", op, err)
	}

	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		_, err = l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Get(context.Background(), svcName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		l.t.Errorf("%s: test service was not removed after 1 minute: %s", op, err)
	}

	podName := fmt.Sprintf("pod-%s", l.podName)
	err = l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		l.t.Errorf("%s: deleting test pod failed: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		_, err = l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		l.t.Errorf("%s: test pod not removed after 1 minute: %s", op, err)
	}
}

// WaitForHttpAvailable tries to connect to the given IP via http
// It tries it for 2 minutes, if after two minutes the connection
// wasn't successful and it wasn't a HTTP 200 response it will fail
func (l *lbTestHelper) WaitForHttpAvailable(ingressIP string) {
	const op = "lbTestHelper/WaitForHttpAvailable"
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	err := wait.Poll(1*time.Second, 2*time.Minute, func() (bool, error) {
		resp, err := client.Get(fmt.Sprintf("http://%s", ingressIP))
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("%s: got HTTP Code %d instead of 200", op, resp.StatusCode)
		}
		return true, nil
	})
	if err != nil {
		l.t.Errorf("%s: not available via client.Get: %s", op, err)
	}
}

type nwTestHelper struct {
	podName    string
	K8sClient  *kubernetes.Clientset
	privateKey string
	server     *hcloud.Server
	t          *testing.T
}

// DeployTestPod deploys a basic nginx pod within the k8s cluster
// and waits until it is "ready"
func (n *nwTestHelper) DeployTestPod() *corev1.Pod {
	const op = "nwTestHelper/DeployTestPod"
	podName := fmt.Sprintf("pod-%s", n.podName)
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"app": podName,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx-hello-world",
					Image: "nginxdemos/hello:plain-text",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
							Name:          "http",
						},
					},
				},
			},
		},
	}

	pod, err := n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Create(context.Background(), &testPod, metav1.CreateOptions{})
	if err != nil {
		n.t.Fatalf("%s: could not create test pod: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		p, err := n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, condition := range p.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		pod = p
		return false, nil
	})
	if err != nil {
		n.t.Fatalf("%s: pod %s did not come up after 1 minute: %s", op, podName, err)
	}
	pod, err = n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		n.t.Fatalf("%s: could not create test pod: %s", op, err)
	}
	return pod
}

// WaitForHttpAvailable tries to connect to the given IP via curl
// It tries it for 2 minutes, if after two minutes the connection
// wasn't successful and it wasn't a HTTP 200 response it will fail
func (n *nwTestHelper) WaitForHttpAvailable(podIp string) {
	const op = "nwTestHelper/WaitForHttpAvailable"
	err := wait.Poll(1*time.Second, 2*time.Minute, func() (bool, error) {
		err := RunCommandOnServer(n.privateKey, n.server, fmt.Sprintf("curl http://%s", podIp))
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		n.t.Errorf("%s: not available via curl: %s", op, err)
	}
}

// TearDown deletes the created pod
func (n *nwTestHelper) TearDown() {
	const op = "nwTestHelper/TearDown"
	podName := fmt.Sprintf("pod-%s", n.podName)
	err := n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		n.t.Errorf("%s: deleting test pod failed: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		_, err = n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		n.t.Errorf("%s: test pod not removed after 1 minute: %s", op, err)
	}
}
