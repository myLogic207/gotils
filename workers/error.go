package workers

import "errors"

var (
	ErrInitWorkerPool = errors.New("worker pool is not initialized")
)

type WorkerError interface {
	error
	Unwrap() error
}

type InitError struct {
	nested error
}

func (e *InitError) Error() string {
	return "Error initializing worker pool: " + e.nested.Error()
}

func (e *InitError) Unwrap() error {
	return e.nested
}

func (e *InitError) Is(target error) bool {
	return target == ErrInitWorkerPool
}
