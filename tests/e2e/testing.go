package e2e

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
	"github.com/hetznercloud/hcloud-go/hcloud"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var rng *rand.Rand
var scopeButcher = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type TestCluster struct {
	hcloud       *hcloud.Client
	k8sClient    *kubernetes.Clientset
	certificates []*hcloud.Certificate
	scope        string
}

func (tc *TestCluster) Start() error {
	tc.scope = os.Getenv("SCOPE")
	if tc.scope == "" {
		tc.scope = "dev"
	}
	tc.scope = scopeButcher.ReplaceAllString(tc.scope, "-")

	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		buf, err := os.ReadFile(fmt.Sprintf("../../hack/.token-%s", tc.scope))
		if err != nil {
			return err
		}
		token = string(buf)
	}

	if token == "" {
		return fmt.Errorf("no valid HCLOUD_TOKEN found")
	}

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-ccm-testsuite", "1.0"),
	}
	hcloudClient := hcloud.NewClient(opts...)
	tc.hcloud = hcloudClient

	err := os.Setenv("KUBECONFIG", "../../hack/.kubeconfig-"+tc.scope)
	if err != nil {
		return err
	}

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

	return nil
}

func (tc *TestCluster) Stop() error {
	for _, c := range tc.certificates {
		if _, err := tc.hcloud.Certificate.Delete(context.Background(), c); err != nil {
			fmt.Printf("delete certificate %d failed: %v", c.ID, err)
		}
	}

	return nil
}

// CreateTLSCertificate creates a TLS certificate used for testing and posts it
// to the Hetzner Cloud backend.
//
// The baseName of the certificate gets a random number suffix attached.
// baseName and suffix are separated by a single "-" character.
func (tc *TestCluster) CreateTLSCertificate(t *testing.T, baseName string) *hcloud.Certificate {

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
		t.Fatalf("%s: %v", name, err)
	}
	if cert == nil {
		t.Fatalf("no certificate created")
	}

	tc.certificates = append(tc.certificates, cert)

	return cert
}

type lbTestHelper struct {
	podName   string
	port      int
	K8sClient *kubernetes.Clientset
	t         *testing.T
	namespace string
}

// DeployTestPod deploys a basic nginx pod within the k8s cluster
// and waits until it is "ready".
func (l *lbTestHelper) DeployTestPod() *corev1.Pod {
	if l.namespace == "" {
		l.namespace = "hccm-test-" + strconv.Itoa(rand.Int())
	}
	_, err := l.K8sClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: l.namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		panic(err)
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

	pod, err := l.K8sClient.CoreV1().Pods(l.namespace).Create(context.Background(), &testPod, metav1.CreateOptions{})
	if err != nil {
		l.t.Fatalf("could not create test pod: %s", err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		p, err := l.K8sClient.CoreV1().Pods(l.namespace).Get(context.Background(), podName, metav1.GetOptions{})
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
		l.t.Fatalf("pod %s did not come up after 1 minute: %s", podName, err)
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
// and waits until it is "ready".
func (l *lbTestHelper) CreateService(lbSvc *corev1.Service) (*corev1.Service, error) {

	// Default is 15s interval, 10s timeout, 3 retries => 45 seconds until up
	// With these changes it should be 1 seconds until up
	// lbSvc.Annotations[string(annotation.LBSvcHealthCheckInterval)] = "1s"
	// lbSvc.Annotations[string(annotation.LBSvcHealthCheckTimeout)] = "2s"
	// lbSvc.Annotations[string(annotation.LBSvcHealthCheckRetries)] = "1"
	// lbSvc.Annotations[string(annotation.LBSvcHealthCheckProtocol)] = "tcp"

	_, err := l.K8sClient.CoreV1().Services(l.namespace).Create(context.Background(), lbSvc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not create service: %s", err)
	}

	err = wait.Poll(1*time.Second, 5*time.Minute, func() (done bool, err error) {
		svc, err := l.K8sClient.CoreV1().Services(l.namespace).Get(context.Background(), lbSvc.Name, metav1.GetOptions{})
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
		return nil, fmt.Errorf("test service (load balancer) did not come up after 5 minute: %s", err)
	}
	return lbSvc, nil
}

// TearDown deletes the created pod and service.
func (l *lbTestHelper) TearDown() {
	svcName := fmt.Sprintf("svc-%s", l.podName)
	err := l.K8sClient.CoreV1().Services(l.namespace).Delete(context.Background(), svcName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		l.t.Errorf("deleting test svc failed: %s", err)
	}

	err = wait.Poll(1*time.Second, 3*time.Minute, func() (done bool, err error) {
		err = l.K8sClient.CoreV1().Namespaces().Delete(context.Background(), l.namespace, metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		panic(err)
	}
}

// WaitForHTTPAvailable tries to connect to the given IP via http
// It tries it for 2 minutes, if after two minutes the connection
// wasn't successful and it wasn't a HTTP 200 response it will fail
func WaitForHTTPAvailable(t *testing.T, ingressIP string, useHTTPS bool) {

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
		switch resp.StatusCode {
		case http.StatusOK:
			// Success
			return true, nil
		case http.StatusServiceUnavailable:
			// Health checks are still evaluating
			return false, nil
		default:
			return false, fmt.Errorf("got HTTP Code %d instead of 200", resp.StatusCode)
		}
	})
	if err != nil {
		t.Errorf("not available via client.Get: %s", err)
	}
}
