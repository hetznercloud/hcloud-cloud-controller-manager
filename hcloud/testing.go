package hcloud

import (
	"context"
	"os"
	"testing"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/syself/hetzner-cloud-controller-manager/internal/annotation"
	"github.com/syself/hetzner-cloud-controller-manager/internal/hcops"
	"github.com/syself/hetzner-cloud-controller-manager/internal/mocks"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Setenv prepares the environment for testing the
// hcloud-cloud-controller-manager.
func Setenv(t *testing.T, args ...string) func() {
	if len(args)%2 != 0 {
		t.Fatal("Sentenv: uneven number of args")
	}

	newVars := make([]string, 0, len(args)/2)
	oldEnv := make(map[string]string, len(newVars))

	for i := 0; i < len(args); i += 2 {
		k, v := args[i], args[i+1]
		newVars = append(newVars, k)

		if old, ok := os.LookupEnv(k); ok {
			oldEnv[k] = old
		}
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Setenv failed: %v", err)
		}
	}

	return func() {
		for _, k := range newVars {
			v, ok := oldEnv[k]
			if !ok {
				if err := os.Unsetenv(k); err != nil {
					t.Errorf("Unsetenv failed: %v", err)
				}
				continue
			}
			if err := os.Setenv(k, v); err != nil {
				t.Errorf("Setenv failed: %v", err)
			}
		}
	}
}

// SkipEnv skips t if any of the passed vars is not set as an environment
// variable.
//
// SkipEnv uses os.LookupEnv. The empty string is therefore a valid value.
func SkipEnv(t *testing.T, vars ...string) {
	for _, v := range vars {
		if _, ok := os.LookupEnv(v); !ok {
			t.Skipf("Environment variable not set: %s", v)
			return
		}
	}
}

type LoadBalancerTestCase struct {
	Name string

	// Defined in test case as needed
	ClusterName                  string
	NetworkID                    int
	ServiceUID                   string
	ServiceAnnotations           map[annotation.Name]interface{}
	DisablePrivateIngressDefault bool
	DisableIPv6Default           bool
	Nodes                        []*v1.Node
	LB                           *hcloud.LoadBalancer
	LBCreateResult               *hcloud.LoadBalancerCreateResult
	Mock                         func(t *testing.T, tt *LoadBalancerTestCase)

	// Defines the actual test
	Perform func(t *testing.T, tt *LoadBalancerTestCase)

	Ctx context.Context // Set to context.Background by run if not defined in test

	// Set by run
	LBOps         *hcops.MockLoadBalancerOps
	LBClient      *mocks.LoadBalancerClient
	ActionClient  *mocks.ActionClient
	LoadBalancers *loadBalancers
	Service       *v1.Service
}

func (tt *LoadBalancerTestCase) run(t *testing.T) {
	t.Helper()

	tt.LBOps = &hcops.MockLoadBalancerOps{}
	tt.LBOps.Test(t)

	tt.ActionClient = &mocks.ActionClient{}
	tt.ActionClient.Test(t)

	tt.LBClient = &mocks.LoadBalancerClient{}
	tt.LBClient.Test(t)

	if tt.ClusterName == "" {
		tt.ClusterName = "test-cluster"
	}
	tt.Service = &v1.Service{
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

	tt.LoadBalancers = newLoadBalancers(tt.LBOps, tt.ActionClient, tt.DisablePrivateIngressDefault, tt.DisableIPv6Default)
	tt.Perform(t, tt)

	tt.LBOps.AssertExpectations(t)
	tt.LBClient.AssertExpectations(t)
	tt.ActionClient.AssertExpectations(t)
}

func RunLoadBalancerTests(t *testing.T, tests []LoadBalancerTestCase) {
	for _, tt := range tests {
		tt.run(t)
	}
}
