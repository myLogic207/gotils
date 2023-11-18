package logger

import (
	config "gotils/config"
	"os"
	"path"
	"strings"
	"testing"
	"time"
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
	loggerConfig := config.NewConfigWithInitialValues(map[string]interface{}{
		"PREFIX":       prefix,
		"PREFIXLENGTH": 6,
		"LEVEL":        "DEBUG",
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

	logger, err := NewLogger(loggerConfig)
	if err != nil || logger == nil || logger.logger == nil {
		t.Errorf("Error registering logger:\n %v", err)
		t.FailNow()
	}

	// t.Cleanup(func() {
	// 	// remove log file folder
	// 	if err := os.RemoveAll(logDir); err != nil {
	// 		t.Errorf("Error removing log file: %v", err)
	// 	}
	// })

	logger.Info(logMessage)

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

func TestLogFileRotate(t *testing.T) {
	loggerConf := config.NewConfigWithInitialValues(map[string]interface{}{
		"PREFIX":       "test",
		"PREFIXLENGTH": 6,
		"LEVEL":        "DEBUG",
		"WRITERS": map[string]interface{}{
			"FILE": map[string]interface{}{
				"ACTIVE":       true,
				"PREFIX":       "test",
				"FOLDER":       "test-logs",
				"SUFFIX":       "log",
				"ROTATING":     true,
				"ROTATEFORMAT": "$prefix.test.$suffix",
			},
		},
	})

	t.Cleanup(func() {
		// remove log file folder
		if err := os.RemoveAll("test-logs"); err != nil {
			t.Errorf("Error removing log file: %v", err)
		}
	})

	logger, err := NewLogger(loggerConf)
	if err != nil {
		t.Errorf("Error registering logger: %v", err)
		t.FailNow()
	}
	logger.Info("This is a test log message")
	pwd, _ := os.Getwd()
	filePath := path.Join(pwd, "test-logs", "test.log")

	// test if log file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Log file does not exist: %v", err)
	}

	t.Log("All good so far, closing logger and awaiting rotation")

	newFileName := path.Join(pwd, "test-logs", "test.test.log")
	// test if file is rotate
	if _, err := os.Stat(newFileName); err != nil && os.IsNotExist(err) {
		t.Errorf("Log file was not rotated: %v", err)
		t.FailNow()
	}
}
