package hcops

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestWithInvalidInputFields(t *testing.T) {
	invalidInput := hcloud.Error{
		Code:    hcloud.ErrorCodeInvalidInput,
		Message: "invalid input in field 'http'",
		Details: hcloud.ErrorDetailsInvalidInput{
			Fields: []hcloud.ErrorDetailsInvalidInputField{
				{Name: "http.timeout_idle", Messages: []string{"must be between 5 and 3600"}},
			},
		},
	}

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "invalid input error",
			err:      invalidInput,
			expected: "invalid input in field 'http' (invalid_input): invalid fields: http.timeout_idle (must be between 5 and 3600)",
		},
		{
			name:     "wrapped invalid input error",
			err:      fmt.Errorf("hcops/LoadBalancerOps.ReconcileHCLBServices: %w", invalidInput),
			expected: "hcops/LoadBalancerOps.ReconcileHCLBServices: invalid input in field 'http' (invalid_input): invalid fields: http.timeout_idle (must be between 5 and 3600)",
		},
		{
			name: "invalid input error without details",
			err: hcloud.Error{
				Code:    hcloud.ErrorCodeInvalidInput,
				Message: "invalid input",
			},
			expected: "invalid input (invalid_input)",
		},
		{
			name:     "other api error",
			err:      hcloud.Error{Code: hcloud.ErrorCodeNotFound, Message: "not found"},
			expected: "not found (not_found)",
		},
		{
			name:     "other error",
			err:      errors.New("something went wrong"),
			expected: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := withInvalidInputFields(tt.err)

			assert.EqualError(t, err, tt.expected)
			// The API error must stay accessible for the callers.
			assert.Equal(t, hcloud.IsError(tt.err, hcloud.ErrorCodeInvalidInput), hcloud.IsError(err, hcloud.ErrorCodeInvalidInput))
		})
	}
}
