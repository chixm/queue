package queue

import "context"

type Task interface {
	ID() string // Unique identifier for the task
	Execute(ctx context.Context) error
}
