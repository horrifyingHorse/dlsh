package datastruct

import "fmt"

type HeapNode[T any] struct {
	Data     T
	Priority uint
}

type HeapType int8

const (
	MaxHeap HeapType = 0
	MinHeap HeapType = 1
)

type Heap[T any] struct {
	data     []*HeapNode[T]
	size     int
	capacity int
	heapType HeapType
}

func (heap *Heap[T]) Size() int {
	return heap.size
}

func (heap *Heap[T]) Insert(item T, priority uint) {
	node := new(HeapNode[T])
	node.Data = item
	node.Priority = priority
	heap.InsertNode(node)
}

func (heap *Heap[T]) InsertNode(node *HeapNode[T]) error {
	if heap.size != heap.capacity {
		return fmt.Errorf("Heap.Insert() not possible, size != cap")
	}
	heap.data = append(heap.data, node)
	heap.size = len(heap.data)
	heap.capacity = heap.size
	index := heap.size - 1

	switch heap.heapType {
	case MinHeap:
		heap.restoreMinHeap(index)
	case MaxHeap:
		heap.restoreMaxHeap(index)
	}
	return nil
}

func (heap *Heap[T]) restoreMinHeap(index int) {
	for index > 0 && heap.data[(index-1)/2].Priority > heap.data[index].Priority {
		heap.data[index], heap.data[(index-1)/2] = heap.data[(index-1)/2], heap.data[index]
		index = (index - 1) / 2
	}
}

func (heap *Heap[T]) restoreMaxHeap(index int) {
	for index > 0 && heap.data[(index-1)/2].Priority < heap.data[index].Priority {
		heap.data[index], heap.data[(index-1)/2] = heap.data[(index-1)/2], heap.data[index]
		index = (index - 1) / 2
	}
}

func (heap *Heap[T]) Heapify(heapType HeapType) error {
	lastNonLeaf := heap.Size()/2 - 1
	if lastNonLeaf < 0 {
		return fmt.Errorf("invalid index %d", lastNonLeaf)
	}

	switch heapType {
	case MaxHeap:
		for i := lastNonLeaf; i >= 0; i-- {
			maxHeapify(heap, i)
		}
	case MinHeap:
		for i := lastNonLeaf; i >= 0; i-- {
			minHeapify(heap, i)
		}
	}
	return nil
}

func maxHeapify[T any](heap *Heap[T], i int) {
	largest := i
	left := 2*i + 1
	right := 2*i + 2
	data := heap.data

	if left < heap.size && data[left].Priority > data[largest].Priority {
		largest = left
	}

	if right < heap.size && data[right].Priority > data[largest].Priority {
		largest = right
	}

	if largest != i {
		heap.data[i], heap.data[largest] = heap.data[largest], heap.data[i]
		maxHeapify(heap, largest)
	}
}

func minHeapify[T any](heap *Heap[T], i int) {
	smallest := i
	left := 2*i + 1
	right := 2*i + 2
	data := heap.data

	if left < heap.size && data[left].Priority < data[smallest].Priority {
		smallest = left
	}

	if right < heap.size && data[right].Priority < data[smallest].Priority {
		smallest = right
	}

	if smallest != i {
		heap.data[i], heap.data[smallest] = heap.data[smallest], heap.data[i]
		minHeapify(heap, smallest)
	}
}

func (heap *Heap[T]) Top() (T, error) {
	var def T
	if heap.size == 0 {
		return def, fmt.Errorf("heap size of 0 has no elements")
	}
	return heap.data[0].Data, nil
}

func (heap *Heap[T]) TopPriority() (uint, error) {
	if heap.size == 0 {
		return 0, fmt.Errorf("heap size of 0 has no elements")
	}
	return heap.data[0].Priority, nil
}

func (heap *Heap[T]) HasNext() bool {
	return heap.size > 1
}

func (heap *Heap[T]) Next() (T, error) {
	var res T

	if heap.size == 0 || heap.capacity == 0 {
		return res, fmt.Errorf("heap is empty")
	} else if heap.size == 1 {
		return res, fmt.Errorf("heap does not have a Next()")
	}
	res = heap.data[0].Data
	heap.data[0], heap.data[heap.size-1] = heap.data[heap.size-1], heap.data[0]
	heap.size--

	switch heap.heapType {
	case MaxHeap:
		maxHeapify(heap, 0)
	case MinHeap:
		minHeapify(heap, 0)
	}
	return res, nil
}

func (heap *Heap[T]) HasPrev() bool {
	return heap.size < heap.capacity
}

func (heap *Heap[T]) Prev() (T, error) {
	var res T

	if !heap.HasPrev() {
		return res, fmt.Errorf("heap does not have a Prev()")
	}
	res = heap.data[0].Data
	heap.size++

	index := heap.size - 1
	switch heap.heapType {
	case MinHeap:
		heap.restoreMinHeap(index)
	case MaxHeap:
		heap.restoreMaxHeap(index)
	}
	return res, nil
}
