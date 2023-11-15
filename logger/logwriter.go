package logger

import (
	"bufio"
	"errors"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

var (
	ErrOpenLogFile        = errors.New("error opening log file")
	ErrFormattingFilename = errors.New("error formatting filename")
)

type logWriter struct {
	console      bool
	rotating     bool
	rotateFormat string
	fileWriter   *bufio.Writer
	fileHandle   *os.File
}

func newLogWriter(console bool, file bool, filepath string, rotating bool, rotateFormat string) (*logWriter, error) {
	writer := logWriter{
		console:      console,
		rotating:     rotating,
		rotateFormat: rotateFormat,
	}

	if !console && !file {
		return nil, errors.New("no output specified")
	}

	if !file {
		return &writer, nil
	}

	var err error
	writer.fileHandle, err = os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, ErrOpenLogFile
	}
	writer.fileWriter = bufio.NewWriter(writer.fileHandle)
	// log.Printf("Creating log writer, to console %t, to file %t (rotating: %t)", console, file, rotating)

	return &writer, nil
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	if l.console {
		println(string(p))
	}

	if l.fileWriter != nil {
		if _, err := l.fileWriter.Write(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (l *logWriter) Close() error {
	if l.fileWriter == nil {
		return nil
	}

	err := l.fileWriter.Flush()
	if err != nil {
		return err
	}

	err = l.fileHandle.Close()
	if err != nil {
		return err
	}

	if !l.rotating {
		return nil
	}
	// move current active file to formatted filename
	currentBaseName := path.Base(l.fileHandle.Name())
	newFileBaseName := formatRotatingFilename(l.rotateFormat, currentBaseName)
	if newFileBaseName == "" {
		return ErrFormattingFilename
	}
	directory := path.Dir(l.fileHandle.Name())
	newFileName := path.Join(directory, newFileBaseName)

	// check if file already exists
	if _, err := os.Stat(newFileName); !os.IsNotExist(err) {
		// file already exists, delete it
		log.Println("Rotating file already exists, deleting it")
		os.Remove(newFileName)
	}

	if err := os.Rename(l.fileHandle.Name(), newFileName); err != nil {
		return err
	}
	return nil
}

func formatRotatingFilename(format string, basename string) string {
	parts := strings.SplitN(basename, ".", 2)
	prefix := strings.Join(parts[:len(parts)-1], "-")
	suffix := parts[len(parts)-1]
	now := time.Now()
	date := now.Format("2006-01-02")
	timestamp := now.Format("15-04-05")

	buffer := strings.Builder{}

	for _, part := range strings.Split(format, ".") {
		switch part {
		case "$prefix":
			buffer.WriteString(prefix)
		case "$suffix":
			buffer.WriteString(suffix)
		case "$date":
			buffer.WriteString(date)
		case "$time":
			buffer.WriteString(timestamp)
		default:
			buffer.WriteString(part)
		}
		buffer.WriteString(".")
	}
	return strings.TrimSuffix(buffer.String(), ".")
}
