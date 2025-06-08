package queue_test

import (
	"context"
	"errors"
	"log"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chixm/queue"
	"github.com/stretchr/testify/assert"
)

var tasks = []*MockTask{
	{waitTime: 300},
	{waitTime: 100},
	{waitTime: 100},
	{waitTime: 100},
	{waitTime: 200},
	{waitTime: 300},
	{waitTime: 200},
	{waitTime: 100},
	{waitTime: 100},
}

func TestWorker(t *testing.T) {
	ctx := context.Background()
	maxWorkers := 5
	worker := queue.NewWorker(ctx, maxWorkers)

	taskInterfaces := make([]queue.Task, len(tasks))
	for i, t := range tasks {
		taskInterfaces[i] = t
	}
	worker.SetTasks(taskInterfaces)

	err := worker.Start()

	assert.Equal(t, err, nil, "Worker should not return an error on successful execution")
}

func TestCancelTaskFromOutsideContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	maxWorkers := 5
	worker := queue.NewWorker(ctx, maxWorkers)

	taskInterfaces := make([]queue.Task, len(tasks))
	for i, t := range tasks {
		taskInterfaces[i] = t
	}
	worker.SetTasks(taskInterfaces)

	go func() {
		time.Sleep(200 * time.Millisecond) // Cancel after some tasks have started
		cancel()
	}()

	err := worker.Start()

	if err != nil {
		log.Printf("Worker finished with error: %v\n", err)
	}
	assert.Equal(t, err, context.Canceled, "Worker should return context.Canceled when tasks are canceled")
}

func TestCancelTaskFromWokder(t *testing.T) {
	ctx := context.Background()
	maxWorkers := 5
	worker := queue.NewWorker(ctx, maxWorkers)

	taskInterfaces := make([]queue.Task, len(tasks))
	for i, t := range tasks {
		taskInterfaces[i] = t
	}
	worker.SetTasks(taskInterfaces)

	go func() {
		time.Sleep(200 * time.Millisecond) // Cancel after some tasks have started
		worker.CancelAllFunc()             // Cancel all tasks
	}()

	err := worker.Start()

	if err != nil {
		log.Printf("Worker finished with error: %v\n", err)
	}
	assert.Equal(t, err, context.Canceled, "Worker should return context.Canceled when tasks are canceled")

}

func TestWorkerWithFailingTask(t *testing.T) {
	ctx := context.Background()
	maxWorkers := 5
	worker := queue.NewWorker(ctx, maxWorkers)
	failingTask := &MockFailingTask{waitTime: 100}
	taskInterfaces := make([]queue.Task, len(tasks)+1)
	for i, t := range tasks {
		taskInterfaces[i] = t
	}
	taskInterfaces[len(tasks)] = failingTask

	worker.SetTasks(taskInterfaces)

	err := worker.Start()

	failedTasks := worker.FailedTasks()

	if err != nil {
		log.Printf("Worker finished with error: %v\n", err)
	}

	// worker does not regard the failed task as an error, it just keeps it in the failedTasks slice for retry.
	assert.Nil(t, err, "Failed tasks should not be nil")
	assert.Equal(t, len(failedTasks), 1)
}

type MockTask struct {
	waitTime int
}

var atomicCounter atomic.Int32

func (m *MockTask) ID() string {
	v := atomicCounter.Add(1) //rand.Int()
	return strconv.Itoa(int(v))
}

func (m *MockTask) Execute(ctx context.Context) error {
	// Simulate some work
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(m.waitTime) * time.Millisecond):
		log.Printf("Task %s completed after %d ms \n", m.ID(), m.waitTime)
		return nil
	}
}

// task that always fails
type MockFailingTask struct {
	waitTime int
}

func (m *MockFailingTask) ID() string {
	v := atomicCounter.Add(1)
	return strconv.Itoa(int(v)) + `F`
}

func (m *MockFailingTask) Execute(ctx context.Context) error {
	// Simulate some work
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(m.waitTime) * time.Millisecond):
		log.Printf("Task %s failed after %d ms \n", m.ID(), m.waitTime)
		return errors.New(`failed while execute task ` + m.ID()) // Simulate a failure
	}
}
