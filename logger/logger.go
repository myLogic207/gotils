package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/myLogic207/gotils/config"
)

// type loggerCtxKey string

// const (
// 	ckeyPrefix loggerCtxKey = "logger-prefix"
// )

var (
	ErrCopyConfig     = errors.New("error copying config")
	ErrInitConfig     = errors.New("error initializing config")
	ErrPrefixNotSet   = errors.New("prefix not set")
	ErrLoggerExists   = errors.New("logger already exists")
	ErrFileInUse      = errors.New("log file is already in use")
	ErrFileNotActive  = errors.New("log file is not active")
	ErrSetLogger      = errors.New("error setting logger")
	invalidCharacters = []string{" ", "\t", "\n", "\r", "\v", "\f", ":", "=", "#", "\\", "\"", "'", "`", "/", ".", ",", ";", "!", "@", "$", "%", "^", "&", "*", "(", ")", "+", "-", "|", "[", "]", "{", "}", "<", ">", "?", "~"}
	defaultLogConfig  = map[string]interface{}{
		"PREFIX":      "LOGGER",
		"FLAGS":       "date,time,microseconds,utc,msgprefix",
		"COLUMLENGTH": 16,
		"REPLACECHAR": "-",
		"LEVEL":       "DEBUG",
		"WRITERS": map[string]interface{}{
			"STDOUT": true,
			"SYSLOG": false,
			"FILE": map[string]interface{}{
				"ACTIVE": false,
			},
		},
	}
	logFlagMap = map[string]int{
		"date":         log.Ldate,
		"time":         log.Ltime,
		"microseconds": log.Lmicroseconds,
		"utc":          log.LUTC,
		"shortfile":    log.Lshortfile,
		"longfile":     log.Llongfile,
		"msgprefix":    log.Lmsgprefix,
		"stdflags":     log.LstdFlags,
	}
)

type Logger interface {
	Shutdown(ctx context.Context) error
	UpdateLogger(ctx context.Context, config config.Config) error
	LogMode(ctx context.Context, level LogLevel) Logger
	Debug(ctx context.Context, msg string, args ...interface{})
	Info(ctx context.Context, msg string, args ...interface{})
	Warn(ctx context.Context, msg string, args ...interface{})
	Error(ctx context.Context, msg string, args ...interface{})
	// Trace for SQL/GORM
	Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
}

type logger struct {
	config  *config.Config
	logger  *log.Logger
	logFile *LogFile
}

// func GetDefaultConfig() *config.Config {
// 	if conf, err := config.NewWithInitialValues(context.Background(), defaultLogConfig); err != nil {
// 		panic(err)
// 	} else {
// 		return conf
// 	}
// }

// func SetDefaultConfig(ctx context.Context, cnf config.Config) error {
// 	if err := cnf.CompareMap(ctx, defaultLogConfig, false); err != nil {
// 		return err
// 	}

// 	configMap := cnf.GetMap()
// 	for key, value := range configMap {
// 		defaultLogConfig[key] = value
// 	}

// 	return nil
// }

func Init(ctx context.Context, configOptions *config.Config) (Logger, error) {
	cfg, err := config.WithInitialValuesAndOptions(ctx, defaultLogConfig, configOptions)
	if err != nil {
		return nil, err
	}

	wrapper := &logger{
		config: cfg,
		logger: nil,
	}

	if err := wrapper.setLogger(ctx); err != nil {
		return nil, err
	}

	log.Println("Starting log stream for new logger:", strings.TrimSpace(wrapper.logger.Prefix()))
	return wrapper, nil
}

func (l *logger) parseLogLevel(ctx context.Context) error {
	rawLogLevel, _ := l.config.Get(ctx, "LEVEL")
	logLevel, err := resolveLogLevel(rawLogLevel)
	if err != nil {
		return err
	}
	if err := l.config.Set(ctx, "LEVEL", logLevel.Sprint(), true); err != nil {
		return err
	}
	return nil
}

func (l *logger) Shutdown(ctx context.Context) error {
	println("Shutting down logger")
	logToFile, _ := l.config.Get(ctx, "WRITERS/FILE/ACTIVE")
	if ok, err := strconv.ParseBool(logToFile); err != nil {
		return err
	} else if !ok {
		return nil
	}
	if err := l.logFile.Close(ctx); err != nil {
		log.Println("Error closing log writer:", err)
		return err
	}
	l.logger.Println("Shutting down logger")
	l.logger = nil
	return nil
}

func (l *logger) UpdateLogger(ctx context.Context, config config.Config) error {
	currentConfig, err := l.config.Copy(ctx)
	if err != nil {
		return ErrCopyConfig
	}
	if err := l.config.Merge(ctx, config, true); err != nil {
		return err
	}

	fileActive, _ := currentConfig.Get(ctx, "WRITERS/FILE/ACTIVE")
	if ok, _ := strconv.ParseBool(fileActive); ok {
		if err := l.logFile.Close(ctx); err != nil {
			return err
		}
		l.logFile = nil
	}
	if err := l.setLogger(ctx); err != nil {
		return err
	}

	return nil
}

