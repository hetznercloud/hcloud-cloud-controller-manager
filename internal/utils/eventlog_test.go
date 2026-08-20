package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/hetznercloud/hcloud-cloud-controller-manager/internal/testsupport"
)

func TestWarnEventLogf(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		msg      string
		args     []any
		expected string
	}{
		{
			name:     "without args",
			reason:   "InternalIPNotConfigured",
			msg:      "no InternalIP found for node",
			expected: "no InternalIP found for node",
		},
		{
			name:     "with args",
			reason:   "UnsupportedProtocolConfigured",
			msg:      "unsupported protocol %s for service %s",
			args:     []any{corev1.ProtocolUDP, "my-service"},
			expected: "unsupported protocol UDP for service my-service",
		},
		{
			name:     "verbs in args are not expanded again",
			reason:   "LoadBalancerTypeUnconfigured",
			msg:      "set it with the annotation %q",
			args:     []any{"load-balancer.hetzner.cloud/type=%s"},
			expected: `set it with the annotation "load-balancer.hetzner.cloud/type=%s"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := testsupport.CaptureKlog(t)
			recorder := record.NewFakeRecorder(1)

			WarnEventLogf(recorder, &corev1.Node{}, tt.reason, tt.msg, tt.args...)

			assert.Equal(t, "Warning "+tt.reason+" "+tt.expected, <-recorder.Events)

			// klog prefix `W` for warning
			assert.Regexp(t, `^W\d`, logs.String())
			assert.Contains(t, logs.String(), tt.expected)
		})
	}
}

func TestWarnEventLogfEventObject(t *testing.T) {
	testsupport.CaptureKlog(t)

	recorder := record.NewFakeRecorder(1)
	recorder.IncludeObject = true

	node := &corev1.Node{
		TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "my-node"},
	}

	WarnEventLogf(recorder, node, "ServerNotFound", "no server with id %d was found in Robot", 42)

	// FakeRecorder adds `involvedObject` section
	assert.Equal(
		t,
		"Warning ServerNotFound no server with id 42 was found in Robot involvedObject{kind=Node,apiVersion=v1}",
		<-recorder.Events,
	)
}
