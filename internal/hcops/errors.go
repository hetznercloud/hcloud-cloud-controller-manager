package hcops

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var (
	// ErrNotFound signals that an item was not found by the Hetzner Cloud
	// backend.
	ErrNotFound = errors.New("not found")

	// ErrNonUniqueResult signals that more than one matching item was returned
	// by the Hetzner Cloud backend was returned where only one item was
	// expected.
	ErrNonUniqueResult = errors.New("non-unique result")

	// ErrAlreadyExists signals that the resource creation failed, because the
	// resource already exists.
	ErrAlreadyExists = errors.New("already exists")
)

// withInvalidInputFields adds the validation errors of an 'invalid_input' API
// error to the error message.
func withInvalidInputFields(err error) error {
	apiErr, ok := errors.AsType[hcloud.Error](err)
	if !ok {
		return err
	}
	details, ok := apiErr.Details.(hcloud.ErrorDetailsInvalidInput)
	if !ok || len(details.Fields) == 0 {
		return err
	}

	fields := make([]string, 0, len(details.Fields))
	for _, field := range details.Fields {
		fields = append(fields, fmt.Sprintf("%s (%s)", field.Name, strings.Join(field.Messages, ", ")))
	}

	return fmt.Errorf("%w: invalid fields: %s", err, strings.Join(fields, ", "))
}
