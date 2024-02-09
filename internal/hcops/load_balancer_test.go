package hcops_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	hrobotmodels "github.com/syself/hrobot-go/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/annotation"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/config"
	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/hcops"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var errTestLbClient = errors.New("lb client failed")

func TestLoadBalancerOps_GetByName(t *testing.T) {
	tests := []struct {
		name   string
		lbName string
		mock   func(t *testing.T, fx *hcops.LoadBalancerOpsFixture)
		lb     *hcloud.LoadBalancer
		err    error
	}{
		{
			name:   "client responds with hcloud.ErrorCodeNotFound",
			lbName: "some-lb",
			mock: func(_ *testing.T, fx *hcops.LoadBalancerOpsFixture) {
				err := hcloud.Error{Code: hcloud.ErrorCodeNotFound}
				fx.LBClient.
					On("GetByName", fx.Ctx, "some-lb").
					Return(nil, nil, err)
			},
			err: hcops.ErrNotFound,
		},
		{
			name:   "Load Balancer is nil",
			lbName: "some-lb",
			mock: func(_ *testing.T, fx *hcops.LoadBalancerOpsFixture) {
				fx.LBClient.
					On("GetByName", fx.Ctx, "some-lb").
					Return(nil, nil, nil)
			},
			err: hcops.ErrNotFound,
		},
		{
			name:   "Load Balancer found",
			lbName: "some-lb",
			mock: func(_ *testing.T, fx *hcops.LoadBalancerOpsFixture) {
				lb := &hcloud.LoadBalancer{ID: 1}
				fx.LBClient.
					On("GetByName", fx.Ctx, "some-lb").
					Return(lb, nil, nil)
			},
			lb: &hcloud.LoadBalancer{ID: 1},
		},
		{
			name:   "client returns other error",
			lbName: "some-lb",
			mock: func(_ *testing.T, fx *hcops.LoadBalancerOpsFixture) {
				fx.LBClient.
					On("GetByName", fx.Ctx, "some-lb").
					Return(nil, nil, errTestLbClient)
			},
			err: errTestLbClient,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fx := hcops.NewLoadBalancerOpsFixture(t)
			if tt.mock != nil {
				tt.mock(t, fx)
			}
			lb, err := fx.LBOps.GetByName(fx.Ctx, tt.lbName)
			if !errors.Is(err, tt.err) {
				t.Errorf("expected error: %v; got: %v", tt.err, err)
			}
			assert.Equal(t, tt.lb, lb)

			fx.AssertExpectations()
		})
	}
}

func TestLoadBalancerOps_GetByID(t *testing.T) {
	tests := []struct {
		name string
		lbID int64
		mock func(t *testing.T, fx *hcops.LoadBalancerOpsFixture)
		lb   *hcloud.LoadBalancer
		err  error
	}{
		{
			name: "client responds with hcloud.ErrorCodeNotFound",
			lbID: 1,
			mock: func(_ *testing.T, fx *hcops.LoadBalancerOpsFixture) {
				err := hcloud.Error{Code: hcloud.ErrorCodeNotFound}
				fx.LBClient.
					On("GetByID", fx.Ctx, int64(1)).
					Return(nil, nil, err)
			},
			err: hcops.ErrNotFound,
		},
		{
			name: "Load Balancer is nil",
			lbID: 2,
			mock: func(_ *testing.T, fx *hcops.LoadBalancerOpsFixture) {
				fx.LBClient.
					On("GetByID", fx.Ctx, int64(2)).
					Return(nil, nil, nil)
			},
			err: hcops.ErrNotFound,
		},
		{
			name: "Load Balancer found",
			lbID: 3,
			mock: func(_ *testing.T, fx *hcops.LoadBalancerOpsFixture) {
				lb := &hcloud.LoadBalancer{ID: 3}
				fx.LBClient.
					On("GetByID", fx.Ctx, int64(3)).
					Return(lb, nil, nil)
			},
			lb: &hcloud.LoadBalancer{ID: 3},
		},
		{
			name: "client returns other error",
			lbID: 4,
			mock: func(_ *testing.T, fx *hcops.LoadBalancerOpsFixture) {
				fx.LBClient.
					On("GetByID", fx.Ctx, int64(4)).
					Return(nil, nil, errTestLbClient)
			},
			err: errTestLbClient,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fx := hcops.NewLoadBalancerOpsFixture(t)
			if tt.mock != nil {
				tt.mock(t, fx)
			}
			lb, err := fx.LBOps.GetByID(fx.Ctx, tt.lbID)
			if !errors.Is(err, tt.err) {
				t.Errorf("expected error: %v; got: %v", tt.err, err)
			}
			assert.Equal(t, tt.lb, lb)

			fx.AssertExpectations()
		})
	}
}

