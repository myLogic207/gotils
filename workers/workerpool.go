package workers

import (
	"context"
	"sync"

	log "github.com/myLogic207/gotils/logger"
)

type contextKey string

type WorkerPool interface {
	Add(context.Context, Task) error
	Execute(context.Context, Task)
	Stop(context.Context)
}

var workerId = contextKey("worker_id")

type WorkerPoolImpl struct {
	logger    log.Logger
	waitGroup sync.WaitGroup
	tasks     chan Task
}

func InitPool(ctx context.Context, size int, logger log.Logger) (WorkerPool, error) {
	pool := &WorkerPoolImpl{
		logger:    logger,
		waitGroup: sync.WaitGroup{},
		tasks:     make(chan Task, size),
	}
	pool.logger.Debug(ctx, "Worker pool created")
	pool.logger.Debug(ctx, "Worker pool size: %d", size)
	pool.Start(ctx, size)
	return pool, nil
}

func (w *WorkerPoolImpl) Start(ctx context.Context, size int) {
	w.logger.Info(ctx, "Starting worker pool")
	for i := 0; i < size; i++ {
		go w.worker(context.WithValue(ctx, workerId, i))
	}
	w.logger.Info(ctx, "Worker pool started")
}

func (w *WorkerPoolImpl) Stop(ctx context.Context) {
	w.logger.Warn(ctx, "Stopping worker pool")
	close(w.tasks)
	w.logger.Info(ctx, "Awaiting worker pool to finish")
	w.logger.Debug(ctx, "Tasks in queue: %d", len(w.tasks))
	w.waitGroup.Wait()
	w.logger.Info(ctx, "Worker pool finished")
}

func (w *WorkerPoolImpl) Add(ctx context.Context, task Task) error {
	w.logger.Debug(ctx, "Received, task. Tasks in queue: %d", len(w.tasks))
	if len(w.tasks)-1 >= cap(w.tasks) {
		w.logger.Warn(ctx, "Worker pool is full or closed, dropping task")
		return ErrWorkerPoolFull
	}
	w.tasks <- task
	w.logger.Info(ctx, "Added task to worker pool")
	return nil
}

func (w *WorkerPoolImpl) worker(ctx context.Context) {
	w.logger.Debug(ctx, "Worker %d started", ctx.Value(workerId).(int))
	defer w.logger.Debug(ctx, "Worker %d stopped", ctx.Value(workerId).(int))

	for {
		select {
		case <-ctx.Done():
			w.logger.Debug(ctx, "Worker %d received quit signal", ctx.Value(workerId).(int))
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			w.waitGroup.Add(1)
			w.Execute(ctx, task)
			w.waitGroup.Done()
		}
	}
}

func (w *WorkerPoolImpl) Execute(ctx context.Context, task Task) {
	taskCtx, cancel := context.WithCancel(ctx)
	w.logger.Debug(taskCtx, "Worker %d received task", ctx.Value(workerId).(int))
	err := task.Do(taskCtx)
	if err != nil {
		w.logger.Error(taskCtx, err.Error())
	}
	task.OnFinish(taskCtx, err)
	cancel()
}
