package e2etests

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type lbTests struct {
	TestName  string
	K8sClient *kubernetes.Clientset
}

// DeployTestPod deploys a basic nginx pod within the k8s cluster
// and waits until it is "ready"
func (l *lbTests) DeployTestPod(ctx context.Context) (*corev1.Pod, error) {
	const op = "lbTests/DeployTestPod"
	podName := fmt.Sprintf("pod-%s", l.TestName)
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

	pod, err := l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Create(ctx, &testPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: could not create test pod: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		p, err := l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(ctx, podName, metav1.GetOptions{})
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
		return nil, fmt.Errorf("%s: pod %s did not come up after 1 minute: %s", op, podName, err)
	}
	return pod, nil
}

// ServiceDefinition returns a service definition for a Hetzner Cloud Load Balancer (k8s service)
func (l *lbTests) ServiceDefinition(pod *corev1.Pod, annotations map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("svc-%s", l.TestName),
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
func (l *lbTests) CreateService(ctx context.Context, lbSvc *corev1.Service) (*corev1.Service, error) {
	const op = "lbTests/CreateService"
	_, err := l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Create(ctx, lbSvc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: could not create service: %s", op, err)
	}

	err = wait.Poll(1*time.Second, 5*time.Minute, func() (done bool, err error) {
		svc, err := l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Get(ctx, lbSvc.Name, metav1.GetOptions{})
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
func (l *lbTests) TearDown(ctx context.Context) error {
	const op = "lbTests/TearDown"
	svcName := fmt.Sprintf("svc-%s", l.TestName)
	err := l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Delete(ctx, svcName, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("%s: deleting test svc failed: %s", op, err)
	}

	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		_, err = l.K8sClient.CoreV1().Services(corev1.NamespaceDefault).Get(ctx, svcName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("%s: test service was not removed after 1 minute: %s", op, err)
	}

	podName := fmt.Sprintf("pod-%s", l.TestName)
	err = l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("%s: deleting test pod failed: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		_, err = l.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("%s: test pod not removed after 1 minute: %s", op, err)
	}

	return nil
}

// WaitForHttpAvailable tries to connect to the given IP via http
// It tries it for 2 minutes, if after two minutes the connection
// wasn't successful and it wasn't a HTTP 200 response it will fail
func (l *lbTests) WaitForHttpAvailable(ingressIP string) error {
	const op = "lbTests/WaitForHttpAvailable"
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	return wait.Poll(1*time.Second, 2*time.Minute, func() (bool, error) {
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
}

type nwTests struct {
	TestName   string
	K8sClient  *kubernetes.Clientset
	privateKey string
	server     *hcloud.Server
}

// DeployTestPod deploys a basic nginx pod within the k8s cluster
// and waits until it is "ready"
func (n *nwTests) DeployTestPod(ctx context.Context) (*corev1.Pod, error) {
	const op = "nwTests/DeployTestPod"
	podName := fmt.Sprintf("pod-%s", n.TestName)
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

	pod, err := n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Create(ctx, &testPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: could not create test pod: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		p, err := n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(ctx, podName, metav1.GetOptions{})
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
		return nil, fmt.Errorf("%s: pod %s did not come up after 1 minute: %s", op, podName, err)
	}
	pod, err = n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: could not create test pod: %s", op, err)
	}
	return pod, nil
}

// WaitForHttpAvailable tries to connect to the given IP via curl
// It tries it for 2 minutes, if after two minutes the connection
// wasn't successful and it wasn't a HTTP 200 response it will fail
func (n *nwTests) WaitForHttpAvailable(podIp string) error {
	const op = "nwTests/WaitForHttpAvailable"
	return wait.Poll(1*time.Second, 2*time.Minute, func() (bool, error) {
		err := RunCommandOnServer(n.privateKey, n.server, fmt.Sprintf("curl http://%s", podIp))
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

// TearDown deletes the created pod
func (n *nwTests) TearDown(ctx context.Context) error {
	const op = "nwTests/TearDown"
	podName := fmt.Sprintf("pod-%s", n.TestName)
	err := n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("%s: deleting test pod failed: %s", op, err)
	}
	err = wait.Poll(1*time.Second, 1*time.Minute, func() (done bool, err error) {
		_, err = n.K8sClient.CoreV1().Pods(corev1.NamespaceDefault).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("%s: test pod not removed after 1 minute: %s", op, err)
	}

	return nil
}