func TestGetByK8SServiceUID(t *testing.T) {
	tests := []struct {
		name      string
		uid       string
		lbs       []*hcloud.LoadBalancer
		err       error
		clientErr error
	}{
		{
			name: "load balancer found",
			uid:  "some-svc-uid",
			lbs: []*hcloud.LoadBalancer{
				{ID: 1, Name: "some-lb"},
			},
		},
		{
			name: "no load balancer found",
			uid:  "missing-svc-uid",
			err:  hcops.ErrNotFound,
		},
		{
			name: "more than one load balancer found",
			uid:  "non-unique",
			lbs: []*hcloud.LoadBalancer{
				{ID: 1, Name: "first-lb"},
				{ID: 2, Name: "second-lb"},
			},
			err: hcops.ErrNonUniqueResult,
		},
		{
			name:      "error when calling backend API",
			uid:       "another-svc-uid",
			clientErr: errors.New("some error"),
			err:       errors.New("hcops/LoadBalancerOps.GetByK8SServiceUID: api error: some error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fx := hcops.NewLoadBalancerOpsFixture(t)

			opts := hcloud.LoadBalancerListOpts{
				ListOpts: hcloud.ListOpts{
					LabelSelector: fmt.Sprintf("%s=%s", hcops.LabelServiceUID, tt.uid),
				},
			}
			fx.LBClient.
				On("AllWithOpts", mock.Anything, opts).
				Return(tt.lbs, tt.clientErr)

			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID(tt.uid),
				},
			}

			lb, err := fx.LBOps.GetByK8SServiceUID(context.Background(), svc)
			if tt.err != nil {
				if tt.clientErr != nil {
					assert.EqualError(t, err, tt.err.Error())
				}
				if tt.clientErr == nil && !errors.Is(err, tt.err) {
					t.Errorf("Expected error: '%v'; got '%v'", tt.err, err)
				}
				return
			}
			if tt.err == nil && err != nil {
				t.Fatalf("Unexpected error: '%v'", err)
			}

			// We expect only one load balancer to be returned if everything
			// is ok. If tt.lb contains an error this should have been handled
			// by one of the error assertions above.
			assert.Equal(t, tt.lbs[0], lb)
		})
	}
}

func TestLoadBalancerOps_Create(t *testing.T) {
	type testCase struct {
		name               string
		cfg                config.HCCMConfiguration
		serviceAnnotations map[annotation.Name]interface{}
		createOpts         hcloud.LoadBalancerCreateOpts
		mock               func(t *testing.T, tt *testCase, fx *hcops.LoadBalancerOpsFixture)
		lb                 *hcloud.LoadBalancer
		err                error
	}
	tests := []testCase{
		{
			name: "create with with location name (and default set)",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					Location: "hel1",
				},
			},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation: "fsn1",
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "some-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location: &hcloud.Location{
					Name: "fsn1",
				},
				Labels: map[string]string{
					hcops.LabelServiceUID: "some-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 1},
		},
		{
			name: "create with network zone name only (and default set)",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					NetworkZone: "eu-central",
				},
			},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBNetworkZone: "eu-central",
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "another-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				NetworkZone:      hcloud.NetworkZoneEUCentral,
				Labels: map[string]string{
					hcops.LabelServiceUID: "another-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 2},
		},
		{
			name: "create with location as default only",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					Location: "fsn1",
				},
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "some-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location: &hcloud.Location{
					Name: "fsn1",
				},
				Labels: map[string]string{
					hcops.LabelServiceUID: "some-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 3},
		},
		{
			name: "create with network zone as default only",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					NetworkZone: "eu-central",
				},
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "some-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				NetworkZone:      hcloud.NetworkZoneEUCentral,
				Labels: map[string]string{
					hcops.LabelServiceUID: "some-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 4},
		},
		{
			name: "create with network zone and reset default location",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					Location: "hel1",
				},
			},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation:    "",
				annotation.LBNetworkZone: "eu-central",
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "another-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				NetworkZone:      hcloud.NetworkZoneEUCentral,
				Labels: map[string]string{
					hcops.LabelServiceUID: "another-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 2},
		},
		{
			name: "create with location and reset default network zone",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					NetworkZone: "eu-central",
				},
			},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation:    "fsn1",
				annotation.LBNetworkZone: "",
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "another-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location: &hcloud.Location{
					Name: "fsn1",
				},
				Labels: map[string]string{
					hcops.LabelServiceUID: "another-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 2},
		},
		{
			name:               "fails if location and network zone missing",
			serviceAnnotations: map[annotation.Name]interface{}{},
			err: fmt.Errorf("hcops/LoadBalancerOps.Create: neither %s nor %s set",
				annotation.LBLocation, annotation.LBNetworkZone),
		},
		{
			name: "gives preference to location name",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation:    "nbg1",
				annotation.LBNetworkZone: "eu-central",
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "another-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1"},
				Labels: map[string]string{
					hcops.LabelServiceUID: "another-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 2},
		},
		{
			name: "set Load Balancer type name",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBType:     "lb21",
				annotation.LBLocation: "nbg1",
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "another-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb21"},
				Location:         &hcloud.Location{Name: "nbg1"},
				Labels: map[string]string{
					hcops.LabelServiceUID: "another-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 3},
		},
		{
			name: "set Load Balancer algorithm type",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation:      "nbg1",
				annotation.LBAlgorithmType: "least_connections",
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "another-lb",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1"},
				Algorithm:        &hcloud.LoadBalancerAlgorithm{Type: hcloud.LoadBalancerAlgorithmTypeLeastConnections},
				Labels: map[string]string{
					hcops.LabelServiceUID: "another-lb-uid",
				},
			},
			lb: &hcloud.LoadBalancer{ID: 4},
		},
		{
			name: "fail on invalid Load Balancer algorithm type",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation:      "nbg1",
				annotation.LBAlgorithmType: "invalidType",
			},
			err: fmt.Errorf("hcops/LoadBalancerOps.Create: annotation/Name.LBAlgorithmTypeFromService: annotation/validateAlgorithmType: invalid: invalidtype"),
		},
		{
			name: "attach Load Balancer to private network",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation: "nbg1",
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "lb-with-priv",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1"},
				Network: &hcloud.Network{
					ID:   4711,
					Name: "some-network",
				},
				Labels: map[string]string{
					hcops.LabelServiceUID: "lb-with-priv-uid",
				},
			},
			mock: func(_ *testing.T, tt *testCase, fx *hcops.LoadBalancerOpsFixture) {
				fx.LBOps.NetworkID = tt.createOpts.Network.ID
				fx.NetworkClient.
					On("GetByID", fx.Ctx, fx.LBOps.NetworkID).
					Return(tt.createOpts.Network, nil, nil)
				action := fx.MockCreate(tt.createOpts, tt.lb, nil)
				fx.MockGetByID(tt.lb, nil)
				fx.MockWatchProgress(action, nil)
			},
			lb: &hcloud.LoadBalancer{ID: 5},
		},
		{
			name: "fail if network could not be found",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation: "nbg1",
			},
			mock: func(_ *testing.T, _ *testCase, fx *hcops.LoadBalancerOpsFixture) {
				fx.LBOps.NetworkID = 4711
				fx.NetworkClient.On("GetByID", fx.Ctx, fx.LBOps.NetworkID).Return(nil, nil, nil)
			},
			err: fmt.Errorf("hcops/LoadBalancerOps.Create: get network %d: %w", 4711, hcops.ErrNotFound),
		},
		{
			name: "fail if looking for network returns an error",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation: "nbg1",
			},
			mock: func(_ *testing.T, _ *testCase, fx *hcops.LoadBalancerOpsFixture) {
				fx.LBOps.NetworkID = 4712
				fx.NetworkClient.On("GetByID", fx.Ctx, fx.LBOps.NetworkID).Return(nil, nil, errTestLbClient)
			},
			err: fmt.Errorf("hcops/LoadBalancerOps.Create: get network %d: %w", 4712, errTestLbClient),
		},
		{
			name: "disable public interface",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBLocation:             "nbg1",
				annotation.LBDisablePublicNetwork: true,
			},
			createOpts: hcloud.LoadBalancerCreateOpts{
				Name:             "lb-with-priv",
				LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
				Location:         &hcloud.Location{Name: "nbg1"},
				PublicInterface:  hcloud.Ptr(false),
				Network: &hcloud.Network{
					ID:   4711,
					Name: "some-network",
				},
				Labels: map[string]string{
					hcops.LabelServiceUID: "lb-with-priv-uid",
				},
			},
			mock: func(_ *testing.T, tt *testCase, fx *hcops.LoadBalancerOpsFixture) {
				fx.LBOps.NetworkID = tt.createOpts.Network.ID

				fx.NetworkClient.On("GetByID", fx.Ctx, fx.LBOps.NetworkID).Return(tt.createOpts.Network, nil, nil)

				action := fx.MockCreate(tt.createOpts, tt.lb, nil)
				fx.MockGetByID(tt.lb, nil)
				fx.MockWatchProgress(action, nil)
			},
			lb: &hcloud.LoadBalancer{ID: 6},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fx := hcops.NewLoadBalancerOpsFixture(t)

			fx.LBOps.Cfg = tt.cfg

			if tt.mock == nil {
				tt.mock = func(_ *testing.T, tt *testCase, fx *hcops.LoadBalancerOpsFixture) {
					if tt.createOpts.Name == "" {
						return
					}
					action := fx.MockCreate(tt.createOpts, tt.lb, nil)
					fx.MockGetByID(tt.lb, nil)
					fx.MockWatchProgress(action, nil)
				}
			}
			tt.mock(t, &tt, fx)

			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID(tt.createOpts.Labels[hcops.LabelServiceUID]),
				},
			}
			for k, v := range tt.serviceAnnotations {
				if err := k.AnnotateService(service, v); err != nil {
					t.Error(err)
				}
			}

			lb, err := fx.LBOps.Create(fx.Ctx, tt.createOpts.Name, service)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.lb, lb)
			fx.AssertExpectations()
		})
	}
}

