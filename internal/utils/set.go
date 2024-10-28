package utils

import (
	"sync"
)

type SyncSet[T comparable] struct {
	lock sync.Mutex
	data map[T]struct{}
}

func NewSyncSet[T comparable]() *SyncSet[T] {
	return &SyncSet[T]{
		lock: sync.Mutex{},
		data: make(map[T]struct{}, 0),
	}
}

func (s *SyncSet[T]) Add(v T) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data[v] = struct{}{}
}

func (s *SyncSet[T]) Size() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return len(s.data)
}

func (s *SyncSet[T]) All() []T {
	s.lock.Lock()
	defer s.lock.Unlock()

	result := make([]T, 0, len(s.data))
	for value := range s.data {
		result = append(result, value)
	}

	return result
}
