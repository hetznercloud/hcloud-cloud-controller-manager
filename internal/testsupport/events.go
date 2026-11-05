package testsupport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/record"
)

// AssertEvent asserts that recorder recorded expected, or nothing at all if expected is empty.
func AssertEvent(t *testing.T, recorder *record.FakeRecorder, expected string) {
	t.Helper()

	if expected == "" {
		AssertEvents(t, recorder)
		return
	}

	AssertEvents(t, recorder, expected)
}

// AssertEvents asserts that recorder recorded exactly the expected events in order.
func AssertEvents(t *testing.T, recorder *record.FakeRecorder, expected ...string) {
	t.Helper()

	close(recorder.Events)

	var events []string
	for event := range recorder.Events {
		events = append(events, event)
	}

	assert.Equal(t, expected, events)
}