func TestLoadBalancerOps_Delete(t *testing.T) {
	tests := []struct {
		name      string
		clientErr error
		err       error
	}{
		{
			name: "deletion successful",
		},
		{
			name:      "load balancer not found",
			clientErr: hcloud.Error{Code: hcloud.ErrorCodeNotFound},
		},
		{
			name:      "deletion fails",
			clientErr: errors.New("deletion failed"),
			err:       errors.New("hcops/LoadBalancerOps.Delete: deletion failed"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fx := hcops.NewLoadBalancerOpsFixture(t)
			ctx := context.Background()
			lb := &hcloud.LoadBalancer{ID: 1}

			fx.LBClient.On("Delete", ctx, lb).Return(nil, tt.clientErr)

			err := fx.LBOps.Delete(ctx, lb)
			if tt.err == nil {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.err.Error())
		})
	}
}

type LBReconcilementTestCase struct {
	name               string
	cfg                config.HCCMConfiguration
	serviceUID         string
	serviceAnnotations map[annotation.Name]interface{}
	servicePorts       []corev1.ServicePort
	k8sNodes           []*corev1.Node
	initialLB          *hcloud.LoadBalancer
	robotServers       []hrobotmodels.Server
	mock               func(t *testing.T, tt *LBReconcilementTestCase)
	perform            func(t *testing.T, tt *LBReconcilementTestCase)

	// set during test execution
	service *corev1.Service
	fx      *hcops.LoadBalancerOpsFixture
}

