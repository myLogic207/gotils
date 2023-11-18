package workers

import (
	"context"
	"fmt"
	"gotils/config"
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
	testConfig := config.NewConfigWithInitialValues(workerTestConfig)
	workerPool, err := NewWorkerPool(testConfig)
	if err != nil {
		t.Log(err)
		t.Error("WorkerPool is not creating correctly")
		t.FailNow()
	}
	workerPool.Start(testCtx)

	for i := 0; i < 10; i++ {
		workerPool.Add(testTask)
	}

	workerPool.Stop()
}
