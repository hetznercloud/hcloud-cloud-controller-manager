package utils

import (
	"errors"
	"sync"
)

var ErrEmptyStack = errors.New("stack is empty")

type Stack[T any] struct {
	lock sync.Mutex

	s []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{
		sync.Mutex{},
		make([]T, 0),
	}
}

func (s *Stack[T]) Push(v T) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.s = append(s.s, v)
}

func (s *Stack[T]) Pop() (T, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var value T

	l := len(s.s)
	if l == 0 {
		return value, ErrEmptyStack
	}

	value = s.s[l-1]
	s.s = s.s[:l-1]
	return value, nil
}
