package lifecycle

import (
	"context"
	"errors"

	"github.com/myLogic207/gotils/config"
)

var (
	ErrInvalidSystem = errors.New("invalid subsystem, must implement SubSystem interface")
	ErrNilConfig     = errors.New("config is nil")
	ErrInvalidHook   = errors.New("invalid hook")
)

type SubSystem interface {
	Init(context.Context, config.Config) error
	Shutdown() error
	// UpdateConfig(*config.Config) error
}

// Init system Wraps a context for a given "Subsystem"
// A subSystem is a struct that implements the SubSystem interface,
// which holds some basic functions for managing the lifecycle of a system
type SystemWrapper struct {
	SubSystem
	name string
}

func NewSystemWrapper(name string, system SubSystem) (*SystemWrapper, error) {
	systemWrapper := &SystemWrapper{
		name:      name,
		SubSystem: system,
	}
	return systemWrapper, nil
}
