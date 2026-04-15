//go:build e2e

package e2e

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hrobot "github.com/syself/hrobot-go"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/utils"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type TestCluster struct {
	hcloud    *hcloud.Client
	hrobot    hrobot.RobotClient
	k8sClient *kubernetes.Clientset

	// envName is used as the name prefix in all resources
	envName    string
	certDomain string

	certificates  *utils.SyncSet[int64]
	loadBalancers *utils.SyncSet[int64]
}

func (tc *TestCluster) Start() error {
	// Hetzner Cloud Client
	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		return fmt.Errorf("no valid HCLOUD_TOKEN found")
	}

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-ccm-testsuite", "1.0"),
		hcloud.WithPollOpts(hcloud.PollOpts{
			BackoffFunc: hcloud.ExponentialBackoffWithOpts(hcloud.ExponentialBackoffOpts{
				Base:       time.Second,
				Multiplier: 2,
				Cap:        10 * time.Second,
			}),
		}),
	}
	tc.hcloud = hcloud.NewClient(opts...)

	// Hetzner Robot Client
	if enabled := os.Getenv("ROBOT_ENABLED"); enabled == "true" {
		robotUser := os.Getenv("ROBOT_USER")
		robotPassword := os.Getenv("ROBOT_PASSWORD")
		tc.hrobot = hrobot.NewBasicAuthClient(robotUser, robotPassword)
	}

	tc.envName = os.Getenv("ENV_NAME")

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	clientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("kubeConfig.ClientConfig: %s", err)
	}

	tc.k8sClient, err = kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return fmt.Errorf("kubernetes.NewForConfig: %s", err)
	}

	// Tests using this value should skip if empty
	// The domain specified here must be available in Hetzner DNS of the account running the tests.
	tc.certDomain = os.Getenv("CERT_DOMAIN")

	tc.certificates = utils.NewSyncSet[int64]()
	tc.loadBalancers = utils.NewSyncSet[int64]()

	return nil
}

func (tc *TestCluster) Stop() error {
	errs := make([]error, 0, tc.loadBalancers.Size()+tc.certificates.Size())
	ctx := context.Background()

	for _, item := range tc.loadBalancers.All() {
		fmt.Printf("deleting load balancer %d\n", item)
		if _, err := tc.hcloud.LoadBalancer.Delete(ctx, &hcloud.LoadBalancer{ID: item}); err != nil {
			errs = append(errs, fmt.Errorf("delete load balancer %d failed: %w", item, err))
		}
	}

	for _, item := range tc.certificates.All() {
		fmt.Printf("deleting certificate %d\n", item)
		if _, err := tc.hcloud.Certificate.Delete(ctx, &hcloud.Certificate{ID: item}); err != nil {
			errs = append(errs, fmt.Errorf("delete certificate %d failed: %w", item, err))
		}
	}

	return errors.Join(errs...)
}

// CreateTLSCertificate creates a TLS certificate used for testing and posts it
// to the Hetzner Cloud backend.
//
// The baseName of the certificate gets a random number suffix attached.
// baseName and suffix are separated by a single "-" character.
func (tc *TestCluster) CreateTLSCertificate(t *testing.T, baseName string) (*hcloud.Certificate, error) {
	rndInt := rng.Int()
	name := fmt.Sprintf("%s-%d", baseName, rndInt)

	p := testsupport.NewTLSPair(t, fmt.Sprintf("www.example%d.com", rndInt))
	opts := hcloud.CertificateCreateOpts{
		Name:        name,
		Certificate: p.Cert,
		PrivateKey:  p.Key,
	}
	cert, _, err := tc.hcloud.Certificate.Create(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if cert == nil {
		return nil, fmt.Errorf("no certificate created")
	}

	tc.certificates.Add(cert.ID)

	return cert, nil
}

// NetworkName returns the network name.
func (tc *TestCluster) NetworkName() string {
	return tc.envName
}

// ControlNodeName returns the control node name.
func (tc *TestCluster) ControlNodeName() string {
	return fmt.Sprintf("%s-control", tc.envName)
}

// WorkerNodeName returns the worker node name, zero indexed.
func (tc *TestCluster) WorkerNodeName(index int) string {
	return fmt.Sprintf("%s-worker-%d", tc.envName, index)
}

type lbTestHelper struct {
	t *testing.T

	namespace string
	podName   string
	port      int
}

// DeployTestPod deploys a basic nginx pod within the k8s cluster
// and waits until it is "ready".
func (l *lbTestHelper) DeployTestPod() (*corev1.Pod, error) {
	l.t.Helper()

	ctx := context.Background()

	if l.namespace == "" {
		l.namespace = "hccm-test-" + strconv.Itoa(rand.Int())
	}
	_, err := testCluster.k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: l.namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("error deploying test pod: %w", err)
	}

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

	pod, err := testCluster.k8sClient.CoreV1().Pods(l.namespace).Create(ctx, &testPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not create test pod: %w", err)
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		p, err := testCluster.k8sClient.CoreV1().Pods(l.namespace).Get(ctx, podName, metav1.GetOptions{})
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
		return nil, fmt.Errorf("pod %s did not come up after 1 minute: %w", podName, err)
	}

	return pod, nil
}

