package logger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/myLogic207/gotils/config"
)

var (
	ErrOpenLogFile        = errors.New("error opening log file")
	ErrRotateFile         = errors.New("error rotating log file")
	ErrFormattingFilename = errors.New("error formatting filename")
	defaultLogFileConfig  = map[string]any{
		"PREFIX":       "service",
		"ACTIVE":       "true",
		"ROTATING":     "true",
		"ROTATEFORMAT": "$prefix.$date.$time.$suffix",
		"FOLDER":       "/var/log",
		"SUFFIX":       "log",
		"FILENAME":     "$prefix.$suffix",
	}
)

type LogFile struct {
	file   *os.File
	config *config.Config
}

func NewLogFile(ctx context.Context, options *config.Config) (*LogFile, error) {
	cfg, err := config.WithInitialValuesAndOptions(ctx, defaultLogFileConfig, options)
	if err != nil {
		return nil, errors.Join(ErrOpenLogFile, err)
	}

	writer := &LogFile{
		config: cfg,
	}

	if err := writer.generateLogFile(ctx); err != nil {
		return nil, errors.Join(ErrOpenLogFile, err)
	}
	println("Log file created")
	println(writer.file.Name())
	// go writer.contextClose(ctx)

	// log.Printf("Creating log writer, to console %t, to file %t (rotating: %t)", console, file, rotating)
	return writer, nil
}

// func (l *LogFile) contextClose(ctx context.Context) {
// 	<-ctx.Done()
// 	if err := l.Close(); err != nil {
// 		println(err.Error())
// 	}
// }

