package utils

import (
	"sync"
)

type MutexSlice[T any] struct {
	lock sync.Mutex

	s []T
}

func NewMutexSlice[T any]() *MutexSlice[T] {
	return &MutexSlice[T]{
		sync.Mutex{},
		make([]T, 0),
	}
}

func (s *MutexSlice[T]) Append(v ...T) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.s = append(s.s, v...)
}

func (s *MutexSlice[T]) Size() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return len(s.s)
}

func (s *MutexSlice[T]) All() []T {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.s[:]
}