// ServiceDefinition returns a service definition for a Hetzner Cloud Load Balancer (k8s service).
func (l *lbTestHelper) ServiceDefinition(pod *corev1.Pod, annotations map[string]string) *corev1.Service {
	l.t.Helper()

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
// and waits until it is "ready".
func (l *lbTestHelper) CreateService(lbSvc *corev1.Service) (*corev1.Service, error) {
	l.t.Helper()

	lbSvc, err := testCluster.k8sClient.CoreV1().Services(l.namespace).Create(l.t.Context(), lbSvc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not create service: %w", err)
	}

	ctx, cancel := context.WithTimeout(l.t.Context(), 4*time.Minute)
	defer cancel()

	backoffFunc := hcloud.ExponentialBackoff(2.0, time.Second)
	retries := 0
	for {
		svc, err := testCluster.k8sClient.CoreV1().Services(l.namespace).Get(ctx, lbSvc.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error fetching load balancer service: %w", err)
		}

		ingressIPs := svc.Status.LoadBalancer.Ingress
		if len(ingressIPs) > 0 {
			lbSvc = svc
			return lbSvc, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for load balancer service to receive ingress IPs")
		case <-time.After(backoffFunc(retries)):
			retries++
			continue
		}
	}
}

// TearDown deletes the created pod and service.
func (l *lbTestHelper) TearDown() {
	l.t.Helper()

	// No namespace was created yet (e.g. DeployTestPod never ran because a
	// prior step failed); nothing to clean up.
	if l.namespace == "" {
		return
	}

	err := wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
		err := testCluster.k8sClient.CoreV1().Namespaces().Delete(ctx, l.namespace, metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return false, err
		}
		return k8serrors.IsNotFound(err), nil
	})
	require.NoError(l.t, err)
}

// WaitForHTTPAvailable tries to connect to the given IP via HTTP or HTTPS
// (controlled by useHTTPS). It uses exponential backoff starting at 1s and
// capping at 30s, waiting up to 6 minutes for a successful HTTP 200 response.
// Each individual request has a 5s timeout.
func (l *lbTestHelper) WaitForHTTPAvailable(ingressIP string, useHTTPS bool) error {
	l.t.Helper()

	client := &http.Client{
		Timeout: 5 * time.Second,
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

	ctx, cancel := context.WithTimeout(l.t.Context(), 6*time.Minute)
	defer cancel()

	backoffFunc := hcloud.ExponentialBackoffWithOpts(hcloud.ExponentialBackoffOpts{
		Base:       time.Second,
		Multiplier: 2,
		Cap:        30 * time.Second,
	})
	retries := 0
	for {
		resp, err := client.Get(fmt.Sprintf("%s://%s", proto, ingressIP))
		if err != nil {
			l.t.Logf("request to %s failed, keep waiting: %v", ingressIP, err)
		} else {
			resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusOK:
				return nil
			case http.StatusServiceUnavailable:
				l.t.Log("service still unavailable, keep waiting")
			default:
				return fmt.Errorf("got unexpected HTTP status %d", resp.StatusCode)
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out after 6m waiting for %s to be available", ingressIP)
		case <-time.After(backoffFunc(retries)):
			retries++
			continue
		}
	}
}
