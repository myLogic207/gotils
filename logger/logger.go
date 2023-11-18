package logger

import (
	"errors"
	"fmt"
	"gotils/config"
	"io"
	"log"
	"os"
	"strings"
)

var (
	ErrInitConfig    = errors.New("error initializing config")
	ErrPrefixNotSet  = errors.New("prefix not set")
	ErrLoggerExists  = errors.New("logger already exists")
	ErrFileInUse     = errors.New("log file is already in use")
	ErrFileNotActive = errors.New("log file is not active")
	ErrSetLogger     = errors.New("error setting logger")
	defaultLogConfig = map[string]interface{}{
		"PREFIX":       "LOGGER",
		"FLAGS":        "date,time,microseconds,utc,msgprefix",
		"PREFIXLENGTH": 16,
		"REPLACECHAR":  "-",
		"LEVEL":        "DEBUG",
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

type Logger struct {
	config  *config.Config
	logger  *log.Logger
	logFile *LogFile
}

func GetDefaultConfig() *config.Config {
	return config.NewConfigWithInitialValues(defaultLogConfig)
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

func NewLogger(configOptions *config.Config) (*Logger, error) {
	cfg := config.NewConfigWithInitialValues(defaultLogConfig)
	if err := cfg.Merge(configOptions, true); err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}
	if err := cfg.CompareDefault(defaultLogConfig); err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}

	cfg.Sprint()
	wrapper := &Logger{
		config: cfg,
		logger: nil,
	}

	if err := wrapper.setLogger(); err != nil {
		return nil, err
	}

	log.Println("Starting log stream for new logger:", strings.TrimSpace(wrapper.logger.Prefix()))
	return wrapper, nil
}

func (l *Logger) parseLogLevel() error {
	rawLogLevel, _ := l.config.GetString("LEVEL")
	logLevel, err := resolveLogLevel(rawLogLevel)
	if err != nil {
		return err
	}
	if err := l.config.Set("LEVEL", logLevel, true); err != nil {
		return err
	}
	return nil
}

func (l *Logger) Shutdown() error {
	println("Shutting down logger")
	if logToFile, _ := l.config.GetBool("WRITERS/FILE/ACTIVE"); logToFile {
		if err := l.logFile.Close(); err != nil {
			log.Println("Error closing log writer:", err)
			return err
		}
	}
	l.logger.Println("Shutting down logger")
	l.logger = nil
	return nil
}

func (l *Logger) UpdateLogger(config *config.Config) error {
	if err := l.config.Merge(config, true); err != nil {
		return err
	}
	if err := l.config.CompareDefault(defaultLogConfig); err != nil {
		return err
	}

	if err := l.Shutdown(); err != nil {
		return err
	}

	if err := l.setLogger(); err != nil {
		return err
	}

	return nil
}

func (l *Logger) setLogger() error {
	if err := l.parseLogLevel(); err != nil {
		return err
	}

	prefixLength, _ := l.config.GetInt("PREFIXLENGTH")
	rawPrefix, _ := l.config.GetString("PREFIX")
	rawFlags, _ := l.config.GetString("FLAGS")

	if writer, err := l.generateWriter(); err != nil {
		return errors.Join(ErrSetLogger, err)
	} else {
		l.logger = log.New(writer,
			formatPrefix(rawPrefix, prefixLength),
			generateLogFlags(rawFlags))
	}
	return nil
}

func (l *Logger) generateWriter() (io.Writer, error) {
	var writers []io.Writer
	if ok, _ := l.config.GetBool("WRITERS/STDOUT"); ok {
		writers = append(writers, os.Stdout)
	}

	file, err := l.getLogFile()
	if err != nil && !errors.Is(err, ErrFileNotActive) {
		return nil, errors.Join(ErrOpenLogFile, err)
	} else if err == nil && file != nil {
		writers = append(writers, file)
	}

	return io.MultiWriter(writers...), nil
}

func (l *Logger) getLogFile() (*LogFile, error) {
	fileActive, _ := l.config.GetBool("WRITERS/FILE/ACTIVE")
	if !fileActive {
		return nil, ErrFileNotActive
	}
	fileOptions, _ := l.config.GetConfig("WRITERS/FILE")
	if file, err := NewLogFile(fileOptions); err != nil {
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

func formatPrefix(rawPrefix string, prefixLength int) string {
	prefix := strings.ReplaceAll(strings.ToUpper(rawPrefix), " ", defaultLogConfig["REPLACECHAR"].(string))

	if len(prefix) < prefixLength {
		prefix += strings.Repeat(" ", prefixLength-len(rawPrefix))
	}

	return prefix
}

// Arguments are handled in the manner of fmt.Print.
func (l *Logger) logf(callDepth int, level LogLevel, msg string, args ...any) {
	callDepth++ // =1 for this frame
	logsMsg := level.Sprint() + " " + fmt.Sprintf(msg, args...)
	if err := l.logger.Output(callDepth, logsMsg); err != nil {
		println("Error logging message:", err)
	}
}

func (l *Logger) log(callDepth int, level LogLevel, msg ...any) {
	callDepth++ // =1 for this frame
	logsMsg := level.Sprint() + " " + fmt.Sprint(msg...)
	if err := l.logger.Output(callDepth, logsMsg); err != nil {
		println("Error logging message:", err)
	}
}

func (l *Logger) Log(level LogLevel, msg ...any) {
	loggerLevel, _ := l.config.Get("LEVEL")
	if loggerLevel.(LogLevel) > level {
		return
	}

	if !strings.Contains(msg[0].(string), "%") {
		l.log(2, LevelInfo, msg...)
	} else {
		l.logf(2, LevelInfo, msg[0].(string), msg[1:]...)
	}
}

func (l *Logger) Info(msg ...any) {
	l.Log(LevelInfo, msg...)
}

func (l *Logger) Debug(msg ...any) {
	l.Log(LevelDebug, msg...)
}

func (l *Logger) Warn(msg ...any) {
	l.Log(LevelWarn, msg...)
}

func (l *Logger) Error(msg ...any) {
	l.Log(LevelError, msg...)
}
