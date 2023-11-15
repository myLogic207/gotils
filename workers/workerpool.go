package workers

import (
	"context"
	"fmt"
	"gotils/config"
	"gotils/logger"
	"sync"
)

var (
	defaultWorkerConfig = map[string]interface{}{
		"LOGGER": map[string]interface{}{
			"PREFIX": "WORKERPOOL",
		},
		"WORKERS": 3,
		"BUFFER": map[string]interface{}{
			"SIZE": 100,
			"MODE": 1, // 1 = queue, 0 = stack
			"LOGGER": map[string]interface{}{
				"PREFIX": "WORKERPOOL-BUFFER",
			},
		},
	}
)

type WorkerPool struct {
	ctx       context.Context
	prefix    string
	workers   int
	waitGroup sync.WaitGroup
	tasks     chan Task
	start     sync.Once
	stop      sync.Once
	quit      chan bool
}

func NewWorkerPool(ctx context.Context, options *config.Config) (*WorkerPool, error) {
	config := config.NewConfigWithInitialValues(ctx, defaultWorkerConfig)
	config.Merge(options, true)
	workers, err := config.GetInt("WORKERS")
	if err != nil {
		return nil, err
	}
	loggerConfig, err := config.GetConfig("LOGGER")
	if err != nil {
		return nil, err
	}
	prefix, err := loggerConfig.GetString("PREFIX")
	if err != nil {
		return nil, err
	}

	if err := logger.RegisterLogger(ctx, loggerConfig); err != nil {
		return nil, err
	}

	return &WorkerPool{
		ctx:       ctx,
		prefix:    prefix,
		workers:   workers,
		waitGroup: sync.WaitGroup{},
		tasks:     make(chan Task, workers),
		quit:      make(chan bool, workers),
		start:     sync.Once{},
		stop:      sync.Once{},
	}, nil
}

func (w *WorkerPool) Start() {
	w.start.Do(func() {
		logger.Info(w.prefix, "Starting worker pool")
		go w.cancelHook()
		for i := 0; i < w.workers; i++ {
			go w.worker()
		}
	})
}

func (w *WorkerPool) cancelHook() {
	<-w.ctx.Done()
	logger.Debug(w.prefix, "Shutting down worker pool")
	w.Stop()
}

func (w *WorkerPool) Stop() {
	w.stop.Do(func() {
		logger.Info(w.prefix, "Stopping worker pool")
		close(w.quit)

		logger.Debug(w.prefix, "Awaiting worker pool to finish")
		logger.Debug(w.prefix, fmt.Sprintf("Tasks in queue: %d", len(w.tasks)))
		w.waitGroup.Wait()
	})
}

func (w *WorkerPool) Add(task Task) {
	select {
	case <-w.quit:
		logger.Debug(w.prefix, "Worker pool is closed, dropping task")
	case <-w.ctx.Done():
		logger.Debug(w.prefix, "Worker pool is closed, dropping task")
	case w.tasks <- task:
		logger.Debug(w.prefix, "Adding task to worker pool")
		w.waitGroup.Add(1)
	}
}

func (w *WorkerPool) worker() {
	for {
		select {
		case <-w.quit:
			logger.Debug(w.prefix, "Worker received quit signal")
			return
		case <-w.ctx.Done():
			logger.Debug(w.prefix, "Worker received context cancel signal")
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			if err := task.Do(); err != nil {
				logger.Error(w.prefix, err.Error())
				task.OnError(err)
			} else {
				task.OnFinish()
			}
			w.waitGroup.Done()
		}
	}
}

type Task interface {
	Do() error
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

func (t *SimpleTask) Do() error {
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
