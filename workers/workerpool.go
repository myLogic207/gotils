package workers

import (
	"context"
	"sync"

	log "github.com/myLogic207/gotils/logger"
)

type contextKey string

var workerId = contextKey("worker_id")

type WorkerPool struct {
	logger    log.Logger
	waitGroup sync.WaitGroup
	quit      chan bool
	tasks     chan Task
}

func NewWorkerPool(ctx context.Context, size int, logger log.Logger) (*WorkerPool, WorkerError) {
	pool := &WorkerPool{
		logger:    logger,
		waitGroup: sync.WaitGroup{},
		quit:      make(chan bool),
		tasks:     make(chan Task, size),
	}
	pool.logger.Debug(ctx, "Creating worker pool with %d workers", size)
	for i := 0; i < size; i++ {
		workerCtx := context.WithValue(ctx, workerId, i)
		go pool.worker(workerCtx)
	}
	go pool.stopOnCancel(ctx)
	pool.logger.Debug(ctx, "Worker pool initialized")
	return pool, nil
}

func (w *WorkerPool) stopOnCancel(ctx context.Context) {
	<-ctx.Done()
	w.logger.Warn(ctx, "Stopping worker pool")
	w.logger.Info(ctx, "Awaiting worker pool to finish")
	close(w.tasks)
	w.logger.Debug(ctx, "Tasks in queue: %d", len(w.tasks))
	w.waitGroup.Wait()
	w.logger.Info(ctx, "Worker pool finished")
}

func (w *WorkerPool) Add(ctx context.Context, task Task) {
	w.logger.Debug(ctx, "Received, task. Tasks in queue: %d", len(w.tasks))
	if len(w.tasks)-1 >= cap(w.tasks) {
		w.logger.Warn(ctx, "Worker pool is full or closed, dropping task")
		return
	}
	w.tasks <- task
	w.logger.Info(ctx, "Adding task to worker pool")
	w.waitGroup.Add(1)
}

func (w *WorkerPool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Debug(ctx, "Worker %d received quit signal", ctx.Value(workerId).(int))
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			taskCtx := context.WithValue(ctx, workerId, ctx.Value(workerId).(int))
			if err := task.Do(taskCtx); err != nil {
				w.logger.Error(taskCtx, err.Error())
				task.OnError(taskCtx, err)
			} else {
				task.OnFinish(taskCtx)
			}
			w.waitGroup.Done()
		}
	}
}

type Task interface {
	Do(context.Context) error
	OnFinish(context.Context)
	OnError(context.Context, error)
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

func (t *SimpleTask) OnFinish(ctx context.Context) {
}

func (t *SimpleTask) OnError(ctx context.Context, err error) {
}

type AdvancedTask struct {
	function     func(context.Context, ...interface{}) error
	callback     func(context.Context, ...interface{})
	errorhandler func(context.Context, error, ...interface{})
	params       []interface{}
}

func NewAdvancedTask(
	function func(context.Context, ...interface{}) error,
	callback func(context.Context, ...interface{}),
	errorhandler func(context.Context, error, ...interface{}),
	params []interface{}) *AdvancedTask {
	return &AdvancedTask{
		params:       params,
		function:     function,
		callback:     callback,
		errorhandler: errorhandler,
	}
}

func (t *AdvancedTask) Do(ctx context.Context) error {
	return t.function(ctx, t.params...)
}

func (t *AdvancedTask) OnFinish(ctx context.Context) {
	t.callback(ctx, t.params...)
}

func (t *AdvancedTask) OnError(ctx context.Context, err error) {
	t.errorhandler(ctx, err, t.params...)
}
