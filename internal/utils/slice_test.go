package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	stack := NewMutexSlice[int]()

	stack.Append(1)
	stack.Append(2)
	stack.Append(3)

	assert.Equal(t, []int{1, 2, 3}, stack.All())
}
