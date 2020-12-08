package hcloud

import (
	"errors"
	"net"
	"testing"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestLoadBalancers_GetLoadBalancer(t *testing.T) {
	tests := []LoadBalancerTestCase{
		{
			Name:       "get load balancer without host name",
			ServiceUID: "1",
			LB: &hcloud.LoadBalancer{
				ID:   1,
				Name: "no-host-name",
				PublicNet: hcloud.LoadBalancerPublicNet{
					IPv4: hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("1.2.3.4")},
					// IPv6:    hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("fe80::1")},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				status, exists, err := tt.LoadBalancers.GetLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service)
				assert.NoError(t, err)
				assert.True(t, exists)

				if !assert.Len(t, status.Ingress, 1) {
					return
				}
				assert.Equal(t, tt.LB.PublicNet.IPv4.IP.String(), status.Ingress[0].IP)
			},
		},
		{
			Name:       "get load balancer with host name",
			ServiceUID: "2",
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(tt.LB, nil)
			},
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBHostname: "hostname",
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				status, exists, err := tt.LoadBalancers.GetLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service)
				assert.NoError(t, err)
				assert.True(t, exists)

				if !assert.Len(t, status.Ingress, 1) {
					return
				}
				assert.Equal(t, "hostname", status.Ingress[0].Hostname)
			},
		},
		{
			Name:       "load balancer not found",
			ServiceUID: "3",
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, hcops.ErrNotFound)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, exists, err := tt.LoadBalancers.GetLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service)
				assert.NoError(t, err)
				assert.False(t, exists)
			},
		},
		{
			Name:       "lookup failed",
			ServiceUID: "4",
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, errors.New("some error"))
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, _, err := tt.LoadBalancers.GetLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service)
				assert.EqualError(t, err, "hcloud/loadBalancers.GetLoadBalancer: some error")
			},
		},
	}

	RunLoadBalancerTests(t, tests)
}

func TestLoadBalancers_EnsureLoadBalancer_CreateLoadBalancer(t *testing.T) {
	tests := []LoadBalancerTestCase{
		{
			Name:       "check for existing Load Balancer fails",
			ServiceUID: "1",
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, errors.New("test error"))
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.EqualError(t, err, "hcloud/loadBalancers.EnsureLoadBalancer: test error")
			},
		},
		{
			Name:       "public network only",
			ServiceUID: "2",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "pub-net-only",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				Name:             "pub-net-only",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv4:    hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("1.2.3.4")},
					// IPv6:    hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("fe80::1")},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("GetByName", tt.Ctx, "pub-net-only").
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("Create", tt.Ctx, tt.LB.Name, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				expected := &v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{IP: tt.LB.PublicNet.IPv4.IP.String()},
						// {IP: tt.LB.PublicNet.IPv6.IP.String()},
					},
				}
				lbStat, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
				assert.Equal(t, expected, lbStat)
			},
		},
		{
			Name:       "attach Load Balancer to public and private network",
			NetworkID:  4711,
			ServiceUID: "3",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "with-priv-net",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				Name:             "with-priv-net",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv4:    hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("1.2.3.4")},
					// IPv6:    hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("fe80::1")},
				},
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{
							ID:   4711,
							Name: "priv-net",
						},
						IP: net.ParseIP("10.10.10.2"),
					},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("GetByName", tt.Ctx, "with-priv-net").
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("Create", tt.Ctx, tt.LB.Name, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				expected := &v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{IP: tt.LB.PublicNet.IPv4.IP.String()},
						// {IP: tt.LB.PublicNet.IPv6.IP.String()},
						{IP: tt.LB.PrivateNet[0].IP.String()},
					},
				}
				lbStat, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
				assert.Equal(t, expected, lbStat)
			},
		},
		{
			Name:       "disable private ingress",
			NetworkID:  4711,
			ServiceUID: "5",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName:                  "with-priv-net-no-priv-ingress",
				annotation.LBDisablePrivateIngress: "true",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				Name:             "with-priv-net-no-priv-ingress",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv4:    hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("1.2.3.4")},
					// IPv6:    hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("fe80::1")},
				},
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{
							ID:   4711,
							Name: "priv-net",
						},
						IP: net.ParseIP("10.10.10.2"),
					},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("GetByName", tt.Ctx, "with-priv-net-no-priv-ingress").
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("Create", tt.Ctx, tt.LB.Name, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				expected := &v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{IP: tt.LB.PublicNet.IPv4.IP.String()},
						// {IP: tt.LB.PublicNet.IPv6.IP.String()},
					},
				}
				lbStat, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
				assert.Equal(t, expected, lbStat)
			},
		},
		{
			Name:       "attach Load Balancer to private network only",
			NetworkID:  4711,
			ServiceUID: "6",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName:                 "priv-net-only",
				annotation.LBDisablePublicNetwork: true,
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				Name:             "priv-net-only",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{
							ID:   4711,
							Name: "priv-net",
						},
						IP: net.ParseIP("10.10.10.2"),
					},
				},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("GetByName", tt.Ctx, "priv-net-only").
					Return(nil, hcops.ErrNotFound)
				tt.LBOps.
					On("Create", tt.Ctx, tt.LB.Name, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
				tt.LBOps.
					On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).
					Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				expected := &v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{IP: tt.LB.PrivateNet[0].IP.String()},
					},
				}
				lbStat, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
				assert.Equal(t, expected, lbStat)
			},
		},
	}

	RunLoadBalancerTests(t, tests)
}

