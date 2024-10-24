package utils

import (
	"sync"
)

type MutexSet[T comparable] struct {
	lock sync.Mutex

	data map[T]struct{}
}

func NewMutexSet[T comparable]() *MutexSet[T] {
	return &MutexSet[T]{
		lock: sync.Mutex{},
		data: make(map[T]struct{}, 0),
	}
}

func (s *MutexSet[T]) Set(v T) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data[v] = struct{}{}
}

func (s *MutexSet[T]) Size() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return len(s.data)
}

func (s *MutexSet[T]) All() []T {
	s.lock.Lock()
	defer s.lock.Unlock()

	result := make([]T, 0, len(s.data))
	for value := range s.data {
		result = append(result, value)
	}

	return result
}
