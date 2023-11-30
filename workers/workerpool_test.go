package workers

import (
	"context"
	"fmt"
	"gotils/config"
	log "gotils/logger"
	"math/rand"
	"testing"
	"time"
)

var (
	workerTestConfig = map[string]interface{}{
		"WORKERS": 10,
		"LOGGER": map[string]interface{}{
			"LEVEL":  "DEBUG",
			"PREFIX": "WORKERPOOL-TEST",
		},
	}
	testTask = NewSimpleTask(func() error {
		<-time.After(time.Duration(rand.Intn(30)) * time.Millisecond)
		fmt.Println("Hello World")
		return nil
	})
)

func TestSimpleTaskExecution(t *testing.T) {
	testCtx := context.Background()
	logger, err := log.NewLogger(config.NewConfigWithInitialValues(workerTestConfig))
	if err != nil {
		t.Log(err)
		t.Error("Logger is not creating correctly")
		t.FailNow()
	}
	workerPool, err := NewWorkerPool(testCtx, workerTestConfig["WORKERS"].(int), logger)
	if err != nil {
		t.Log(err)
		t.Error("WorkerPool is not creating correctly")
		t.FailNow()
	}

	for i := 0; i < 10; i++ {
		workerPool.Add(testTask)
	}

	// allow the workers to finish
	<-time.After(1 * time.Second)

	// workerPool.Stop()
}
