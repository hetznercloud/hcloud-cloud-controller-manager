package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
)

type unregisteredObject struct {
	metav1.TypeMeta
}

func (o *unregisteredObject) DeepCopyObject() runtime.Object {
	return &unregisteredObject{TypeMeta: o.TypeMeta}
}

func TestRecorder(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "my-node"}}
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"}}

	tests := []struct {
		name          string
		emit          func(r *Recorder)
		expectedEvent string
		expectedLevel string
		expectedLog   string
	}{
		{
			name: "warning",
			emit: func(r *Recorder) {
				r.Warnf(node, PossibleNodeDeletion, "node %s might be deleted", node.Name)
			},
			expectedEvent: "Warning PossibleNodeDeletion node my-node might be deleted",
			expectedLevel: "I",
			expectedLog:   `"node my-node might be deleted" kind="Node" object="my-node" type="Warning" reason="PossibleNodeDeletion"`,
		},
		{
			name: "normal",
			emit: func(r *Recorder) {
				r.Normalf(svc, LoadBalancerTypeDeprecated, "type %s is deprecated", "lb11")
			},
			expectedEvent: "Normal LoadBalancerTypeDeprecated type lb11 is deprecated",
			expectedLevel: "I",
			expectedLog:   `"type lb11 is deprecated" kind="Service" object="default/my-svc" type="Normal" reason="LoadBalancerTypeDeprecated"`,
		},
		{
			name: "error",
			emit: func(r *Recorder) {
				r.Errorf(svc, InvalidLoadBalancerAnnotation, "Invalid Load Balancer annotation %s", "foo")
			},
			expectedEvent: "Warning InvalidLoadBalancerAnnotation Invalid Load Balancer annotation foo",
			expectedLevel: "E",
			expectedLog:   `"Invalid Load Balancer annotation foo" kind="Service" object="default/my-svc" type="Warning" reason="InvalidLoadBalancerAnnotation"`,
		},
		{
			name: "without args",
			emit: func(r *Recorder) {
				r.Warnf(svc, InternalIPNotConfigured, "no InternalIP found")
			},
			expectedEvent: "Warning InternalIPNotConfigured no InternalIP found",
			expectedLevel: "I",
			expectedLog:   `"no InternalIP found" kind="Service" object="default/my-svc" type="Warning" reason="InternalIPNotConfigured"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := testsupport.CaptureKlog(t)
			fakeRecorder := record.NewFakeRecorder(1)

			tt.emit(NewRecorder(fakeRecorder))

			testsupport.AssertEvent(t, fakeRecorder, tt.expectedEvent)
			assert.Regexp(t, `^`+tt.expectedLevel+`\d`, logs.String())
			assert.Contains(t, logs.String(), tt.expectedLog)
			assert.Regexp(t, `events_test\.go:\d+\] `, logs.String())
		})
	}
}

func TestObjKind(t *testing.T) {
	tests := []struct {
		name     string
		obj      runtime.Object
		expected string
	}{
		{
			name:     "from type meta",
			obj:      &corev1.Node{TypeMeta: metav1.TypeMeta{Kind: "CustomNode"}},
			expected: "CustomNode",
		},
		{
			name:     "from scheme",
			obj:      &corev1.Service{},
			expected: "Service",
		},
		{
			name:     "unregistered type",
			obj:      &unregisteredObject{},
			expected: "*events.unregisteredObject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, objKind(tt.obj))
		})
	}
}

func TestObjRef(t *testing.T) {
	tests := []struct {
		name     string
		obj      runtime.Object
		expected any
	}{
		{
			name:     "cluster scoped",
			obj:      &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "my-node"}},
			expected: klog.KRef("", "my-node"),
		},
		{
			name:     "namespaced",
			obj:      &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"}},
			expected: klog.KRef("default", "my-svc"),
		},
		{
			name:     "without metadata",
			obj:      &unregisteredObject{},
			expected: "*events.unregisteredObject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, objRef(tt.obj))
		})
	}
}
