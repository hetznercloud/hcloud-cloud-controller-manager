package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

func WarnEventLogf(recorder record.EventRecorder, obj runtime.Object, reason string, msg string, args ...any) {
	msgf := fmt.Sprintf(msg, args...)
	recorder.Event(obj, corev1.EventTypeWarning, reason, msgf)
	klog.Warning(msgf)
}
