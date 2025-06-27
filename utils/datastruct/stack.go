package datastruct

import (
	"fmt"
	"os"
)

type Stack[T any] struct {
	data []T
	size uint
}

func (s *Stack[T]) IsEmpty() bool {
	return s.size == 0
}

func (s *Stack[T]) Top() T {
	if s.IsEmpty() {
		fmt.Println("Stack.Top(): Stack is empty")
		os.Exit(1)
	}
	return s.data[s.size-1]
}

func (s *Stack[T]) Push(item T) {
	s.data = append(s.data, item)
	s.size++
}

func (s *Stack[T]) Pop() T {
	item := s.Top()
	s.data = s.data[:s.size-1]
	s.size--
	return item
}
