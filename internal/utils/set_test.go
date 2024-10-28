package utils

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncSet(t *testing.T) {
	o := NewSyncSet[int]()

	o.Add(1)
	o.Add(2)
	o.Add(3)

	all := o.All()
	sort.Ints(all)
	assert.Equal(t, []int{1, 2, 3}, all)
}