func (l *logger) LogMode(ctx context.Context, level LogLevel) Logger {
	if err := l.config.Set(ctx, "LEVEL", level.Sprint(), true); err != nil {
		println("Error setting log level:", err)
	}

	return l
}

func (l *logger) setLogger(ctx context.Context) error {
	if err := l.parseLogLevel(ctx); err != nil {
		return err
	}

	prefixLengthRaw, _ := l.config.Get(ctx, "COLUMLENGTH")
	prefixLength, err := strconv.Atoi(prefixLengthRaw)
	if err != nil {
		return fmt.Errorf("cannot use %s as prefix length: %w", prefixLengthRaw, err)
	}
	rawPrefix, _ := l.config.Get(ctx, "PREFIX")
	rawFlags, _ := l.config.Get(ctx, "FLAGS")
	replaceChar, _ := l.config.Get(ctx, "REPLACECHAR")

	if writer, err := l.generateWriter(ctx); err != nil {
		return errors.Join(ErrSetLogger, err)
	} else {
		l.logger = log.New(writer,
			formatPrefix(rawPrefix, prefixLength, []rune(replaceChar)[0]),
			generateLogFlags(rawFlags))
	}
	return nil
}

func (l *logger) generateWriter(ctx context.Context) (io.Writer, error) {
	var writers []io.Writer
	writeStdout, _ := l.config.Get(ctx, "WRITERS/STDOUT")
	if ok, err := strconv.ParseBool(writeStdout); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", writeStdout, err)
	} else if ok {
		writers = append(writers, os.Stdout)
	}

	file, err := l.getLogFile(ctx)
	if err != nil && !errors.Is(err, ErrFileNotActive) {
		return nil, errors.Join(ErrOpenLogFile, err)
	} else if err == nil && file != nil {
		l.logFile = file
		writers = append(writers, file)
	}

	return io.MultiWriter(writers...), nil
}

func (l *logger) getLogFile(ctx context.Context) (*LogFile, error) {
	fileActive, _ := l.config.Get(ctx, "WRITERS/FILE/ACTIVE")
	if ok, err := strconv.ParseBool(fileActive); err != nil || !ok {
		return nil, ErrFileNotActive
	}
	fileOptions, _ := l.config.GetConfig(ctx, "WRITERS/FILE")
	// check if prefix is not default
	if prefix, _ := l.config.Get(ctx, "PREFIX"); prefix != defaultLogConfig["PREFIX"] {
		// override logfile prefix with custom logger prefix
		if err := fileOptions.Set(ctx, "PREFIX", prefix, true); err != nil {
			return nil, errors.Join(ErrOpenLogFile, err)
		}
	}
	if file, err := NewLogFile(ctx, fileOptions); err != nil {
		return nil, errors.Join(ErrOpenLogFile, err)
	} else {
		return file, nil
	}
}

// SetDefaultLoggerFlags sets the default flags for the logger
func generateLogFlags(flags string) int {
	if flags == "" {
		return 0
	}
	flagList := strings.Split(flags, ",")
	flagBuffer := 0
	for _, flag := range flagList {
		if flag, ok := logFlagMap[flag]; ok {
			flagBuffer |= flag
		}
	}
	return flagBuffer
}

func formatPrefix(rawPrefix string, prefixLength int, replaceChar rune) string {
	prefix := strings.ToUpper(rawPrefix)
	for _, char := range invalidCharacters {
		prefix = strings.ReplaceAll(prefix, char, string(replaceChar))
	}

	if len(prefix) < prefixLength {
		prefix += strings.Repeat(" ", prefixLength-len(rawPrefix))
	}

	return prefix
}

func (l *logger) Logf(ctx context.Context, level LogLevel, msg string, args ...any) {
	loggerLevel, _ := l.config.Get(ctx, "LEVEL")
	resolvedLevel, err := resolveLogLevel(loggerLevel)
	if err != nil {
		resolvedLevel = LogLevel(0)
	}

	if resolvedLevel > level {
		return
	}
	logsMsg := level.Sprint() + " " + fmt.Sprintf(msg, args...)
	if err := l.logger.Output(2, logsMsg); err != nil {
		println("Error logging message:", err)
	}
}

func (l *logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	l.Logf(ctx, LevelInfo, "SQL Trace (starting: %s): %s", begin.String(), err)
}

func (l *logger) Info(ctx context.Context, msg string, args ...any) {
	l.Logf(ctx, LevelInfo, msg, args...)
}

func (l *logger) Debug(ctx context.Context, msg string, args ...any) {
	l.Logf(ctx, LevelDebug, msg, args...)
}

func (l *logger) Warn(ctx context.Context, msg string, args ...any) {
	l.Logf(ctx, LevelWarn, msg, args...)
}

func (l *logger) Error(ctx context.Context, msg string, args ...any) {
	l.Logf(ctx, LevelError, msg, args...)
}
