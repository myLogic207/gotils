package workers

import (
	"context"
	"fmt"
	log "gotils/logger"
	"sync"
)

type contextKey string

var workerId = contextKey("worker_id")

type WorkerPool struct {
	logger    *log.Logger
	waitGroup sync.WaitGroup
	quit      chan bool
	tasks     chan Task
}

func NewWorkerPool(ctx context.Context, size int, logger *log.Logger) (*WorkerPool, WorkerError) {
	pool := &WorkerPool{
		logger:    logger,
		waitGroup: sync.WaitGroup{},
		quit:      make(chan bool),
		tasks:     make(chan Task, size),
	}
	pool.logger.Debug(fmt.Sprintf("Creating worker pool with %d workers", size))
	for i := 0; i < size; i++ {
		workerCtx := context.WithValue(ctx, workerId, i)
		go pool.worker(workerCtx)
	}
	go pool.stopOnCancel(ctx)
	pool.logger.Debug("Worker pool initialized")
	return pool, nil
}

func (w *WorkerPool) stopOnCancel(ctx context.Context) {
	<-ctx.Done()
	w.logger.Warn("Stopping worker pool")
	w.logger.Info("Awaiting worker pool to finish")
	close(w.tasks)
	w.logger.Debug(fmt.Sprintf("Tasks in queue: %d", len(w.tasks)))
	w.waitGroup.Wait()
	w.logger.Info("Worker pool finished")
}

func (w *WorkerPool) Add(task Task) {
	w.logger.Debug(fmt.Sprintf("Received, task. Tasks in queue: %d", len(w.tasks)))
	if len(w.tasks)-1 >= cap(w.tasks) {
		w.logger.Warn("Worker pool is full or closed, dropping task")
		return
	}
	w.tasks <- task
	w.logger.Info("Adding task to worker pool")
	w.waitGroup.Add(1)
}

func (w *WorkerPool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("Worker %d received quit signal", ctx.Value(workerId).(int))
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			if err := task.Do(ctx); err != nil {
				w.logger.Error(err.Error())
				task.OnError(err)
			} else {
				task.OnFinish()
			}
			w.waitGroup.Done()
		}
	}
}

type Task interface {
	Do(ctx context.Context) error
	OnFinish()
	OnError(error)
}

type SimpleTask struct {
	Task
	function func() error
}

func NewSimpleTask(function func() error) *SimpleTask {
	return &SimpleTask{
		function: function,
	}
}

func (t *SimpleTask) Do(ctx context.Context) error {
	return t.function()
}

func (t *SimpleTask) OnFinish() {
}

func (t *SimpleTask) OnError(err error) {
}

type AdvancedTask struct {
	function     func(...interface{}) error
	callback     func(...interface{})
	errorhandler func(error, ...interface{})
	params       []interface{}
}

func NewAdvancedTask(
	function func(...interface{}) error,
	callback func(...interface{}),
	errorhandler func(error, ...interface{}),
	params []interface{}) *AdvancedTask {
	return &AdvancedTask{
		params:       params,
		function:     function,
		callback:     callback,
		errorhandler: errorhandler,
	}
}

func (t *AdvancedTask) Do() error {
	return t.function(t.params...)
}

func (t *AdvancedTask) OnFinish() {
	t.callback(t.params...)
}

func (t *AdvancedTask) OnError(err error) {
	t.errorhandler(err, t.params...)
}
