package workers

import (
	"context"
	"fmt"
	"gotils/config"
	log "gotils/logger"
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
	logger    *log.Logger
	config    *config.Config
	waitGroup sync.WaitGroup
	start     sync.Once
	stop      sync.Once
	quit      chan bool
	tasks     chan Task
}

func NewWorkerPool(options *config.Config) (*WorkerPool, WorkerError) {
	config := config.NewConfigWithInitialValues(defaultWorkerConfig)
	if err := config.Merge(options, true); err != nil {
		return nil, &InitError{nested: err}
	}
	if err := config.CompareDefault(defaultWorkerConfig); err != nil {
		return nil, &InitError{nested: err}
	}

	loggerConfig, _ := config.GetConfig("LOGGER")

	logger, err := log.NewLogger(loggerConfig)
	if err != nil {
		return nil, &InitError{nested: err}
	}

	return &WorkerPool{
		logger:    logger,
		config:    config,
		waitGroup: sync.WaitGroup{},
		start:     sync.Once{},
		stop:      sync.Once{},
	}, nil
}

func (w *WorkerPool) Start(ctx context.Context) {
	w.start.Do(func() {
		w.logger.Info("Starting worker pool")
		workers, _ := w.config.GetInt("WORKERS")
		w.quit = make(chan bool)
		w.tasks = make(chan Task, workers*2)
		go w.cancelHook(ctx)
		for i := 0; i < workers; i++ {
			go w.worker()
		}
	})
}

func (w *WorkerPool) cancelHook(ctx context.Context) {
	<-ctx.Done()
	w.logger.Debug("Shutting down worker pool")
	w.Stop()
}

func (w *WorkerPool) Stop() {
	w.stop.Do(func() {
		w.logger.Info("Stopping worker pool")
		close(w.quit)

		w.logger.Debug("Awaiting worker pool to finish")
		w.logger.Debug(fmt.Sprintf("Tasks in queue: %d", len(w.tasks)))
		w.waitGroup.Wait()
	})
}

func (w *WorkerPool) Add(task Task) {
	select {
	case <-w.quit:
		w.logger.Debug("Worker pool is closed, dropping task")
	case w.tasks <- task:
		w.logger.Debug("Adding task to worker pool")
		w.waitGroup.Add(1)
	}
}

func (w *WorkerPool) worker() {
	for {
		select {
		case <-w.quit:
			w.logger.Debug("Worker received quit signal")
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			if err := task.Do(); err != nil {
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
