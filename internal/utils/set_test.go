package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMutexSet(t *testing.T) {
	o := NewMutexSet[int]()

	o.Set(1)
	o.Set(2)
	o.Set(3)

	assert.Equal(t, []int{1, 2, 3}, o.All())
}
