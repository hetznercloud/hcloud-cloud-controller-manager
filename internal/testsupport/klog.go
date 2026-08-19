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
	// klog buffers its output, flush it so that the caller sees the log
	// records that were written up to this point.
	klog.Flush()
	return c.buf.String()
}

// CaptureKlog redirects the klog output into a buffer for the duration of the
// test, so that tests can assert on what was logged. The previous klog
// configuration is restored once the test finishes.
//
// klog is configured globally, tests using this helper must not run in
// parallel.
func CaptureKlog(t *testing.T) *KlogCapture {
	t.Helper()

	state := klog.CaptureState()
	t.Cleanup(state.Restore)

	var buf bytes.Buffer
	klog.LogToStderr(false)
	klog.SetOutput(&buf)

	return &KlogCapture{buf: &buf}
}
