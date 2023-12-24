package logger

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/myLogic207/gotils/config"
)

func TestLogToFile(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current working directory: %v", err)
	}

	logDir := "test-logs"
	logFile := "test.log"
	logFile = path.Join(cwd, logDir, logFile)
	logMessage := "This is a test log message"
	prefix := "test"
	loggerConfig, err := config.WithInitialValues(context.TODO(), map[string]interface{}{
		"PREFIX":      prefix,
		"COLUMLENGTH": 6,
		"LEVEL":       "DEBUG",
		"WRITERS": map[string]interface{}{
			"STDOUT": true,
			"FILE": map[string]interface{}{
				"PREFIX":   prefix,
				"ACTIVE":   true,
				"FILENAME": "$prefix.$suffix",
				"FOLDER":   logDir,
				"SUFFIX":   "log",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	logger, err := Init(context.TODO(), loggerConfig)
	if err != nil || logger == nil {
		t.Errorf("Error registering logger:\n %v", err)
		t.FailNow()
	}

	t.Cleanup(func() {
		// remove log file folder
		if err := os.RemoveAll(logDir); err != nil {
			t.Errorf("Error removing log file: %v", err)
		}
	})

	logger.Info(context.TODO(), logMessage)

	<-time.After(100 * time.Millisecond)

	if file, err := os.OpenFile(logFile, os.O_RDONLY, 0644); err != nil {
		t.Errorf("Error opening log file: %v", err)
	} else {
		defer file.Close()
		// check if the log file contains the log message
		buf := make([]byte, 1024)
		n, err := file.Read(buf)
		if err != nil {
			t.Errorf("Error reading log file: %v", err)
		}
		if n == 0 {
			t.Errorf("Log file is empty")
		}
		if strings.Contains(string(buf), logMessage) == false {
			t.Log("Log file does not contain log message:")
			t.Log(logMessage)
			t.Log("Log file contents:")
			t.Log(string(buf))
			t.Error()
		}
	}
}
