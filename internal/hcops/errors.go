package hcops

import "errors"

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
