package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	stack := NewMutexSet[int]()

	stack.Set(1)
	stack.Set(2)
	stack.Set(3)

	assert.Equal(t, []int{1, 2, 3}, stack.All())
}
