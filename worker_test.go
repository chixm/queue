package queue_test

import (
	"context"
	"log"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/chixm/queue"
	"github.com/stretchr/testify/assert"
)

func TestWorker(t *testing.T) {
	ctx := context.Background()
	maxWorkers := 5
	worker := queue.NewWorker[*MockTask](ctx, maxWorkers)

	tasks := []*MockTask{
		{waitTime: 300},
		{waitTime: 100},
		{waitTime: 100},
	}
	taskInterfaces := make([]queue.Task, len(tasks))
	for i, t := range tasks {
		taskInterfaces[i] = t
	}
	worker.SetTasks(taskInterfaces)

	err := worker.Start()

	assert.Equal(t, err, nil, "Worker should not return an error on successful execution")
}

type MockTask struct {
	waitTime int
}

func (m *MockTask) ID() string {
	v := rand.Int()
	return strconv.Itoa(v)
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
