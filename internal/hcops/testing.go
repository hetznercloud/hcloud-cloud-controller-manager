package hcops

import (
	"context"
	"math/rand"
	"net"
	"testing"

	hrobotmodels "github.com/syself/hrobot-go/models"
	"k8s.io/client-go/tools/record"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/mocks"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

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
		CertOps:       &CertificateOps{CertClient: fx.CertClient},
		ActionClient:  fx.ActionClient,
		NetworkClient: fx.NetworkClient,
		RobotClient:   fx.RobotClient,
		Recorder:      &record.FakeRecorder{},
	}

	return fx
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

func (fx *LoadBalancerOpsFixture) MockWatchProgress(a *hcloud.Action, err error) {
	fx.ActionClient.MockWatchProgress(fx.Ctx, a, err)
}

func (fx *LoadBalancerOpsFixture) AssertExpectations() {
	fx.ActionClient.AssertExpectations(fx.T)
	fx.LBClient.AssertExpectations(fx.T)
	fx.CertClient.AssertExpectations(fx.T)
	fx.NetworkClient.AssertExpectations(fx.T)
}