func TestLoadBalancer_EnsureLoadBalancer_UpdateLoadBalancer(t *testing.T) {
	tests := []LoadBalancerTestCase{
		{
			Name:       "Load balancer unchanged",
			ServiceUID: "1",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByK8SServiceUID", tt.Ctx, tt.Service).Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "Load balancer changed",
			ServiceUID: "2",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               2,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByK8SServiceUID", tt.Ctx, tt.Service).Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(true, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("GetByID", tt.Ctx, tt.LB.ID).Times(1).Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "Load balancer targets changed",
			ServiceUID: "3",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               3,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByK8SServiceUID", tt.Ctx, tt.Service).Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(true, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("GetByID", tt.Ctx, tt.LB.ID).Times(1).Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "Load balancer services changed",
			ServiceUID: "4",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               4,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByK8SServiceUID", tt.Ctx, tt.Service).Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(true, nil)
				tt.LBOps.On("GetByID", tt.Ctx, tt.LB.ID).Times(1).Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "fall back to load balancer name",
			ServiceUID: "5",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "pre-existing-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               5,
				Name:             "pre-existing-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByK8SServiceUID", tt.Ctx, tt.Service).Return(nil, hcops.ErrNotFound)
				tt.LBOps.On("GetByName", tt.Ctx, "pre-existing-lb").Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(true, nil)
				tt.LBOps.On("GetByID", tt.Ctx, tt.LB.ID).Times(1).Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				_, err := tt.LoadBalancers.EnsureLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
	}

	RunLoadBalancerTests(t, tests)
}

func TestLoadBalancer_UpdateLoadBalancer(t *testing.T) {
	tests := []LoadBalancerTestCase{
		{
			Name:       "Load Balancer not found",
			ServiceUID: "1",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByK8SServiceUID", tt.Ctx, tt.Service).Return(nil, hcops.ErrNotFound)
				tt.LBOps.On("GetByName", tt.Ctx, "test-lb").Return(nil, hcops.ErrNotFound)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.UpdateLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "calls all reconcilement ops",
			ServiceUID: "2",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "test-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               1,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByK8SServiceUID", tt.Ctx, tt.Service).Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.UpdateLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "fall back to load balancer name",
			ServiceUID: "3",
			ServiceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "previously-created-lb",
			},
			LB: &hcloud.LoadBalancer{
				ID:               3,
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1", NetworkZone: hcloud.NetworkZoneEUCentral},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.On("GetByK8SServiceUID", tt.Ctx, tt.Service).Return(nil, hcops.ErrNotFound)
				tt.LBOps.On("GetByName", tt.Ctx, "previously-created-lb").Return(tt.LB, nil)
				tt.LBOps.On("ReconcileHCLB", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBTargets", tt.Ctx, tt.LB, tt.Service, tt.Nodes).Return(false, nil)
				tt.LBOps.On("ReconcileHCLBServices", tt.Ctx, tt.LB, tt.Service).Return(false, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.UpdateLoadBalancer(tt.Ctx, tt.ClusterName, tt.Service, tt.Nodes)
				assert.NoError(t, err)
			},
		},
	}

	RunLoadBalancerTests(t, tests)
}

func TestLoadBalancers_EnsureLoadBalancerDeleted(t *testing.T) {
	tests := []LoadBalancerTestCase{
		{
			Name:       "delete load balancer",
			ServiceUID: "1",
			LB: &hcloud.LoadBalancer{
				ID:   1,
				Name: "delete me",
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("Delete", tt.Ctx, tt.LB).
					Return(nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.EnsureLoadBalancerDeleted(tt.Ctx, tt.ClusterName, tt.Service)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "delete missing load balancer",
			ServiceUID: "2",
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, hcops.ErrNotFound)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.EnsureLoadBalancerDeleted(tt.Ctx, tt.ClusterName, tt.Service)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "load balancer concurrently deleted",
			ServiceUID: "3",
			LB: &hcloud.LoadBalancer{
				ID:   3,
				Name: "someone else deleted me",
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("Delete", tt.Ctx, tt.LB).
					Return(hcops.ErrNotFound)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.EnsureLoadBalancerDeleted(tt.Ctx, tt.ClusterName, tt.Service)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "delete protected load balancer",
			ServiceUID: "4",
			LB: &hcloud.LoadBalancer{
				ID:         4,
				Name:       "deletion protection enabled",
				Protection: hcloud.LoadBalancerProtection{Delete: true},
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(tt.LB, nil)
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.EnsureLoadBalancerDeleted(tt.Ctx, tt.ClusterName, tt.Service)
				assert.NoError(t, err)
			},
		},
		{
			Name:       "load balancer lookup fails",
			ServiceUID: "5",
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(nil, errors.New("lookup error"))
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.EnsureLoadBalancerDeleted(tt.Ctx, tt.ClusterName, tt.Service)
				assert.EqualError(t, err, "hcloud/loadBalancers.EnsureLoadBalancerDeleted: lookup error")
			},
		},
		{
			Name:       "load balancer deletion fails",
			ServiceUID: "6",
			LB: &hcloud.LoadBalancer{
				ID:   6,
				Name: "can't be deleted",
			},
			Mock: func(t *testing.T, tt *LoadBalancerTestCase) {
				tt.LBOps.
					On("GetByK8SServiceUID", tt.Ctx, tt.Service).
					Return(tt.LB, nil)
				tt.LBOps.
					On("Delete", tt.Ctx, tt.LB).
					Return(errors.New("deletion error"))
			},
			Perform: func(t *testing.T, tt *LoadBalancerTestCase) {
				err := tt.LoadBalancers.EnsureLoadBalancerDeleted(tt.Ctx, tt.ClusterName, tt.Service)
				assert.EqualError(t, err, "hcloud/loadBalancers.EnsureLoadBalancerDeleted: deletion error")
			},
		},
	}

	RunLoadBalancerTests(t, tests)
}
