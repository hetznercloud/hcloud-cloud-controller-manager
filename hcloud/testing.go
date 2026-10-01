package hcloud

import (
	"context"
	"maps"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/lbspec"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type LoadBalancerTestCase struct {
	Name string

	// Defined in test case as needed
	ClusterName              string
	NetworkID                int
	ServiceUID               string
	ServiceAnnotations       map[string]string
	UsePrivateIngressDefault *bool
	UseIPv6Default           *bool
	Nodes                    []*corev1.Node
	LB                       *hcloud.LoadBalancer
	LBCreateResult           *hcloud.LoadBalancerCreateResult
	Mock                     func(t *testing.T, tt *LoadBalancerTestCase)

	// Defines the actual test
	Perform func(t *testing.T, tt *LoadBalancerTestCase)

	Ctx context.Context // Set to context.Background by run if not defined in test

	// Set by run
	LBOps         *hcops.MockLoadBalancerOps
	LBClient      *mocks.LoadBalancerClient
	LoadBalancers *loadBalancers
	Service       *corev1.Service
	Spec          lbspec.Spec
}

func (tt *LoadBalancerTestCase) run(t *testing.T) {
	t.Helper()

	if tt.UsePrivateIngressDefault == nil {
		tt.UsePrivateIngressDefault = new(true)
	}

	if tt.UseIPv6Default == nil {
		tt.UseIPv6Default = new(true)
	}

	tt.LBOps = &hcops.MockLoadBalancerOps{}
	tt.LBOps.Test(t)

	tt.LBClient = &mocks.LoadBalancerClient{}
	tt.LBClient.Test(t)

	if tt.ClusterName == "" {
		tt.ClusterName = "test-cluster"
	}
	tt.Service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:         types.UID(tt.ServiceUID),
			Annotations: map[string]string{},
		},
	}
	maps.Copy(tt.Service.Annotations, tt.ServiceAnnotations)
	if tt.Ctx == nil {
		tt.Ctx = context.Background()
	}

	lbConfig := &config.LoadBalancerConfiguration{
		PrivateIngressEnabled: *tt.UsePrivateIngressDefault,
		IPv6Enabled:           *tt.UseIPv6Default,
	}

	// The error is ignored on purpose: a test case may set an invalid
	// annotation to check how the cloud provider reports it.
	tt.Spec, _ = lbspec.Resolve(&MockEventRecorder{}, tt.Service, *lbConfig)

	if tt.Mock != nil {
		tt.Mock(t, tt)
	}

	tt.LoadBalancers = newLoadBalancers(tt.LBOps, lbConfig, &MockEventRecorder{})
	tt.Perform(t, tt)

	tt.LBOps.AssertExpectations(t)
	tt.LBClient.AssertExpectations(t)
}

func RunLoadBalancerTests(t *testing.T, tests []LoadBalancerTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) { tt.run(t) })
	}
}
