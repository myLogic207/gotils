package lifecycle

import (
	"context"
	"errors"
	"gotils/config"
)

var (
	ErrInvalidSystem = errors.New("invalid subsystem, must implement SubSystem interface")
	ErrNilConfig     = errors.New("config is nil")
	ErrInvalidHook   = errors.New("invalid hook")
)

type SubSystem interface {
	Init(context.Context, *config.Config) error
	Shutdown() error
	// UpdateConfig(*config.Config) error
}

// Init system Wraps a context for a given "Subsystem"
// A subSystem is a struct that implements the SubSystem interface,
// which holds some basic functions for managing the lifecycle of a system
type SystemWrapper struct {
	SubSystem
	ctx  context.Context
	Name string
}

func NewSystemWrapper(ctx context.Context, name string, system SubSystem) (*SystemWrapper, error) {
	systemWrapper := &SystemWrapper{
		ctx:       ctx,
		Name:      name,
		SubSystem: system,
	}
	return systemWrapper, nil
}
