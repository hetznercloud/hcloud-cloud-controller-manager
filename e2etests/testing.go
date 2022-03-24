package e2etests

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/syself/hetzner-cloud-controller-manager/internal/testsupport"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type TestCluster struct {
	KeepOnFailure bool
	useNetworks   bool
	setup         *hcloudK8sSetup
	k8sClient     *kubernetes.Clientset
	started       bool
	certificates  []*hcloud.Certificate

	mu sync.Mutex
}

func (tc *TestCluster) initialize() error {
	const op = "e2tests/TestCluster.initialize"

	if tc.started {
		return nil
	}

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
		k8sVersion = "k8s-1.18.9"
	}

	k8sVersionsDetails := strings.Split(k8sVersion, "-")
	if len(k8sVersionsDetails) != 2 {
		return fmt.Errorf("%s: invalid k8s version: %v should be format <distribution>-<version>", op, k8sVersion)
	}

	token := os.Getenv("HCLOUD_TOKEN")
	if len(token) != 64 {
		return fmt.Errorf("%s: No valid HCLOUD_TOKEN found", op)
	}
	tc.KeepOnFailure = os.Getenv("KEEP_SERVER_ON_FAILURE") == "yes"

	var additionalSSHKeys []*hcloud.SSHKey

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-ccm-testsuite", "1.0"),
	}
	hcloudClient := hcloud.NewClient(opts...)
	additionalSSHKeysIDOrName := os.Getenv("USE_SSH_KEYS")
	if additionalSSHKeysIDOrName != "" {
		idsOrNames := strings.Split(additionalSSHKeysIDOrName, ",")
		for _, idOrName := range idsOrNames {
			additionalSSHKey, _, err := hcloudClient.SSHKey.Get(context.Background(), idOrName)
			if err != nil {
				return fmt.Errorf("%s: %s", op, err)
			}
			additionalSSHKeys = append(additionalSSHKeys, additionalSSHKey)
		}
	}

	fmt.Printf("%s: Test against %s\n", op, k8sVersion)

	imageName := os.Getenv("CCM_IMAGE_NAME")
	buildImage := false
	if imageName == "" {
		imageName = fmt.Sprintf("hcloud-ccm:ci_%s", testIdentifier)
		buildImage = true
	}
	if buildImage {
		fmt.Printf("%s: Building ccm image\n", op)

		err := runCmd(
			"go",
			[]string{"build", "-o", "../hcloud-cloud-controller-manager", "../."},
			[]string{"CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64"},
		)
		if err != nil {
			return fmt.Errorf("%s: %v", op, err)
		}

		if err := runCmd("docker", []string{"build", "-t", imageName, "../"}, nil); err != nil {
			return fmt.Errorf("%s: %v", op, err)
		}
	}

	fmt.Printf("%s: Saving ccm image to disk\n", op)
	if err := runCmd("docker", []string{"save", "--output", "ci-hcloud-ccm.tar", imageName}, nil); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	tc.setup = &hcloudK8sSetup{
		Hcloud:          hcloudClient,
		K8sDistribution: K8sDistribution(k8sVersionsDetails[0]),
		K8sVersion:      k8sVersionsDetails[1],
		TestIdentifier:  testIdentifier,
		ImageName:       imageName,
		HcloudToken:     token,
		KeepOnFailure:   tc.KeepOnFailure,
		UseNetworks:     tc.useNetworks,
	}
	fmt.Printf("%s: Setting up test env\n", op)

	kubeconfigPath, err := tc.setup.PrepareTestEnv(context.Background(), additionalSSHKeys, tc.useNetworks)
	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("%s: clientcmd.BuildConfigFromFlags: %s", op, err)
	}

	tc.k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("%s: kubernetes.NewForConfig: %s", op, err)
	}

	tc.started = true
	return nil
}

func (tc *TestCluster) Start() error {
	const op = "e2etests/TestCluster.Start"

	tc.mu.Lock()
	defer tc.mu.Unlock()

	if err := tc.initialize(); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	if err := tc.ensureNodesReady(); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	if err := tc.ensurePodsReady(); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	return nil
}

func (tc *TestCluster) Stop(testFailed bool) error {
	const op = "e2etests/TestCluster.Stop"

	tc.mu.Lock()
	defer tc.mu.Unlock()

	if !tc.started {
		return nil
	}

	for _, c := range tc.certificates {
		if _, err := tc.setup.Hcloud.Certificate.Delete(context.Background(), c); err != nil {
			fmt.Printf("%s: delete certificate %d: %v", op, c.ID, err)
		}
	}

	if err := tc.setup.TearDown(testFailed); err != nil {
		fmt.Printf("%s: Tear Down: %s", op, err)
	}
	return nil
}

func (tc *TestCluster) ensureNodesReady() error {
	const op = "e2etests/ensureNodesReady"

	err := wait.Poll(1*time.Second, 5*time.Minute, func() (bool, error) {
		var totalNodes = len(tc.setup.WorkerNodes) + 1 // Number Worker Nodes + 1 Cluster Node
		var readyNodes int
		nodes, err := tc.k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					readyNodes++
				}
			}
		}
		pendingNodes := totalNodes - readyNodes
		fmt.Printf("Waiting for %d/%d nodes\n", pendingNodes, totalNodes)
		return pendingNodes == 0, err
	})

	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}
	return nil
}

