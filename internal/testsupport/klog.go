package testsupport

import (
	"bytes"
	"testing"

	"k8s.io/klog/v2"
)

// KlogCapture holds the log output collected by [CaptureKlog].
type KlogCapture struct {
	buf *bytes.Buffer
}

// String returns everything klog has logged so far.
func (c *KlogCapture) String() string {
	klog.Flush()
	return c.buf.String()
}

// CaptureKlog redirects the klog output into a buffer for the duration of the test.
func CaptureKlog(t *testing.T) *KlogCapture {
	t.Helper()

	state := klog.CaptureState()
	t.Cleanup(state.Restore)

	var buf bytes.Buffer
	klog.LogToStderr(false)
	klog.SetOutput(&buf)

	return &KlogCapture{buf: &buf}
}
