package queue

import (
	"context"
	"log"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// worker is goroutine that processes tasks from the queue.

type worker[T Task] struct {
	ctx           context.Context
	CancelAllFunc context.CancelFunc
	errGroup      *errgroup.Group
	maxWorkers    int
	queue         *queue[Task]
	active        atomic.Int32           // Number of active workers
	failedTasks   atomic.Pointer[[]Task] // Pointer to a slice of failed tasks
	logger        Logger
}

func NewWorker[T Task](ctx context.Context, maxWorkers int) *worker[Task] {
	ctx, cancel := context.WithCancel(ctx)

	if maxWorkers <= 0 {
		maxWorkers = 1 // Default to 1 worker if maxWorkers is not positive
	}
	// error handling
	eg, ctx := errgroup.WithContext(ctx)

	eg.SetLimit(maxWorkers)

	return &worker[Task]{
		ctx:           ctx,
		maxWorkers:    maxWorkers,
		queue:         NewQueue[Task](),
		CancelAllFunc: cancel,
		errGroup:      eg,
		active:        atomic.Int32{},
		failedTasks:   atomic.Pointer[[]Task]{},
	}
}

func (w *worker[Task]) SetTasks(task []Task) {
	// Add a single task to the queue
	if task == nil {
		return // No task to add
	}
	for _, t := range task {
		w.queue.Enqueue(t)
	}
}

func (w *worker[Task]) Start() error {
	// pull out queue item and execute task
	for {
		select {
		case <-w.ctx.Done():
			return w.ctx.Err() // Exit if the context is canceled
		default:
			task, ok := w.queue.Dequeue() // Run All tasks in the queue
			if !ok {
				return w.errGroup.Wait() // No more tasks to process
			}
			w.active.Add(1)
			log.Println(`current active workers:`, w.active.Load())
			w.errGroup.Go(func() error {
				defer w.active.Add(-1) // Decrement the active worker count when done
				// Execute the task
				if err := task.Execute(w.ctx); err != nil {
					// if task cancelled it's not a failure
					if w.ctx.Err() != nil {
						return w.ctx.Err()
					}
					// If the task fails, store it in the failedTasks slice
					ft := *w.failedTasks.Load() // Append the failed task
					ft = append(ft, task)
					w.failedTasks.Store(&ft)
					// Log the error if a logger is provided
					w.Log("Task failed:", err)
					return nil // task will not stop by single error
				}
				return nil
			})
		}
	}
}

func (w *worker[Task]) Stop() {
	// Stop the worker by canceling the context
	w.CancelAllFunc()
	// Wait for all tasks to complete
	w.errGroup.Wait()
}

func (w *worker[Task]) CurrentActiveWorkers() int32 {
	// Return the number of currently active workers
	return w.active.Load()
}

func (w *worker[Task]) FailedTasks() []Task {
	// Return the slice of failed tasks
	ptr := w.failedTasks.Load()
	if ptr == nil {
		return nil // No failed tasks
	}
	var slice []Task
	for _, task := range *ptr {
		slice = append(slice, task.(Task)) // Type assertion to ensure Task type
	}
	return slice
}

func (w *worker[Task]) Log(v ...any) {
	// Log a message using the provided logger
	if w.logger != nil {
		w.logger.Println(v...)
	}
}
