package queue

import "sync"

func newQueue[T any]() *queue[T] {
	return &queue[T]{
		items: make([]T, 0),
		mux:   sync.Mutex{},
	}
}

type queue[T any] struct {
	items []T
	mux   sync.Mutex
}

func (q *queue[T]) Enqueue(item T) {
	q.mux.Lock()
	defer q.mux.Unlock()
	q.items = append(q.items, item)
}

func (q *queue[T]) Dequeue() (T, bool) {
	q.mux.Lock()
	defer q.mux.Unlock()

	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *queue[T]) AppedToTop(item T) {
	q.mux.Lock()
	defer q.mux.Unlock()
	q.items = append([]T{item}, q.items...)
}
