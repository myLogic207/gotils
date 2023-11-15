package logger

import (
	"errors"
	"strings"
)

var (
	ErrInvalidLogLevel = errors.New("invalid log level")
)

const (
	LevelDebug LogLevel = 0
	LevelInfo  LogLevel = 100
	LevelWarn  LogLevel = 200
	LevelError LogLevel = 300
)

type LogLevel int

func resolveLogLevel(rawLevel interface{}) (LogLevel, error) {
	switch level := rawLevel.(type) {
	case LogLevel:
		return level, nil
	case string:
		return resolveLogLevelString(strings.ToUpper(level))
	case int:
		return resolveLogLevelInt(level)
	default:
		return 0, ErrInvalidLogLevel
	}
}

func resolveLogLevelString(level string) (LogLevel, error) {
	switch level {
	case "DEBUG":
		return LevelDebug, nil
	case "INFO":
		return LevelInfo, nil
	case "WARN":
		return LevelWarn, nil
	case "ERROR":
		return LevelError, nil
	default:
		return LevelInfo, ErrInvalidLogLevel
	}
}

func resolveLogLevelInt(level int) (LogLevel, error) {
	switch {
	case level < 100:
		return LevelDebug, nil
	case level < 200:
		return LevelInfo, nil
	case level < 300:
		return LevelWarn, nil
	case level < 400:
		return LevelError, nil
	default:
		return LevelInfo, ErrInvalidLogLevel
	}
}

func (level LogLevel) Sprint() string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO "
	case LevelWarn:
		return "WARN "
	case LevelError:
		return "ERROR"
	default:
		return "UNKWN"
	}
}