func (l *LogFile) generateLogFile(ctx context.Context) error {
	rawFilename, _ := l.config.Get(ctx, "FILENAME")
	formattedPath := l.formatFilename(ctx, rawFilename)
	// check if path is valid
	if formattedPath == "" {
		return errors.New("path is empty")
	}
	folder, _ := l.config.Get(ctx, "FOLDER")
	formattedPath = path.Join(folder, formattedPath)

	if !path.IsAbs(formattedPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		formattedPath = path.Join(cwd, formattedPath)
	}

	dir := path.Dir(formattedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := l.prepExisting(ctx, formattedPath); err != nil {
		return err
	}
	println("Opening log file", formattedPath)
	var err error
	l.file, err = os.OpenFile(formattedPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	return err
}

func (l *LogFile) prepExisting(ctx context.Context, filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil && os.IsNotExist(err) {
		// println("File does not exist, all good")
		return nil
	} else if err != nil {
		println("Error checking file", err.Error())
		return err
	} else if info.Size() < 1 {
		println("File is empty, removing")
		return os.Remove(filepath)
	}
	rotating, _ := l.config.Get(ctx, "ROTATING")
	if ok, err := strconv.ParseBool(rotating); err != nil || !ok {
		println("file exists, not empty but allowed to rotate away")
		if err := l.rotate(ctx, info.Name(), false); err != nil {
			// just append anyways...
			return nil
		}
	}
	return nil
}

func (l *LogFile) Write(p []byte) (n int, err error) {
	return l.file.Write(p)
}

func (l *LogFile) Close(ctx context.Context) error {
	if err := l.file.Sync(); err != nil {
		return err
	}

	rotating, _ := l.config.Get(ctx, "ROTATING")
	if ok, err := strconv.ParseBool(rotating); err != nil || !ok {
		return l.file.Close()
	}

	return l.rotate(ctx, l.file.Name(), false)
}

func (l *LogFile) Sync() error {
	return l.file.Sync()
}

// func (l *LogFile) UpdateConfig(ctx context.Context, options config.Config) error {
// 	if err := l.config.Merge(ctx, options, true); err != nil {
// 		return err
// 	}
// 	if err := l.config.CompareMap(ctx, defaultLogConfig, false); err != nil {
// 		return err
// 	}
// 	return nil
// }

func (l *LogFile) RotateFile(ctx context.Context) error {
	if l.file == nil {
		return errors.Join(ErrRotateFile, ErrFileNotActive)
	}

	return l.rotate(ctx, l.file.Name(), true)
}

func (l *LogFile) rotate(ctx context.Context, oldName string, regenerate bool) error {
	if err := l.file.Close(); l.file != nil && err != nil {
		return errors.Join(ErrRotateFile, err)
	}
	// assemble path from pwd, folder and filename
	oldNameFull, err := l.assemblePath(ctx, oldName)
	if err != nil {
		return errors.Join(ErrRotateFile, err)
	}

	rotateFormat, _ := l.config.Get(ctx, "ROTATEFORMAT")
	rotateName, err := l.assemblePath(ctx, rotateFormat)
	if err != nil {
		return errors.Join(ErrRotateFile, err)
	}

	if stat, err := os.Stat(rotateName); err != nil && !os.IsNotExist(err) {
		return errors.Join(ErrRotateFile, err)
	} else if err == nil && stat.Size() > 0 {
		suffix, _ := l.config.Get(ctx, "SUFFIX")
		if newName := generateNextFilename(ctx, rotateName, suffix, 1000); newName != "" {
			// file exists, try next
			rotateName = newName
		} else {
			return errors.Join(ErrRotateFile, ErrFormattingFilename)
		}
	}
	if err := os.Rename(oldNameFull, rotateName); err != nil {
		return err
	}

	println("Rotated log file", oldNameFull, "to", rotateName)
	if regenerate {
		return l.generateLogFile(ctx)
	}
	return nil
}

func generateNextFilename(ctx context.Context, baseName string, suffix string, iterations int) string {
	newName := baseName
	for counter := 0; counter < iterations; counter++ {
		if err := ctx.Err(); err != nil {
			// context cancelled
			return ""
		}

		var intermediateName string
		if index := strings.LastIndex(newName, suffix); index > -1 {
			intermediateName = newName[:index-1]
		} else {
			parts := strings.Split(newName, ".")
			intermediateName = strings.Join(parts[:len(parts)-1], ".")
			suffix = parts[len(parts)-1]
		}
		newName := fmt.Sprintf("%s.%d.%s", intermediateName, counter, suffix)

		if _, err := os.Stat(newName); err != nil {
			if os.IsNotExist(err) {
				// file does not exist, return
				return newName
			} else {
				// some other error
				return ""
			}
		}
		println("File exists, trying next", newName)
	}
	return ""
}

func (l *LogFile) formatFilename(ctx context.Context, format string) string {
	buffer := strings.Builder{}
	for _, part := range strings.Split(format, ".") {
		if err := ctx.Err(); err != nil {
			// context cancelled, return what we have?
			return buffer.String()
		}

		if part == "" {
			continue
		} else if part[0] != '$' {
			buffer.WriteString(part)
			buffer.WriteString(".")
			continue
		}

		switch part {
		case "$prefix":
			prefix, _ := l.config.Get(ctx, "PREFIX")
			prefix = strings.ReplaceAll(prefix, " ", "_")
			prefix = strings.ToLower(prefix)
			buffer.WriteString(prefix)
		case "$suffix":
			suffix, _ := l.config.Get(ctx, "SUFFIX")
			buffer.WriteString(suffix)
		case "$date":
			currentDate := time.Now().Format(time.RFC3339)
			formattedDate := strings.Split(currentDate, "T")[0]
			buffer.WriteString(formattedDate)
		case "$time":
			currentTime := time.Now().Format(time.Kitchen)
			formattedTime := strings.ReplaceAll(currentTime, ":", "_")
			buffer.WriteString(formattedTime)
		default:
			buffer.WriteString(part)
		}
		buffer.WriteString(".")
	}
	return strings.TrimSuffix(buffer.String(), ".")
}

func (l *LogFile) assemblePath(ctx context.Context, raw string) (string, error) {
	// check if template name, always base
	if strings.Contains(raw, "$") {
		raw = l.formatFilename(ctx, raw)
	} else {
		// make sure raw is basename
		raw = path.Base(raw)
	}
	fullPath, _ := l.config.Get(ctx, "FOLDER")
	fullPath = path.Join(fullPath, raw)
	// make sure path is absolute
	if !path.IsAbs(fullPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		fullPath = path.Join(cwd, fullPath)
	}
	return fullPath, nil
}
