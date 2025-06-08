package queue

import (
	"context"
	"log"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// taskWorker is goroutine that processes tasks from the queue.

type taskWorker struct {
	ctx           context.Context
	CancelAllFunc context.CancelFunc
	errGroup      *errgroup.Group
	maxWorkers    int
	queue         *queue[Task]
	activeTaskCnt atomic.Int32           // Number of active workers
	failedTasks   atomic.Pointer[[]Task] // Pointer to a slice of failed tasks
	logger        Logger
}

func NewWorker(ctx context.Context, maxWorkers int) *taskWorker {
	ctx, cancel := context.WithCancel(ctx)

	if maxWorkers <= 0 {
		maxWorkers = 1 // Default to 1 worker if maxWorkers is not positive
	}
	// error handling
	eg, ctx := errgroup.WithContext(ctx)

	eg.SetLimit(maxWorkers)

	return &taskWorker{
		ctx:           ctx,
		maxWorkers:    maxWorkers,
		queue:         NewQueue[Task](),
		CancelAllFunc: cancel,
		errGroup:      eg,
		activeTaskCnt: atomic.Int32{},
		failedTasks:   atomic.Pointer[[]Task]{},
	}
}

func (w *taskWorker) SetTasks(task []Task) {
	// Add a single task to the queue
	if task == nil {
		return // No task to add
	}
	for _, t := range task {
		w.queue.Enqueue(t)
	}
}

func (w *taskWorker) AddTask(task Task) {
	w.queue.Enqueue(task)
}

func (w *taskWorker) Start() error {
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

			log.Println(`current active workers:`, w.activeTaskCnt.Load())
			w.errGroup.Go(func() error {
				w.activeTaskCnt.Add(1)
				defer w.activeTaskCnt.Add(-1) // Decrement the active worker count when done
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

func (w *taskWorker) Stop() {
	// Stop the worker by canceling the context
	w.CancelAllFunc()
	// Wait for all tasks to complete
	w.errGroup.Wait()
}

func (w *taskWorker) CurrentActiveWorkers() int32 {
	// Return the number of currently active workers
	return w.activeTaskCnt.Load()
}

func (w *taskWorker) FailedTasks() []Task {
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

func (w *taskWorker) Log(v ...any) {
	// Log a message using the provided logger
	if w.logger != nil {
		w.logger.Println(v...)
	}
}
