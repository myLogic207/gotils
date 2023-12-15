package workers

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/myLogic207/gotils/config"
	log "github.com/myLogic207/gotils/logger"
)

var (
	workerTestConfig = map[string]interface{}{
		"WORKERS": 10,
		"LOGGER": map[string]interface{}{
			"LEVEL":  "DEBUG",
			"PREFIX": "WORKERPOOL-TEST",
		},
	}
	testTask = NewTask(func(context.Context) error {
		<-time.After(time.Duration(rand.Intn(100)) * time.Millisecond)
		fmt.Println("Hello World")
		return nil
	}, func(ctx context.Context, err error) {
		fmt.Println("Task finished")
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
	})
)

func TestSimpleTaskExecution(t *testing.T) {
	testCtx, cancel := context.WithCancel(context.Background())
	logger, err := log.NewLogger(config.NewWithInitialValues(workerTestConfig))
	if err != nil {
		t.Log(err)
		t.Error("Logger is not creating correctly")
		t.FailNow()
	}
	workerPool, err := InitPool(testCtx, workerTestConfig["WORKERS"].(int), logger)
	if err != nil {
		t.Log(err)
		t.Error("WorkerPool is not creating correctly")
		t.FailNow()
	}

	for i := 0; i < 10; i++ {
		workerPool.Add(testCtx, testTask)
	}

	// Wait for some tasks to finish
	// <-time.After(50 * time.Millisecond)

	workerPool.Stop(testCtx)
	cancel()
}
