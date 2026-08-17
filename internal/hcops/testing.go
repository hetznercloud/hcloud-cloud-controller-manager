package hcops

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	hrobotmodels "github.com/syself/hrobot-go/models"
	"k8s.io/client-go/tools/record"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/cache"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// LBTypesResponse is served by the fixtures Load Balancer type API. Tests that expect a
// Load Balancer type to be looked up have to use one of these types.
const LBTypesResponse = `{
	"load_balancer_types": [
		{"id": 1, "name": "lb11"},
		{"id": 2, "name": "lb21"},
		{"id": 3, "name": "lb31"}
	],
	"meta": {"pagination": {"page": 1, "per_page": 50, "previous_page": null, "next_page": null, "last_page": 1, "total_entries": 3}}
}`

type LoadBalancerOpsFixture struct {
	Name          string
	Ctx           context.Context
	LBClient      *mocks.LoadBalancerClient
	CertClient    *mocks.CertificateClient
	ActionClient  *mocks.ActionClient
	NetworkClient *mocks.NetworkClient
	RobotClient   *mocks.RobotClient

	LBOps *LoadBalancerOps

	T *testing.T
}

func NewLoadBalancerOpsFixture(t *testing.T) *LoadBalancerOpsFixture {
	fx := &LoadBalancerOpsFixture{
		Ctx:           context.Background(),
		ActionClient:  &mocks.ActionClient{},
		LBClient:      &mocks.LoadBalancerClient{},
		CertClient:    &mocks.CertificateClient{},
		NetworkClient: &mocks.NetworkClient{},
		RobotClient:   &mocks.RobotClient{},
		T:             t,
	}

	fx.ActionClient.Test(t)
	fx.LBClient.Test(t)
	fx.CertClient.Test(t)
	fx.NetworkClient.Test(t)
	fx.RobotClient.Test(t)

	fx.LBOps = &LoadBalancerOps{
		LBClient:      fx.LBClient,
		CertOps:       &CertificateOps{ActionClient: fx.ActionClient, CertClient: fx.CertClient},
		ActionClient:  fx.ActionClient,
		NetworkClient: fx.NetworkClient,
		RobotClient:   fx.RobotClient,
		LBTypeCache:   newLBTypeCacheFixture(t),
		Recorder:      &record.FakeRecorder{},
	}

	return fx
}

// newLBTypeCacheFixture returns a Load Balancer type cache backed by a test server that
// always serves [LBTypesResponse].
func newLBTypeCacheFixture(t *testing.T) *cache.Cache[hcloud.LoadBalancerType] {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(LBTypesResponse)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)

	client := hcloud.NewClient(hcloud.WithEndpoint(server.URL))
	return cache.NewLoadBalancerTypeCache(client, cache.ModeAll, time.Minute)
}

func (fx *LoadBalancerOpsFixture) MockGetByID(lb *hcloud.LoadBalancer, err error) {
	fx.LBClient.On("GetByID", fx.Ctx, lb.ID).Return(lb, nil, err)
}

func (fx *LoadBalancerOpsFixture) MockCreate(
	opts hcloud.LoadBalancerCreateOpts, lb *hcloud.LoadBalancer, err error,
) *hcloud.Action {
	action := &hcloud.Action{ID: rand.Int63()}
	createResult := hcloud.LoadBalancerCreateResult{Action: action, LoadBalancer: lb}
	fx.LBClient.On("Create", fx.Ctx, opts).Return(createResult, nil, err)
	return action
}

func (fx *LoadBalancerOpsFixture) MockAddService(
	opts hcloud.LoadBalancerAddServiceOpts, lb *hcloud.LoadBalancer, err error,
) *hcloud.Action {
	action := &hcloud.Action{ID: rand.Int63()}
	fx.LBClient.On("AddService", fx.Ctx, lb, opts).Return(action, nil, err)
	return action
}

func (fx *LoadBalancerOpsFixture) MockUpdateService(
	opts hcloud.LoadBalancerUpdateServiceOpts, lb *hcloud.LoadBalancer, listenPort int, err error,
) *hcloud.Action {
	action := &hcloud.Action{ID: rand.Int63()}
	fx.LBClient.On("UpdateService", fx.Ctx, lb, listenPort, opts).Return(action, nil, err)
	return action
}

func (fx *LoadBalancerOpsFixture) MockDeleteService(lb *hcloud.LoadBalancer, port int, err error) *hcloud.Action {
	action := &hcloud.Action{ID: rand.Int63()}
	fx.LBClient.On("DeleteService", fx.Ctx, lb, port).Return(action, nil, err)
	return action
}

func (fx *LoadBalancerOpsFixture) MockAddServerTarget(
	lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServerTargetOpts, err error,
) *hcloud.Action {
	action := &hcloud.Action{ID: rand.Int63()}
	fx.LBClient.On("AddServerTarget", fx.Ctx, lb, opts).Return(action, nil, err)
	return action
}

func (fx *LoadBalancerOpsFixture) MockRemoveServerTarget(
	lb *hcloud.LoadBalancer, s *hcloud.Server, err error,
) *hcloud.Action {
	action := &hcloud.Action{ID: rand.Int63()}
	fx.LBClient.On("RemoveServerTarget", fx.Ctx, lb, s).Return(action, nil, err)
	return action
}

func (fx *LoadBalancerOpsFixture) MockAddIPTarget(
	lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddIPTargetOpts, err error,
) *hcloud.Action {
	action := &hcloud.Action{ID: rand.Int63()}
	fx.LBClient.On("AddIPTarget", fx.Ctx, lb, opts).Return(action, nil, err)
	return action
}

func (fx *LoadBalancerOpsFixture) MockRemoveIPTarget(
	lb *hcloud.LoadBalancer, ip net.IP, err error,
) *hcloud.Action {
	action := &hcloud.Action{ID: rand.Int63()}
	fx.LBClient.On("RemoveIPTarget", fx.Ctx, lb, ip).Return(action, nil, err)
	return action
}

func (fx *LoadBalancerOpsFixture) MockListRobotServers(
	serverList []hrobotmodels.Server, err error,
) {
	fx.RobotClient.On("ServerGetList").Return(serverList, err)
}

func (fx *LoadBalancerOpsFixture) AssertExpectations() {
	fx.ActionClient.AssertExpectations(fx.T)
	fx.LBClient.AssertExpectations(fx.T)
	fx.CertClient.AssertExpectations(fx.T)
	fx.NetworkClient.AssertExpectations(fx.T)
}