func (tc *TestCluster) ensurePodsReady() error {
	const op = "e2etests/ensurePodsReady"

	err := wait.Poll(1*time.Second, 10*time.Minute, func() (bool, error) {
		pods, err := tc.k8sClient.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		totalPods := len(pods.Items)

		var readyPods int
		for _, pod := range pods.Items {
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					readyPods++
				}
			}
		}

		pendingPods := totalPods - readyPods
		fmt.Printf("Waiting for %d/%d pods\n", pendingPods, totalPods)
		return pendingPods == 0, err
	})

	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}
	return nil
}

func extractPodName(k8sName string) string {
	parts := strings.Split(k8sName, "-")
	return strings.Join(parts[0:len(parts)-2], "-")
}

// CreateTLSCertificate creates a TLS certificate used for testing and posts it
// to the Hetzner Cloud backend.
//
// The baseName of the certificate gets a random number suffix attached.
// baseName and suffix are separated by a single "-" character.
func (tc *TestCluster) CreateTLSCertificate(t *testing.T, baseName string) *hcloud.Certificate {
	const op = "e2etests/TestCluster.CreateTLSCertificate"

	rndInt := rng.Int()
	name := fmt.Sprintf("%s-%d", baseName, rndInt)

	p := testsupport.NewTLSPair(t, fmt.Sprintf("www.example%d.com", rndInt))
	opts := hcloud.CertificateCreateOpts{
		Name:        name,
		Certificate: p.Cert,
		PrivateKey:  p.Key,
	}
	cert, _, err := tc.setup.Hcloud.Certificate.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("%s: %s: %v", op, name, err)
	}
	if cert == nil {
		t.Fatalf("%s: no certificate created", op)
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.certificates = append(tc.certificates, cert)

	return cert
}

type lbTestHelper struct {
	podName       string
	port          int
	K8sClient     *kubernetes.Clientset
	KeepOnFailure bool
	t             *testing.T
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
	port := l.port
	if port == 0 {
		port = 80
	}

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
					Port:       int32(port),
					TargetPort: intstr.FromInt(80),
					Name:       "http",
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
		return nil, fmt.Errorf("%s: test service (load balancer) did not come up after 5 minute: %s", op, err)
	}
	return lbSvc, nil
}

// TearDown deletes the created pod and service
func (l *lbTestHelper) TearDown() {
	const op = "lbTestHelper/TearDown"

	if l.KeepOnFailure && l.t.Failed() {
		return
	}

	svcName := fmt.Sprintf("svc-%s", l.podName)
	err := l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Delete(context.Background(), svcName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		l.t.Errorf("%s: deleting test svc failed: %s", op, err)
	}

	err = wait.Poll(1*time.Second, 3*time.Minute, func() (done bool, err error) {
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
		l.t.Errorf("%s: test service was not removed after 3 minutes: %s", op, err)
	}

	podName := fmt.Sprintf("pod-%s", l.podName)
	err = l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		l.t.Errorf("%s: deleting test pod failed: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 3*time.Minute, func() (done bool, err error) {
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
		l.t.Errorf("%s: test pod not removed after 3 minutes: %s", op, err)
	}
}

type nwTestHelper struct {
	podName    string
	K8sClient  *kubernetes.Clientset
	privateKey string
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

// WaitForHTTPAvailable tries to connect to the given IP via http
// It tries it for 2 minutes, if after two minutes the connection
// wasn't successful and it wasn't a HTTP 200 response it will fail
func WaitForHTTPAvailable(t *testing.T, ingressIP string, useHTTPS bool) {
	const op = "e2etests/WaitForHTTPAvailable"

	client := &http.Client{
		Timeout: 1 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // nolint
			},
		},
	}
	proto := "http"
	if useHTTPS {
		proto = "https"
	}

	err := wait.Poll(1*time.Second, 2*time.Minute, func() (bool, error) {
		resp, err := client.Get(fmt.Sprintf("%s://%s", proto, ingressIP))
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
		t.Errorf("%s: not available via client.Get: %s", op, err)
	}
}

// WaitForHTTPOnServer tries to connect to the given IP using curl.
//
// It tries it for 2 minutes, if after two minutes the connection wasn't
// successful or it was not a HTTP 200 response it will fail
func WaitForHTTPOnServer(t *testing.T, srv *hcloud.Server, privateKey, tgtIP string, useHTTPS bool) {
	const op = "e2etests/WaitForHTTPOnServer"

	proto := "http"
	if useHTTPS {
		proto = "https"
	}
	cmd := fmt.Sprintf("curl -k %s://%s", proto, tgtIP)
	if net.ParseIP(tgtIP).To4() == nil {
		// Assume its a IPv6 address
		cmd = fmt.Sprintf("curl -6 -kg %s://[%s]", proto, tgtIP)
	}

	err := wait.Poll(1*time.Second, 2*time.Minute, func() (bool, error) {
		if err := RunCommandOnServer(privateKey, srv, cmd); err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Errorf("%s: not available via %q: %s", op, cmd, err)
	}
}
