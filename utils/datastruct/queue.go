package datastruct

import (
	"fmt"
	"os"
)

type Queue[T any] struct {
	data []T
	size uint
}

func (q *Queue[T]) Front() T {
	if q.size == 0 {
		fmt.Println("Queue.Front(): queue is empty")
		os.Exit(1)
	}
	return q.data[0]
}

func (q *Queue[T]) Rear() T {
	if q.size == 0 {
		fmt.Println("Queue.Rear(): queue is empty")
		os.Exit(1)
	}
	return q.data[q.size-1]
}

func (q *Queue[T]) Enqueue(item T) {
	q.data = append(q.data, item)
	q.size++
}

func (q *Queue[T]) Dequeue() T {
	item := q.Front()
	if q.size == 1 {
		q.data = []T{}
	} else {
		q.data = q.data[1:]
	}
	q.size--
	return item
}

func (q *Queue[T]) Empty() bool {
	return q.size == 0
}
