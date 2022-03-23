package hcops

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/mock"
	"github.com/syself/hetzner-cloud-controller-manager/internal/mocks"
	corev1 "k8s.io/api/core/v1"
)

type MockLoadBalancerOps struct {
	mock.Mock
}

func (m *MockLoadBalancerOps) GetByName(ctx context.Context, name string) (*hcloud.LoadBalancer, error) {
	args := m.Called(ctx, name)
	return mocks.GetLoadBalancerPtr(args, 0), args.Error(1)
}

func (m *MockLoadBalancerOps) GetByID(ctx context.Context, id int) (*hcloud.LoadBalancer, error) {
	args := m.Called(ctx, id)
	return mocks.GetLoadBalancerPtr(args, 0), args.Error(1)
}

func (m *MockLoadBalancerOps) Create(
	ctx context.Context, lbName string, service *corev1.Service,
) (*hcloud.LoadBalancer, error) {
	args := m.Called(ctx, lbName, service)
	return mocks.GetLoadBalancerPtr(args, 0), args.Error(1)
}

func (m *MockLoadBalancerOps) Delete(ctx context.Context, lb *hcloud.LoadBalancer) error {
	args := m.Called(ctx, lb)
	return args.Error(0)
}

func (m *MockLoadBalancerOps) ReconcileHCLB(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service,
) (bool, error) {
	args := m.Called(ctx, lb, svc)
	return args.Bool(0), args.Error(1)
}

func (m *MockLoadBalancerOps) ReconcileHCLBTargets(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service, nodes []*corev1.Node,
) (bool, error) {
	args := m.Called(ctx, lb, svc, nodes)
	return args.Bool(0), args.Error(1)
}

func (m *MockLoadBalancerOps) ReconcileHCLBServices(
	ctx context.Context, lb *hcloud.LoadBalancer, svc *corev1.Service,
) (bool, error) {
	args := m.Called(ctx, lb, svc)
	return args.Bool(0), args.Error(1)
}

func (m *MockLoadBalancerOps) GetByK8SServiceUID(ctx context.Context, svc *corev1.Service) (*hcloud.LoadBalancer, error) {
	args := m.Called(ctx, svc)
	return mocks.GetLoadBalancerPtr(args, 0), args.Error(1)
}
