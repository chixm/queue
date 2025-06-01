package queue

import "errors"

var ErrAlreadyDone = errors.New("task already done")
var ErrTaskNotFound = errors.New("task not found")
