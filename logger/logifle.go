package logger

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/myLogic207/gotils/config"
)

var (
	ErrOpenLogFile        = errors.New("error opening log file")
	ErrRotateFile         = errors.New("error rotating log file")
	ErrFormattingFilename = errors.New("error formatting filename")
	defaultLogFileConfig  = map[string]interface{}{
		"PREFIX":       "service",
		"ACTIVE":       true,
		"ROTATING":     true,
		"ROTATEFORMAT": "$prefix.$date.$time.$suffix",
		"FOLDER":       "/var/log",
		"SUFFIX":       "log",
		"FILENAME":     "$prefix.$suffix",
	}
)

type LogFile struct {
	file   *os.File
	config config.Config
}

func NewLogFile(options config.Config) (*LogFile, error) {
	cfg := config.NewConfigWithInitialValues(defaultLogFileConfig)
	if err := cfg.Merge(options, true); err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}
	if err := cfg.CompareDefault(defaultLogFileConfig); err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}

	writer := &LogFile{
		config: cfg,
	}

	if err := writer.generateLogFile(); err != nil {
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

func (l *LogFile) generateLogFile() error {
	rawFilename, _ := l.config.GetString("FILENAME")
	formattedPath := l.formatFilename(rawFilename)
	// check if path is valid
	if formattedPath == "" {
		return errors.New("path is empty")
	}
	println("Opening log file", formattedPath)
	folder, _ := l.config.GetString("FOLDER")
	formattedPath = path.Join(folder, formattedPath)
	println("Opening log file", formattedPath)

	if !path.IsAbs(formattedPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		formattedPath = path.Join(cwd, formattedPath)
	}
	println("Opening log file", formattedPath)

	dir := path.Dir(formattedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := l.prepExisting(formattedPath); err != nil {
		return err
	}
	println("Opening log file", formattedPath)
	var err error
	l.file, err = os.OpenFile(formattedPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	return err
}

func (l *LogFile) prepExisting(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil && os.IsNotExist(err) {
		println("File does not exist, all good")
		return nil
	} else if err != nil {
		println("Error checking file", err.Error())
		return err
	} else if info.Size() < 1 {
		println("File is empty, removing")
		return os.Remove(filepath)
	} else if rotating, _ := l.config.GetBool("ROTATING"); rotating {
		println("file exists, not empty but can rotate away")
		if err := l.rotate(info.Name(), false); err != nil {
			// just append anyways...
			return nil
		}
	}
	return nil
}

func (l *LogFile) Write(p []byte) (n int, err error) {
	return l.file.Write(p)
}

func (l *LogFile) Close() error {
	if err := l.file.Sync(); err != nil {
		return err
	}

	if rotating, _ := l.config.GetBool("ROTATING"); !rotating {
		return l.file.Close()
	}

	return l.rotate(l.file.Name(), false)
}

func (l *LogFile) Sync() error {
	return l.file.Sync()
}

func (l *LogFile) UpdateConfig(options config.Config) error {
	if err := l.config.Merge(options, true); err != nil {
		return err
	}
	if err := l.config.CompareDefault(defaultLogConfig); err != nil {
		return err
	}
	return nil
}

func (l *LogFile) RotateFile() error {
	if l.file == nil {
		return errors.Join(ErrRotateFile, ErrFileNotActive)
	}

	return l.rotate(l.file.Name(), true)
}

func (l *LogFile) rotate(oldName string, regenerate bool) error {
	if err := l.file.Close(); l.file != nil && err != nil {
		return errors.Join(ErrRotateFile, err)
	}
	// assemble path from pwd, folder and filename
	oldNameFull, err := l.assemblePath(oldName)
	if err != nil {
		return errors.Join(ErrRotateFile, err)
	}

	rotateFormat, _ := l.config.GetString("ROTATEFORMAT")
	rotateName, err := l.assemblePath(rotateFormat)
	if err != nil {
		return errors.Join(ErrRotateFile, err)
	}

	if _, err := os.Stat(rotateName); err != nil && !os.IsNotExist(err) {
		return errors.Join(ErrRotateFile, ErrFileInUse, err)
	} else if err == nil {
		// if rotate file exists, append number
		for i := 0; i < 100; i++ {
			suffix, _ := l.config.GetString("SUFFIX")
			newName := ""
			if index := strings.LastIndex(rotateName, suffix); index > -1 {
				newName = rotateName[:index-1]
			} else {
				newName = rotateName
			}

			newName = fmt.Sprintf("%s.%d.%s", newName, i, suffix)
			if _, err := os.Stat(newName); err != nil && !os.IsNotExist(err) {
				return errors.Join(ErrRotateFile, ErrFileInUse, err)
			} else if err == nil {
				continue
			}
			rotateName = newName
			break
		}
	}

	if err := os.Rename(oldNameFull, rotateName); err != nil {
		return err
	}

	if regenerate {
		return l.generateLogFile()
	}
	return nil
}

func (l *LogFile) formatFilename(format string) string {
	buffer := strings.Builder{}
	for _, part := range strings.Split(format, ".") {
		switch part {
		case "$prefix":
			prefix, _ := l.config.GetString("PREFIX")
			prefix = strings.ReplaceAll(prefix, " ", "_")
			prefix = strings.ToLower(prefix)
			buffer.WriteString(prefix)
		case "$suffix":
			suffix, _ := l.config.GetString("SUFFIX")
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

func (l *LogFile) assemblePath(raw string) (string, error) {
	// check if template name, always base
	if strings.Contains(raw, "$") {
		raw = l.formatFilename(raw)
	} else {
		// make sure raw is basename
		raw = path.Base(raw)
	}
	fullPath, _ := l.config.GetString("FOLDER")
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
