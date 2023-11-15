package logger

import (
	"context"
	"errors"
	"fmt"
	"gotils/config"
	"log"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strings"
)

var (
	ErrPrefixNotSet  = errors.New("prefix not set")
	ErrLoggerExists  = errors.New("logger already exists")
	ErrFileInUse     = errors.New("log file is already in use")
	loggerList       = make(map[string]*loggerWrapper)
	defaultLogConfig = map[string]interface{}{
		"FLAGS":        "date,time,microseconds,utc,msgprefix",
		"PREFIXLENGTH": 16,
		"SUFFIX":       ".log",
		"FOLDER":       "/var/log",
		"REPLACECHAR":  "-",
		"LEVEL":        "DEBUG",
		"WRITERS": map[string]interface{}{
			"STDOUT": true,
			"FILE":   false,
		},
		"ROTATING":     false,
		"ROTATEFORMAT": "$prefix.$date.$time.$suffix",
		// "OUTFILE": path.Join(FOLDER, PREFIX+SUFFIX)
	}
)

type loggerWrapper struct {
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *log.Logger
	writer     *logWriter
	level      LogLevel
	terminated chan bool
}

func GetDefaultConfig(ctx context.Context) *config.Config {
	return config.NewConfigWithInitialValues(ctx, defaultLogConfig)
}

func SetDefaultConfig(cnf *config.Config) error {
	if err := cnf.CompareDefault(defaultLogConfig); err != nil {
		return err
	}

	configMap := cnf.GetMap()
	for key, value := range configMap {
		defaultLogConfig[key] = value
	}

	return nil
}

func RegisterLogger(ctx context.Context, configOptions *config.Config) error {
	cfg := config.NewConfigWithInitialValues(ctx, defaultLogConfig)
	if err := cfg.Merge(configOptions, true); err != nil {
		return err
	}

	prefix, err := cfg.GetString("PREFIX")
	if err != nil {
		return ErrPrefixNotSet
	}
	prefix = strings.ReplaceAll(prefix, " ", defaultLogConfig["REPLACECHAR"].(string))
	prefix = strings.ToUpper(prefix)

	if _, ok := loggerList[prefix]; ok {
		return ErrLoggerExists
	}

	if err := cfg.CompareDefault(defaultLogConfig); err != nil {
		return err
	}

	prefixLength, _ := cfg.GetInt("PREFIXLENGTH")
	flags, _ := cfg.GetString("FLAGS")
	rawLogLevel, _ := cfg.GetString("LEVEL")
	logLevel, err := resolveLogLevel(rawLogLevel)
	if err != nil {
		return err
	}
	logToConsole, _ := cfg.GetBool("WRITERS.STDOUT")
	logToFile, _ := cfg.GetBool("WRITERS.FILE")
	logSuffix, _ := cfg.GetString("SUFFIX")
	logFolder, _ := cfg.GetString("FOLDER")
	rotating, _ := cfg.GetBool("ROTATING")
	rotateFormat, _ := cfg.GetString("ROTATEFORMAT")

	var logFile string
	logFile, err = cfg.GetString("OUTFILE")
	if err != nil || logFile == "" {
		logFile = path.Join(logFolder, prefix+logSuffix)
		logFile = strings.ToLower(logFile)
	}
	if !path.IsAbs(logFile) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		logFile = path.Join(cwd, logFile)
	}

	if logToFile {
		if file, err := generateLogFile(logFile); err != nil {
			return err
		} else {
			file.Close()
		}
	}

	writer, err := newLogWriter(logToConsole, logToFile, logFile, rotating, rotateFormat)
	if err != nil {
		return err
	}

	logger := log.New(
		writer,
		generateLogPrefix(prefix, prefixLength),
		generateLogFlags(flags),
	)

	log.Println("Starting log stream for new logger: ", prefix)

	subCtx, cancel := context.WithCancel(ctx)
	logWrapper := &loggerWrapper{
		ctx:        subCtx,
		cancel:     cancel,
		logger:     logger,
		writer:     writer,
		level:      logLevel,
		terminated: make(chan bool),
	}
	go logWrapper.shutdownHook()
	loggerList[prefix] = logWrapper

	return nil
}

func (l *loggerWrapper) shutdownHook() {
	<-l.ctx.Done()
	println("Shutting down logger")
	if err := l.writer.Close(); err != nil {
		log.Println("Error closing log writer:", err)
	}
	l.terminated <- true
}

func UnregisterLogger(prefix string) error {
	if _, ok := loggerList[prefix]; !ok {
		return ErrLoggerExists
	}
	loggerList[prefix].cancel()
	<-loggerList[prefix].terminated
	delete(loggerList, prefix)
	return nil
}

// SetDefaultLoggerFlags sets the default flags for the logger
func generateLogFlags(flags string) int {
	if flags == "" {
		return 0
	}
	flagList := strings.Split(flags, ",")
	flagBuffer := 0
	for _, flag := range flagList {
		switch strings.ToLower(flag) {
		case "date":
			flagBuffer |= log.Ldate
		case "time":
			flagBuffer |= log.Ltime
		case "microseconds":
			flagBuffer |= log.Lmicroseconds
		case "utc":
			flagBuffer |= log.LUTC
		case "shortfile":
			flagBuffer |= log.Lshortfile
		case "longfile":
			flagBuffer |= log.Llongfile
		case "msgprefix":
			flagBuffer |= log.Lmsgprefix
		case "stdflags":
			flagBuffer |= log.LstdFlags
		default:
			log.Println("unknown flag", flag)
		}
	}
	return flagBuffer
}

func generateLogFile(filepath string) (*os.File, error) {
	// check if path is valid
	if filepath == "" {
		return nil, errors.New("path is empty")
	}

	dir := path.Dir(filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(filepath); !os.IsNotExist(err) {
		return nil, ErrFileInUse
	}

	// log.Println("Creating log file at", filepath)
	file, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func generateLogPrefix(raw_prefix string, length int) string {
	var prefix string
	if raw_prefix == "" {
		prefix = fmt.Sprintf("logger-%d", rand.Intn(1000))
	} else {
		prefix = raw_prefix
	}

	if len(raw_prefix) < length {
		prefix += strings.Repeat(" ", length-len(raw_prefix))
	}

	return prefix
}

func Log(prefix string, level LogLevel, msg string) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	if !ok {
		log.Printf("Invalid call to logger %s from %s:%d\n", prefix, file, line)
		return
	}
	wrapper, ok := loggerList[prefix]
	if !ok || level < wrapper.level {
		return
	}
	file = path.Base(file)
	message := fmt.Sprintf(" [%s] (%s:%d)\t%s", level.Sprint(), file, line, msg)
	wrapper.logger.Println(message)
}

func Info(prefix string, msg string) {
	Log(prefix, LevelInfo, msg)
}

func Debug(prefix string, msg string) {
	Log(prefix, LevelDebug, msg)
}

func Warn(prefix string, msg string) {
	Log(prefix, LevelWarn, msg)
}

func Error(prefix string, msg string) {
	Log(prefix, LevelError, msg)
}

func Cleanup() {
	log.Println("Closing all loggers")
	for prefix := range loggerList {
		if err := UnregisterLogger(prefix); err != nil {
			log.Println("Error closing logger:", err)
		}
	}
}

func CleanupMsg(msg string) {
	for prefix, wrapper := range loggerList {
		wrapper.logger.Println(msg)
		if err := UnregisterLogger(prefix); err != nil {
			log.Println("Error closing logger:", err)
		}
	}
}