func (tt *LBReconcilementTestCase) run(t *testing.T) {
	t.Helper()

	tt.fx = hcops.NewLoadBalancerOpsFixture(t)
	tt.fx.LBOps.Cfg = tt.cfg

	if tt.service == nil {
		tt.service = &corev1.Service{
			Spec:       corev1.ServiceSpec{Ports: tt.servicePorts},
			ObjectMeta: metav1.ObjectMeta{UID: types.UID(tt.serviceUID)},
		}
	}
	for k, v := range tt.serviceAnnotations {
		if err := k.AnnotateService(tt.service, v); err != nil {
			t.Error(err)
		}
	}
	if tt.mock != nil {
		tt.mock(t, tt)
	}
	tt.perform(t, tt)
	tt.fx.AssertExpectations()
}

func TestLoadBalancerOps_ReconcileHCLB(t *testing.T) {
	tests := []LBReconcilementTestCase{
		{
			name: "update algorithm",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBAlgorithmType: string(hcloud.LoadBalancerAlgorithmTypeLeastConnections),
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 1,
				Algorithm: hcloud.LoadBalancerAlgorithm{
					Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin,
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				opts := hcloud.LoadBalancerChangeAlgorithmOpts{Type: hcloud.LoadBalancerAlgorithmTypeLeastConnections}

				action := &hcloud.Action{ID: 4711}
				tt.fx.LBClient.
					On("ChangeAlgorithm", tt.fx.Ctx, tt.initialLB, opts).
					Return(action, nil, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "update to invalid algorithm",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBAlgorithmType: "invalidType",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 2,
				Algorithm: hcloud.LoadBalancerAlgorithm{
					Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin,
				},
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.EqualError(t, err,
					"hcops/LoadBalancerOps.ReconcileHCLB: hcops/LoadBalancerOps.changeAlgorithm: annotation/Name.LBAlgorithmTypeFromService: annotation/validateAlgorithmType: invalid: invalidtype")
				assert.False(t, changed)
			},
		},
		{
			name: "don't update unchanged algorithm",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBAlgorithmType: string(hcloud.LoadBalancerAlgorithmTypeRoundRobin),
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 3,
				Algorithm: hcloud.LoadBalancerAlgorithm{
					Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin,
				},
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			name: "update type",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBType: "lb21",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 1,
				LoadBalancerType: &hcloud.LoadBalancerType{
					Name: "lb11",
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				opts := hcloud.LoadBalancerChangeTypeOpts{
					LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb21"},
				}

				action := &hcloud.Action{ID: 4711}
				tt.fx.LBClient.
					On("ChangeType", tt.fx.Ctx, tt.initialLB, opts).
					Return(action, nil, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "don't update unchanged type",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBType: "lb21",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 1,
				LoadBalancerType: &hcloud.LoadBalancerType{
					Name: "lb21",
				},
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			name: "don't update correct IPv4 RNDS",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBPublicIPv4RDNS: "lb.example.com",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 6,
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv4: hcloud.LoadBalancerPublicNetIPv4{
						DNSPtr: "lb.example.com",
					},
				},
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			name: "update incorrect IPv4 RNDS",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBPublicIPv4RDNS: "new-name-lb.example.com",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 6,
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv4: hcloud.LoadBalancerPublicNetIPv4{
						DNSPtr: "lb.example.com",
						IP:     net.ParseIP("1.2.3.4"),
					},
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				action := &hcloud.Action{ID: rand.Int63()}
				newRDNS := "new-name-lb.example.com"
				tt.fx.LBClient.On("ChangeDNSPtr", tt.fx.Ctx, tt.initialLB, net.ParseIP("1.2.3.4").String(), &newRDNS).Return(action, nil, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "don't update correct IPv6 RNDS",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBPublicIPv6RDNS: "lb.example.com",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 6,
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv6: hcloud.LoadBalancerPublicNetIPv6{
						DNSPtr: "lb.example.com",
					},
				},
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			name: "update incorrect IPv6 RNDS",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBPublicIPv6RDNS: "new-name-lb.example.com",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 6,
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
					IPv6: hcloud.LoadBalancerPublicNetIPv6{
						DNSPtr: "lb.example.com",
						IP:     net.ParseIP("fe80::1"),
					},
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				action := &hcloud.Action{ID: rand.Int63()}
				newRDNS := "new-name-lb.example.com"
				tt.fx.LBClient.On("ChangeDNSPtr", tt.fx.Ctx, tt.initialLB, net.ParseIP("fe80::1").String(), &newRDNS).Return(action, nil, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "detach Load Balancer from network",
			initialLB: &hcloud.LoadBalancer{
				ID: 4,
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{ID: 14, Name: "some-network"},
					},
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				opts := hcloud.LoadBalancerDetachFromNetworkOpts{
					Network: &hcloud.Network{ID: 14, Name: "some-network"},
				}
				action := &hcloud.Action{ID: rand.Int63()}
				tt.fx.LBClient.On("DetachFromNetwork", tt.fx.Ctx, tt.initialLB, opts).Return(action, nil, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "don't detach Load Balancer from current network",
			initialLB: &hcloud.LoadBalancer{
				ID: 5,
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{ID: 15, Name: "some-network"},
					},
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				tt.fx.LBOps.NetworkID = tt.initialLB.PrivateNet[0].Network.ID
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			name:      "attach Load Balancer to network",
			initialLB: &hcloud.LoadBalancer{ID: 4},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				nw := &hcloud.Network{ID: 15, Name: "some-network"}
				tt.fx.NetworkClient.On("GetByID", tt.fx.Ctx, nw.ID).Return(nw, nil, nil)

				tt.fx.LBOps.NetworkID = nw.ID

				opts := hcloud.LoadBalancerAttachToNetworkOpts{Network: nw}
				action := &hcloud.Action{ID: rand.Int63()}
				tt.fx.LBClient.On("AttachToNetwork", tt.fx.Ctx, tt.initialLB, opts).Return(action, nil, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name:      "re-try attach to network on conflict",
			initialLB: &hcloud.LoadBalancer{ID: 5},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				nw := &hcloud.Network{ID: 15, Name: "some-network"}
				tt.fx.NetworkClient.On("GetByID", tt.fx.Ctx, nw.ID).Return(nw, nil, nil)

				tt.fx.LBOps.NetworkID = nw.ID

				opts := hcloud.LoadBalancerAttachToNetworkOpts{Network: nw}
				tt.fx.LBClient.
					On("AttachToNetwork", tt.fx.Ctx, tt.initialLB, opts).
					Return(nil, nil, hcloud.Error{Code: hcloud.ErrorCodeConflict}).
					Once()

				action := &hcloud.Action{ID: rand.Int63()}
				tt.fx.LBClient.
					On("AttachToNetwork", tt.fx.Ctx, tt.initialLB, opts).
					Return(action, nil, nil).
					Once()

				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name:      "re-try attach to network on locked error",
			initialLB: &hcloud.LoadBalancer{ID: 5},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				nw := &hcloud.Network{ID: 15, Name: "some-network"}
				tt.fx.NetworkClient.On("GetByID", tt.fx.Ctx, nw.ID).Return(nw, nil, nil)

				tt.fx.LBOps.NetworkID = nw.ID

				opts := hcloud.LoadBalancerAttachToNetworkOpts{Network: nw}
				tt.fx.LBClient.
					On("AttachToNetwork", tt.fx.Ctx, tt.initialLB, opts).
					Return(nil, nil, hcloud.Error{Code: hcloud.ErrorCodeLocked}).
					Once()

				action := &hcloud.Action{ID: rand.Int63()}
				tt.fx.LBClient.
					On("AttachToNetwork", tt.fx.Ctx, tt.initialLB, opts).
					Return(action, nil, nil).
					Once()

				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "don't re-attach to current network",
			initialLB: &hcloud.LoadBalancer{
				ID: 5,
				PrivateNet: []hcloud.LoadBalancerPrivateNet{
					{
						Network: &hcloud.Network{ID: 16, Name: "some-network"},
					},
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				tt.fx.LBOps.NetworkID = tt.initialLB.PrivateNet[0].Network.ID
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			name: "disable enabled public network",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBDisablePublicNetwork: true,
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 6,
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				action := &hcloud.Action{ID: rand.Int63()}
				tt.fx.LBClient.
					On("DisablePublicInterface", tt.fx.Ctx, tt.initialLB).
					Return(action, nil, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "keep disabled public interface",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBDisablePublicNetwork: true,
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 7,
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: false,
				},
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			name: "enable disabled public interface",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBDisablePublicNetwork: false,
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 8,
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: false,
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				action := &hcloud.Action{ID: rand.Int63()}
				tt.fx.LBClient.
					On("EnablePublicInterface", tt.fx.Ctx, tt.initialLB).
					Return(action, nil, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "keep enabled public interface",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBDisablePublicNetwork: false,
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 9,
				PublicNet: hcloud.LoadBalancerPublicNet{
					Enabled: true,
				},
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			name:       "add missing service UID label",
			serviceUID: "10",
			initialLB: &hcloud.LoadBalancer{
				ID: 10,
				Labels: map[string]string{
					"some-label": "some-value",
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				updated := *tt.initialLB
				updated.Labels = map[string]string{
					hcops.LabelServiceUID: tt.serviceUID,
					"some-label":          "some-value",
				}
				opts := hcloud.LoadBalancerUpdateOpts{
					Labels: map[string]string{
						hcops.LabelServiceUID: tt.serviceUID,
						"some-label":          "some-value",
					},
				}
				tt.fx.LBClient.
					On("Update", tt.fx.Ctx, tt.initialLB, opts).
					Return(&updated, nil, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
				assert.Equal(t, tt.serviceUID, tt.initialLB.Labels[hcops.LabelServiceUID])
				assert.Equal(t, "some-value", tt.initialLB.Labels["some-label"])
			},
		},
		{
			name:       "rename load balancer",
			serviceUID: "11",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBName: "new-name",
			},
			initialLB: &hcloud.LoadBalancer{
				ID:   11,
				Name: "old-name",
				Labels: map[string]string{
					hcops.LabelServiceUID: "11",
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				updated := *tt.initialLB
				updated.Name = "new-name"
				opts := hcloud.LoadBalancerUpdateOpts{Name: "new-name"}
				tt.fx.LBClient.
					On("Update", tt.fx.Ctx, tt.initialLB, opts).
					Return(&updated, nil, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLB(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
				assert.Equal(t, "new-name", tt.initialLB.Name)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, tt.run)
	}
}

func TestLoadBalancerOps_ReconcileHCLBTargets(t *testing.T) {
	tests := []LBReconcilementTestCase{
		{
			name: "add k8s nodes as hc Load Balancer targets",
			k8sNodes: []*corev1.Node{
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://1"}},
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://2"}},
				{Spec: corev1.NodeSpec{ProviderID: "hrobot://3"}, ObjectMeta: metav1.ObjectMeta{Name: "robot-3"}},
				{Spec: corev1.NodeSpec{ProviderID: "hrobot://4"}, ObjectMeta: metav1.ObjectMeta{Name: "robot-4"}},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 1,
				LoadBalancerType: &hcloud.LoadBalancerType{
					MaxTargets: 25,
				},
			},
			robotServers: []hrobotmodels.Server{
				{
					ServerNumber:  3,
					Name:          "robot-3",
					ServerIP:      "1.2.3.4",
					ServerIPv6Net: "2a01:f48:111:4221::",
				},
				{
					ServerNumber:  4,
					Name:          "robot-4",
					ServerIP:      "1.2.3.5",
					ServerIPv6Net: "2a01:f48:111:4222::",
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				opts := hcloud.LoadBalancerAddServerTargetOpts{Server: &hcloud.Server{ID: 1}, UsePrivateIP: hcloud.Ptr(false)}
				action := tt.fx.MockAddServerTarget(tt.initialLB, opts, nil)
				tt.fx.MockWatchProgress(action, nil)

				opts = hcloud.LoadBalancerAddServerTargetOpts{Server: &hcloud.Server{ID: 2}, UsePrivateIP: hcloud.Ptr(false)}
				action = tt.fx.MockAddServerTarget(tt.initialLB, opts, nil)
				tt.fx.MockWatchProgress(action, nil)

				optsIP := hcloud.LoadBalancerAddIPTargetOpts{IP: net.ParseIP("1.2.3.4")}
				action = tt.fx.MockAddIPTarget(tt.initialLB, optsIP, nil)
				tt.fx.MockWatchProgress(action, nil)

				optsIP = hcloud.LoadBalancerAddIPTargetOpts{IP: net.ParseIP("1.2.3.5")}
				action = tt.fx.MockAddIPTarget(tt.initialLB, optsIP, nil)
				tt.fx.MockWatchProgress(action, nil)

				tt.fx.MockListRobotServers(tt.robotServers, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBTargets(tt.fx.Ctx, tt.initialLB, tt.service, tt.k8sNodes)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{DisableIPv6: false},
				Robot:        config.RobotConfiguration{Enabled: true},
			},
		},
		{
			name: "remove unused k8s nodes from hc Load Balancer",
			k8sNodes: []*corev1.Node{
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://1"}},
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://2"}},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 2,
				Targets: []hcloud.LoadBalancerTarget{
					{
						Type:   hcloud.LoadBalancerTargetTypeServer,
						Server: &hcloud.LoadBalancerTargetServer{Server: &hcloud.Server{ID: 1}},
					},
					{
						Type:   hcloud.LoadBalancerTargetTypeServer,
						Server: &hcloud.LoadBalancerTargetServer{Server: &hcloud.Server{ID: 2}},
					},
					{
						Type:   hcloud.LoadBalancerTargetTypeServer,
						Server: &hcloud.LoadBalancerTargetServer{Server: &hcloud.Server{ID: 3}},
					},
					{
						Type:   hcloud.LoadBalancerTargetTypeServer,
						Server: &hcloud.LoadBalancerTargetServer{Server: &hcloud.Server{ID: 4}},
					},
					{
						Type: hcloud.LoadBalancerTargetTypeIP,
						IP:   &hcloud.LoadBalancerTargetIP{IP: "1.2.3.4"},
					},
				},
			},
			robotServers: []hrobotmodels.Server{
				{
					ServerNumber:  5,
					ServerIP:      "1.2.3.4",
					ServerIPv6Net: "2a01:f48:111:4221::",
				},
				{
					ServerNumber:  6,
					ServerIP:      "1.2.3.5",
					ServerIPv6Net: "2a01:f48:111:4222::",
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				action := tt.fx.MockRemoveServerTarget(tt.initialLB, &hcloud.Server{ID: 3}, nil)
				tt.fx.MockWatchProgress(action, nil)

				action = tt.fx.MockRemoveServerTarget(tt.initialLB, &hcloud.Server{ID: 4}, nil)
				tt.fx.MockWatchProgress(action, nil)

				action = tt.fx.MockRemoveIPTarget(tt.initialLB, net.ParseIP("1.2.3.4"), nil)
				tt.fx.MockWatchProgress(action, nil)

				tt.fx.MockListRobotServers(tt.robotServers, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBTargets(tt.fx.Ctx, tt.initialLB, tt.service, tt.k8sNodes)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
			cfg: config.HCCMConfiguration{LoadBalancer: config.LoadBalancerConfiguration{DisableIPv6: true}},
		},
		{
			name: "too many targets",
			k8sNodes: []*corev1.Node{
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://1"}},
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://2"}},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 2,
				Targets: []hcloud.LoadBalancerTarget{
					{
						Type:   hcloud.LoadBalancerTargetTypeServer,
						Server: &hcloud.LoadBalancerTargetServer{Server: &hcloud.Server{ID: 1}},
					},
				},
				LoadBalancerType: &hcloud.LoadBalancerType{
					MaxTargets: 1,
				},
			},
			mock: func(_ *testing.T, _ *LBReconcilementTestCase) {
				// Nothing to mock because no action will be taken besides logging an info message,
				// will fail if an action would be taken instead.
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBTargets(tt.fx.Ctx, tt.initialLB, tt.service, tt.k8sNodes)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
			cfg: config.HCCMConfiguration{LoadBalancer: config.LoadBalancerConfiguration{DisableIPv6: true}},
		},
		{
			name: "enable use of private network via default",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					// Make sure the annotation overrides the default
					UsePrivateIP: true,
					DisableIPv6:  true,
				},
			},
			k8sNodes: []*corev1.Node{
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://1"}},
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://2"}},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 3,
				LoadBalancerType: &hcloud.LoadBalancerType{
					MaxTargets: 25,
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				tt.fx.LBOps.NetworkID = 4711

				opts := hcloud.LoadBalancerAddServerTargetOpts{Server: &hcloud.Server{ID: 1}, UsePrivateIP: hcloud.Ptr(true)}
				action := tt.fx.MockAddServerTarget(tt.initialLB, opts, nil)
				tt.fx.MockWatchProgress(action, nil)

				opts = hcloud.LoadBalancerAddServerTargetOpts{Server: &hcloud.Server{ID: 2}, UsePrivateIP: hcloud.Ptr(true)}
				action = tt.fx.MockAddServerTarget(tt.initialLB, opts, nil)
				tt.fx.MockWatchProgress(action, nil)

				tt.fx.MockListRobotServers(nil, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBTargets(tt.fx.Ctx, tt.initialLB, tt.service, tt.k8sNodes)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "enable use of private network via annotation",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					// Make sure the annotation overrides the default
					UsePrivateIP: false,
					DisableIPv6:  true,
				},
			},
			k8sNodes: []*corev1.Node{
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://1"}},
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://2"}},
			},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBUsePrivateIP: "true",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 3,
				LoadBalancerType: &hcloud.LoadBalancerType{
					MaxTargets: 25,
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				tt.fx.LBOps.NetworkID = 4711

				opts := hcloud.LoadBalancerAddServerTargetOpts{Server: &hcloud.Server{ID: 1}, UsePrivateIP: hcloud.Ptr(true)}
				action := tt.fx.MockAddServerTarget(tt.initialLB, opts, nil)
				tt.fx.MockWatchProgress(action, nil)

				opts = hcloud.LoadBalancerAddServerTargetOpts{Server: &hcloud.Server{ID: 2}, UsePrivateIP: hcloud.Ptr(true)}
				action = tt.fx.MockAddServerTarget(tt.initialLB, opts, nil)
				tt.fx.MockWatchProgress(action, nil)

				tt.fx.MockListRobotServers(nil, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBTargets(tt.fx.Ctx, tt.initialLB, tt.service, tt.k8sNodes)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "disable use of private network via annotation",
			cfg: config.HCCMConfiguration{
				LoadBalancer: config.LoadBalancerConfiguration{
					// Make sure the annotation overrides the default
					UsePrivateIP: true,
					DisableIPv6:  true,
				},
			},
			k8sNodes: []*corev1.Node{
				{Spec: corev1.NodeSpec{ProviderID: "hcloud://1"}},
			},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBUsePrivateIP: "false",
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 4,
				Targets: []hcloud.LoadBalancerTarget{
					{
						Type:         hcloud.LoadBalancerTargetTypeServer,
						Server:       &hcloud.LoadBalancerTargetServer{Server: &hcloud.Server{ID: 1}},
						UsePrivateIP: true,
					},
				},
				LoadBalancerType: &hcloud.LoadBalancerType{
					MaxTargets: 25,
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				action := tt.fx.MockRemoveServerTarget(tt.initialLB, &hcloud.Server{ID: 1}, nil)
				tt.fx.MockWatchProgress(action, nil)

				opts := hcloud.LoadBalancerAddServerTargetOpts{
					Server:       &hcloud.Server{ID: 1},
					UsePrivateIP: hcloud.Ptr(false),
				}
				action = tt.fx.MockAddServerTarget(tt.initialLB, opts, nil)
				tt.fx.MockWatchProgress(action, nil)

				tt.fx.MockListRobotServers(nil, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBTargets(tt.fx.Ctx, tt.initialLB, tt.service, tt.k8sNodes)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, tt.run)
	}
}

func TestLoadBalancerOps_ReconcileHCLBServices(t *testing.T) {
	tests := []LBReconcilementTestCase{
		{
			name: "add services to hc Load Balancer",
			servicePorts: []corev1.ServicePort{
				{Port: 80, NodePort: 8080},
				{Port: 443, NodePort: 8443},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 4,
				LoadBalancerType: &hcloud.LoadBalancerType{
					MaxTargets: 25,
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				opts := hcloud.LoadBalancerAddServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
					ListenPort:      hcloud.Ptr(80),
					DestinationPort: hcloud.Ptr(8080),
					HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8080),
					},
				}
				action := tt.fx.MockAddService(opts, tt.initialLB, nil)
				tt.fx.MockWatchProgress(action, nil)

				opts = hcloud.LoadBalancerAddServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
					ListenPort:      hcloud.Ptr(443),
					DestinationPort: hcloud.Ptr(8443),
					HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8443),
					},
				}
				action = tt.fx.MockAddService(opts, tt.initialLB, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBServices(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "reference TLS certificate by id",
			servicePorts: []corev1.ServicePort{
				{Port: 443, NodePort: 8443},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 10,
				LoadBalancerType: &hcloud.LoadBalancerType{
					MaxTargets: 25,
				},
			},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcHTTPCertificates: []string{"1"},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				opts := hcloud.LoadBalancerAddServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
					ListenPort:      hcloud.Ptr(443),
					DestinationPort: hcloud.Ptr(8443),
					HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
						Certificates: []*hcloud.Certificate{
							{ID: 1},
						},
					},
					HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8443),
					},
				}
				action := tt.fx.MockAddService(opts, tt.initialLB, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBServices(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "reference TLS certificate by name",
			servicePorts: []corev1.ServicePort{
				{Port: 443, NodePort: 8443},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 10,
				LoadBalancerType: &hcloud.LoadBalancerType{
					MaxTargets: 25,
				},
			},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcHTTPCertificates: []string{"some-cert"},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				cert := &hcloud.Certificate{ID: 1}

				opts := hcloud.LoadBalancerAddServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
					ListenPort:      hcloud.Ptr(443),
					DestinationPort: hcloud.Ptr(8443),
					HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
						Certificates: []*hcloud.Certificate{cert},
					},
					HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8443),
					},
				}
				tt.fx.CertClient.
					On("Get", mock.Anything, "some-cert").
					Return(cert, nil, nil)
				action := tt.fx.MockAddService(opts, tt.initialLB, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBServices(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name:         "create managed certificate",
			servicePorts: []corev1.ServicePort{{Port: 443, NodePort: 8443}},
			initialLB:    &hcloud.LoadBalancer{ID: 11},
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcHTTPCertificateType:           hcloud.CertificateTypeManaged,
				annotation.LBSvcHTTPManagedCertificateDomains: []string{"example.com", "*.example.com"},
			},
			serviceUID: "some service uid",
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				cert := &hcloud.Certificate{ID: 1}

				tt.fx.CertClient.
					On("CreateCertificate", mock.Anything, hcloud.CertificateCreateOpts{
						Name:        "ccm-managed-certificate-some service uid",
						Type:        hcloud.CertificateTypeManaged,
						DomainNames: []string{"example.com", "*.example.com"},
						Labels:      map[string]string{hcops.LabelServiceUID: tt.serviceUID},
					}).
					Return(hcloud.CertificateCreateResult{Certificate: cert}, nil, nil)

				tt.fx.CertClient.
					On(
						"AllWithOpts",
						mock.Anything,
						hcloud.CertificateListOpts{
							ListOpts: hcloud.ListOpts{
								LabelSelector: fmt.Sprintf("%s=%s", hcops.LabelServiceUID, tt.serviceUID),
							},
						}).
					Return([]*hcloud.Certificate{cert}, nil, nil)

				opts := hcloud.LoadBalancerAddServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
					ListenPort:      hcloud.Ptr(443),
					DestinationPort: hcloud.Ptr(8443),
					HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
						Certificates: []*hcloud.Certificate{cert},
					},
					HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8443),
					},
				}
				action := tt.fx.MockAddService(opts, tt.initialLB, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBServices(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "replace hc Load Balancer services",
			serviceAnnotations: map[annotation.Name]interface{}{
				annotation.LBSvcProtocol: string(hcloud.LoadBalancerServiceProtocolHTTP),
			},
			servicePorts: []corev1.ServicePort{
				{Port: 81, NodePort: 8081},
				{Port: 444, NodePort: 8444},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 4,
				Services: []hcloud.LoadBalancerService{
					{ListenPort: 80, DestinationPort: 8080},
					{ListenPort: 443, DestinationPort: 8443},
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				opts := hcloud.LoadBalancerAddServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
					ListenPort:      hcloud.Ptr(81),
					DestinationPort: hcloud.Ptr(8081),
					HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8081),
					},
				}
				action := tt.fx.MockAddService(opts, tt.initialLB, nil)
				tt.fx.MockWatchProgress(action, nil)

				opts = hcloud.LoadBalancerAddServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolHTTP,
					ListenPort:      hcloud.Ptr(444),
					DestinationPort: hcloud.Ptr(8444),
					HealthCheck: &hcloud.LoadBalancerAddServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8444),
					},
				}
				action = tt.fx.MockAddService(opts, tt.initialLB, nil)
				tt.fx.MockWatchProgress(action, nil)

				action = tt.fx.MockDeleteService(tt.initialLB, 80, nil)
				tt.fx.MockWatchProgress(action, nil)
				action = tt.fx.MockDeleteService(tt.initialLB, 443, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBServices(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
		{
			name: "update already exposed ports with new hc Load Balancer services",
			servicePorts: []corev1.ServicePort{
				{Port: 80, NodePort: 8081},
				{Port: 443, NodePort: 8444},
			},
			initialLB: &hcloud.LoadBalancer{
				ID: 5,
				Services: []hcloud.LoadBalancerService{
					{ListenPort: 80, DestinationPort: 8080},
					{ListenPort: 443, DestinationPort: 8443},
				},
			},
			mock: func(_ *testing.T, tt *LBReconcilementTestCase) {
				opts := hcloud.LoadBalancerUpdateServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
					DestinationPort: hcloud.Ptr(8081),
					HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8081),
					},
				}
				action := tt.fx.MockUpdateService(opts, tt.initialLB, 80, nil)
				tt.fx.MockWatchProgress(action, nil)

				opts = hcloud.LoadBalancerUpdateServiceOpts{
					Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
					DestinationPort: hcloud.Ptr(8444),
					HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolTCP,
						Port:     hcloud.Ptr(8444),
					},
				}
				action = tt.fx.MockUpdateService(opts, tt.initialLB, 443, nil)
				tt.fx.MockWatchProgress(action, nil)
			},
			perform: func(t *testing.T, tt *LBReconcilementTestCase) {
				changed, err := tt.fx.LBOps.ReconcileHCLBServices(tt.fx.Ctx, tt.initialLB, tt.service)
				assert.NoError(t, err)
				assert.True(t, changed)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, tt.run)
	}
}
