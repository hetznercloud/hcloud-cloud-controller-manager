package mocks

import (
	"context"
	"net"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/mock"
)

type LoadBalancerClient struct {
	mock.Mock
}

func (m *LoadBalancerClient) GetByID(ctx context.Context, id int) (*hcloud.LoadBalancer, *hcloud.Response, error) {
	args := m.Called(ctx, id)
	return GetLoadBalancerPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) GetByName(
	ctx context.Context, name string,
) (*hcloud.LoadBalancer, *hcloud.Response, error) {
	args := m.Called(ctx, name)
	return GetLoadBalancerPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) Create(
	ctx context.Context, opts hcloud.LoadBalancerCreateOpts,
) (hcloud.LoadBalancerCreateResult, *hcloud.Response, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(hcloud.LoadBalancerCreateResult), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) Update(
	ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerUpdateOpts,
) (*hcloud.LoadBalancer, *hcloud.Response, error) {
	args := m.Called(ctx, lb, opts)
	return GetLoadBalancerPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) Delete(ctx context.Context, lb *hcloud.LoadBalancer) (*hcloud.Response, error) {
	args := m.Called(ctx, lb)
	return getResponsePtr(args, 0), args.Error(1)
}

func (m *LoadBalancerClient) AddService(
	ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServiceOpts,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, opts)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) DeleteService(
	ctx context.Context, lb *hcloud.LoadBalancer, listenPort int,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, listenPort)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) ChangeAlgorithm(
	ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerChangeAlgorithmOpts,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, opts)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) ChangeType(
	ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerChangeTypeOpts,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, opts)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) AddServerTarget(
	ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServerTargetOpts,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, opts)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) RemoveServerTarget(ctx context.Context, lb *hcloud.LoadBalancer, server *hcloud.Server) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, server)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) AddIPTarget(
	ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddIPTargetOpts,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, opts)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) RemoveIPTarget(ctx context.Context, lb *hcloud.LoadBalancer, ip net.IP) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, ip)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) UpdateService(
	ctx context.Context, lb *hcloud.LoadBalancer, listenPort int, opts hcloud.LoadBalancerUpdateServiceOpts,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, listenPort, opts)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) AttachToNetwork(
	ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAttachToNetworkOpts,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, opts)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) DetachFromNetwork(
	ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerDetachFromNetworkOpts,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb, opts)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) EnablePublicInterface(
	ctx context.Context, lb *hcloud.LoadBalancer,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) DisablePublicInterface(
	ctx context.Context, lb *hcloud.LoadBalancer,
) (*hcloud.Action, *hcloud.Response, error) {
	args := m.Called(ctx, lb)
	return getActionPtr(args, 0), getResponsePtr(args, 1), args.Error(2)
}

func (m *LoadBalancerClient) AllWithOpts(
	ctx context.Context, opts hcloud.LoadBalancerListOpts,
) ([]*hcloud.LoadBalancer, error) {
	args := m.Called(ctx, opts)
	return getLoadBalancerPtrS(args, 0), args.Error(1)
}
