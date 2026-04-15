//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPodIsPresent(t *testing.T) {
	t.Parallel()

	t.Run("hcloud-cloud-controller-manager pod is present in kube-system", func(t *testing.T) {
		pods, err := testCluster.k8sClient.CoreV1().Pods("kube-system").List(t.Context(), metav1.ListOptions{})
		require.NoError(t, err)

		found := false
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "hcloud-cloud-controller-manager") {
				found = true
				break
			}
		}
		assert.True(t, found, "kube-system does not contain a pod named hcloud-cloud-controller-manager")
	})

	t.Run("pod with app=hcloud-cloud-controller-manager is present in kube-system", func(t *testing.T) {
		pods, err := testCluster.k8sClient.CoreV1().Pods("kube-system").
			List(t.Context(), metav1.ListOptions{
				LabelSelector: "app.kubernetes.io/name=hcloud-cloud-controller-manager",
			})
		require.NoError(t, err)

		require.NotEmpty(t, pods.Items, "kube-system does not contain a pod with label app=hcloud-cloud-controller-manager")
	})
}
