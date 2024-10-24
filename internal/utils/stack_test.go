package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	stack := NewStack[int]()

	stack.Push(1)
	stack.Push(2)
	stack.Push(3)

	got, err := stack.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 3, got)

	got, err = stack.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 2, got)

	got, err = stack.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 1, got)

	got, err = stack.Pop()
	assert.EqualError(t, err, "stack is empty")
	assert.Equal(t, 0, got)
}
