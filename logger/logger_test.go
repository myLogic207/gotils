package logger

import (
	"context"
	config "gotils/config"
	"os"
	"path"
	"strings"
	"testing"
)

func TestLogToFile(t *testing.T) {
	logFile := "test-logs/test.log"
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current working directory: %v", err)
	}
	logFile = path.Join(cwd, logFile)

	logMessage := "This is a test log message"
	prefix := "test"
	loggerConfig := config.NewConfigWithInitialValues(context.Background(), map[string]interface{}{
		"PREFIX": prefix,
		"FOLDER": "test-logs",
		"LEVEL":  "DEBUG",
		"WRITERS": map[string]interface{}{
			"FILE": true,
		},
	})

	if err := RegisterLogger(context.TODO(), loggerConfig); err != nil {
		panic(err)
	}

	file, err := os.OpenFile(logFile, os.O_RDONLY, 0644)
	if err != nil {
		t.Errorf("Error opening log file: %v", err)
	}
	defer file.Close()

	Info(prefix, logMessage)

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
		t.Log(string(buf))
		t.Error()
	}

	// remove log file
	if err := os.Remove(logFile); err != nil {
		t.Errorf("Error removing log file: %v", err)
	}
}

func TestLogFileRotate(t *testing.T) {
	testCtx := context.Background()
	loggerConf := config.NewConfigWithInitialValues(testCtx, map[string]interface{}{
		"PREFIX":       "test",
		"FOLDER":       "test-logs",
		"SUFFIX":       ".log",
		"LEVEL":        "DEBUG",
		"ROTATING":     true,
		"ROTATEFORMAT": "$prefix.test.$suffix",
		"WRITERS": map[string]interface{}{
			"FILE": true,
		},
	})

	if err := RegisterLogger(testCtx, loggerConf); err != nil {
		t.Errorf("Error registering logger: %v", err)
		t.FailNow()
	}
	pwd, _ := os.Getwd()
	filePath := path.Join(pwd, "test-logs", "test.log")

	// test if log file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Log file does not exist: %v", err)
	}

	t.Log("All good so far, closing logger and awaiting rotation")
	Cleanup()

	newFileName := path.Join(pwd, "test-logs", "test.test.log")
	// test if file is rotate
	if _, err := os.Stat(newFileName); err != nil && !os.IsNotExist(err) {
		t.Errorf("Log file was not rotated: %v", err)
	}

	// remove log file
	if err := os.Remove(newFileName); err != nil {
		t.Errorf("Error removing log file: %v", err)
	}
}
