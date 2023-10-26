package testsupport

import (
	"os"
	"testing"
)

// Setenv prepares the environment for testing the
// hcloud-cloud-controller-manager.
func Setenv(t *testing.T, args ...string) func() {
	if len(args)%2 != 0 {
		t.Fatal("Sentenv: uneven number of args")
	}

	newVars := make([]string, 0, len(args)/2)
	oldEnv := make(map[string]string, len(newVars))

	for i := 0; i < len(args); i += 2 {
		k, v := args[i], args[i+1]
		newVars = append(newVars, k)

		if old, ok := os.LookupEnv(k); ok {
			oldEnv[k] = old
		}
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Setenv failed: %v", err)
		}
	}

	return func() {
		for _, k := range newVars {
			v, ok := oldEnv[k]
			if !ok {
				if err := os.Unsetenv(k); err != nil {
					t.Errorf("Unsetenv failed: %v", err)
				}
				continue
			}
			if err := os.Setenv(k, v); err != nil {
				t.Errorf("Setenv failed: %v", err)
			}
		}
	}
}
