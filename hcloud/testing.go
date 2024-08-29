package hcloud

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type LoadBalancerTestCase struct {
	Name string

	// Defined in test case as needed
	ClusterName                  string
	NetworkID                    int
	ServiceUID                   string
	ServiceAnnotations           map[annotation.Name]interface{}
	DisablePrivateIngressDefault bool
	DisableIPv6Default           bool
	Nodes                        []*corev1.Node
	LB                           *hcloud.LoadBalancer
	LBCreateResult               *hcloud.LoadBalancerCreateResult
	Mock                         func(t *testing.T, tt *LoadBalancerTestCase)

	// Defines the actual test
	Perform func(t *testing.T, tt *LoadBalancerTestCase)

	Ctx context.Context // Set to context.Background by run if not defined in test

	// Set by run
	LBOps         *hcops.MockLoadBalancerOps
	LBClient      *mocks.LoadBalancerClient
	LoadBalancers *loadBalancers
	Service       *corev1.Service
}

func (tt *LoadBalancerTestCase) run(t *testing.T) {
	t.Helper()

	tt.LBOps = &hcops.MockLoadBalancerOps{}
	tt.LBOps.Test(t)

	tt.LBClient = &mocks.LoadBalancerClient{}
	tt.LBClient.Test(t)

	if tt.ClusterName == "" {
		tt.ClusterName = "test-cluster"
	}
	tt.Service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{UID: types.UID(tt.ServiceUID)},
	}
	for k, v := range tt.ServiceAnnotations {
		if err := k.AnnotateService(tt.Service, v); err != nil {
			t.Fatal(err)
		}
	}
	if tt.Ctx == nil {
		tt.Ctx = context.Background()
	}

	if tt.Mock != nil {
		tt.Mock(t, tt)
	}

	tt.LoadBalancers = newLoadBalancers(tt.LBOps, tt.DisablePrivateIngressDefault, tt.DisableIPv6Default, "")
	tt.Perform(t, tt)

	tt.LBOps.AssertExpectations(t)
	tt.LBClient.AssertExpectations(t)
}

func RunLoadBalancerTests(t *testing.T, tests []LoadBalancerTestCase) {
	for _, tt := range tests {
		tt.run(t)
	}
}
