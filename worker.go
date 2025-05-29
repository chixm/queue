package queue

import (
	"context"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// worker is goroutine that processes tasks from the queue.

type worker[T Task] struct {
	ctx         context.Context
	CancelFunc  context.CancelFunc
	errGroup    *errgroup.Group
	maxWorkers  int
	queue       *queue[Task]
	active      atomic.Int32           // Number of active workers
	failedTasks atomic.Pointer[[]Task] // Pointer to a slice of failed tasks
}

func NewWorker[T any](ctx context.Context, maxWorkers int) *worker[Task] {
	ctx, cancel := context.WithCancel(ctx)

	if maxWorkers <= 0 {
		maxWorkers = 1 // Default to 1 worker if maxWorkers is not positive
	}
	// error handling
	eg, ctx := errgroup.WithContext(ctx)

	eg.SetLimit(maxWorkers)

	return &worker[Task]{
		ctx:         ctx,
		maxWorkers:  maxWorkers,
		queue:       NewQueue[Task](),
		CancelFunc:  cancel,
		errGroup:    eg,
		active:      atomic.Int32{},
		failedTasks: atomic.Pointer[[]Task]{},
	}
}

func (w *worker[Task]) Start() {
	// pull out queue item and execute task
	for {
		select {
		case <-w.ctx.Done():
			return // Exit if the context is canceled
		default:
			task, ok := w.queue.Dequeue() // Run All tasks in the queue
			if !ok {
				return // No more tasks to process
			}
			w.errGroup.Go(func() error {
				// Increment the active worker count
				w.active.Add(1)
				defer w.active.Add(-1) // Decrement when done
				// Execute the task
				if err := task.Execute(w.ctx); err != nil {
					ft := *w.failedTasks.Load() // Append the failed task
					ft = append(ft, task)
					w.failedTasks.Store(&ft)
					return err
				}
				return nil
			})
			if err := w.errGroup.Wait(); err != nil {
				return
			}
		}
	}
}

func (w *worker[Task]) Stop() {
	// Stop the worker by canceling the context
	w.CancelFunc()
	// Wait for all tasks to complete
	w.errGroup.Wait()
}

func (w *worker[Task]) CurrentActiveWorkers() int32 {
	// Return the number of currently active workers
	return w.active.Load()
}
