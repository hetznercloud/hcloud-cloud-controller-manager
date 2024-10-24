package utils

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMutexSet(t *testing.T) {
	o := NewMutexSet[int]()

	o.Set(1)
	o.Set(2)
	o.Set(3)

	all := o.All()

	sort.Ints(all)

	assert.Equal(t, []int{1, 2, 3}, all)
}
