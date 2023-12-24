package logger

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/myLogic207/gotils/config"
)

func TestLogFileRotate(t *testing.T) {
	logfileConf, err := config.WithInitialValues(context.TODO(), map[string]interface{}{
		"PREFIX":       "test",
		"ACTIVE":       true,
		"FOLDER":       "test-logs",
		"SUFFIX":       "log",
		"ROTATING":     true,
		"ROTATEFORMAT": "$prefix.test.$suffix",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		// remove log file folder
		if err := os.RemoveAll("test-logs"); err != nil {
			t.Errorf("Error removing log file: %v", err)
		}
	})

	logFile, err := NewLogFile(context.Background(), logfileConf)
	if err != nil {
		t.Fatalf("Error creating logger: %v", err)
	}
	logFile.Write([]byte("This is a test log message"))

	pwd, _ := os.Getwd()
	filePath := path.Join(pwd, "test-logs", "test.log")

	// test if log file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Log file does not exist: %v", err)
	}

	t.Log("All good so far, closing logger and awaiting rotation")

	if err := logFile.Close(context.Background()); err != nil {
		t.Fatalf("Error shutting down logger: %v", err)
	}

	newFileName := path.Join(pwd, "test-logs", "test.test.log")
	// test if file is rotate
	if _, err := os.Stat(newFileName); err != nil && os.IsNotExist(err) {
		t.Fatalf("Log file was not rotated: %v", err)
	}
}
