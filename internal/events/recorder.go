package events

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

type Recorder struct {
	recorder record.EventRecorder
}

func NewRecorder(recorder record.EventRecorder) *Recorder {
	return &Recorder{recorder: recorder}
}

// Warnf emits a warning event and a structured info log message.
func (r *Recorder) Warnf(obj runtime.Object, reason Reason, format string, args ...any) {
	r.emit(obj, corev1.EventTypeWarning, reason, fmt.Sprintf(format, args...))
}

// Normalf emits a normal event and a structured info log message.
func (r *Recorder) Normalf(obj runtime.Object, reason Reason, format string, args ...any) {
	r.emit(obj, corev1.EventTypeNormal, reason, fmt.Sprintf(format, args...))
}

// Errorf emits a warning event and a structured error log message.
func (r *Recorder) Errorf(obj runtime.Object, reason Reason, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	r.recorder.Event(obj, corev1.EventTypeWarning, string(reason), msg)
	// Depth 1 skips Warnf/Normalf so file:line points at the caller.
	klog.ErrorSDepth(1, nil, msg, logKeysAndValues(obj, corev1.EventTypeWarning, reason)...)
}

func (r *Recorder) emit(obj runtime.Object, eventType string, reason Reason, msg string) {
	r.recorder.Event(obj, eventType, string(reason), msg)
	// Depth 2 skips emit and Warnf/Normalf so file:line points at the caller.
	klog.InfoSDepth(2, msg, logKeysAndValues(obj, eventType, reason)...)
}

func logKeysAndValues(obj runtime.Object, eventType string, reason Reason) []any {
	return []any{
		"kind", objKind(obj),
		"object", objRef(obj),
		"type", eventType,
		"reason", string(reason),
	}
}

func objRef(obj runtime.Object) any {
	if m, ok := obj.(klog.KMetadata); ok {
		return klog.KObj(m)
	}
	return fmt.Sprintf("%T", obj)
}

func objKind(obj runtime.Object) string {
	if gvk := obj.GetObjectKind().GroupVersionKind(); gvk.Kind != "" {
		return gvk.Kind
	}
	if gvks, _, err := scheme.Scheme.ObjectKinds(obj); err == nil && len(gvks) > 0 {
		return gvks[0].Kind
	}
	return fmt.Sprintf("%T", obj)
}
