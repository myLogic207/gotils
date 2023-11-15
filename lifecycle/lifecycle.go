package lifecycle

import (
	"context"
	"errors"
	"gotils/config"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	CONFIGKEY_SYSTEMS = "SYSTEMCONFIGS"
)

var (
	// ErrInitConfig         = errors.New("error initializing config")
	ErrAlreadyInitialized = errors.New("already initialized")
	ErrNoSystem           = errors.New("system not found")
	ErrSystemRegistered   = errors.New("system already registered")
	ErrNothingToInit      = errors.New("nothing to init")
	ErrWrappingSystem     = errors.New("error wrapping system")
	ErrInitConfig         = errors.New("error initializing, config issue")
	ErrInitSystem         = errors.New("error initializing system")
	ErrTimeout            = errors.New("operation timed out")
	ErrClosedBeforeInit   = errors.New("closed before init")
)

var defaultConfig = map[string]interface{}{
	"LOGGER": map[string]interface{}{
		"PREFIX": "INITIALIZER",
		"WRITERS": map[string]interface{}{
			"STDOUT": true,
		},
	},
	"TIMEOUT": "5s",
}

// CatchInterrupt starts a goroutine that catches SIGINT and SIGTERM and cancels the context
func CatchInterrupt(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()
}

// Initializer is a struct that initializes a system with a config
// when init is called, it loads the config, merges it with the default config and then initializes the subsystems
// it also registers a shutdown hook that cancels the context
type Initializer struct {
	ctx         context.Context
	initialized bool
	systems     map[string]SubSystem
	timeout     time.Duration
	configTree  *config.Config
}

func NewInitializer(ctx context.Context, options *config.Config) (*Initializer, error) {
	cfg := config.NewConfigWithInitialValues(ctx, defaultConfig)
	cfg.Merge(options, true)
	if err := cfg.CompareDefault(defaultConfig); err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}

	timeout, err := cfg.GetDuration("TIMEOUT")
	if err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}

	initSystem := &Initializer{
		ctx:         ctx,
		systems:     make(map[string]SubSystem),
		initialized: false,
		timeout:     timeout,
		configTree:  cfg,
	}
	if err := initSystem.configTree.Set(CONFIGKEY_SYSTEMS, make(map[string]interface{}), true); err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}
	return initSystem, nil
}

func (i *Initializer) AddSystem(name string, system interface{}, configOptions *config.Config) error {
	validSystem, ok := system.(SubSystem)
	if !ok {
		return ErrInvalidSystem
	}
	if _, ok := i.systems[name]; ok {
		return ErrSystemRegistered
	}
	systemWrapper, err := NewSystemWrapper(i.ctx, name, validSystem)
	if err != nil {
		return ErrWrappingSystem
	}
	if err := i.configTree.Set(CONFIGKEY_SYSTEMS+config.CONFIG_TREE_SEPARATOR+name, configOptions, true); err != nil {
		return errors.Join(ErrInitConfig, err)
	}
	i.systems[name] = systemWrapper
	return nil
}

func (i *Initializer) GetSubSystems() map[string]interface{} {
	subSystems := make(map[string]interface{}, len(i.systems))
	for prefix, systemWrapper := range i.systems {
		subSystems[prefix] = systemWrapper
	}
	return subSystems
}

// Removes a system from the initializer, does not shutdown the system
func (i *Initializer) RemoveSystem(name string) error {
	if _, ok := i.systems[name]; !ok {
		return ErrNoSystem
	}
	delete(i.systems, name)
	return nil
}

func (i *Initializer) Init(envPrefix string) error {
	if len(i.systems) == 0 {
		return ErrNothingToInit
	}
	if i.initialized {
		return ErrAlreadyInitialized
	}
	i.initialized = true

	log.Println("Initializing system")
	loadedOptions, err := config.LoadConfig(envPrefix, i.ctx)
	if err != nil {
		return err
	}
	if err := i.configTree.Merge(loadedOptions, true); err != nil {
		return errors.Join(ErrInitConfig, err)
	}

	systemConfig, err := i.configTree.GetConfig(CONFIGKEY_SYSTEMS)
	if err != nil {
		return errors.Join(ErrInitConfig, err)
	}

	return unrestrictedInit(i.ctx, i.timeout, systemConfig, i.systems)
}

func unrestrictedInit(ctx context.Context, timeout time.Duration, masterConfig *config.Config, systems map[string]SubSystem) error {
	errChan := make(chan error)
	finishChan := make(chan bool)
	waitGroup := sync.WaitGroup{}
	masterConfig.Print()

	for systemName, subSystem := range systems {
		waitGroup.Add(1)
		log.Println("Initializing", systemName)
		systemConfig, err := masterConfig.GetConfig(systemName)
		if err != nil || systemConfig == nil {
			return errors.Join(ErrInitConfig, err)
		}
		log.Println("Config loaded")
		go func(system SubSystem, c *config.Config) {
			errChan <- system.Init(ctx, c)
			waitGroup.Done()
		}(subSystem, systemConfig)
	}

	go func() {
		waitGroup.Wait()
		finishChan <- true
		close(errChan)
	}()

	return HandleFinishStream(timeout, errChan, finishChan)
}

func (i *Initializer) Shutdown() error {
	waitGroup := sync.WaitGroup{}
	errChan := make(chan error)
	finishChan := make(chan bool)

	for name, system := range i.systems {
		log.Println("Shutting down", name)
		waitGroup.Add(1)
		go func(s SubSystem) {
			errChan <- s.Shutdown()
			waitGroup.Done()
		}(system)
	}

	go func() {
		waitGroup.Wait()
		finishChan <- true
		close(errChan)
	}()

	return HandleFinishStream(i.timeout, errChan, finishChan)
}

func HandleFinishStream(timeout time.Duration, errChan <-chan error, finished <-chan bool) error {
	var joinedErr error
	for {
		select {
		case <-time.After(timeout):
			return errors.Join(ErrTimeout, joinedErr)
		case err := <-errChan:
			joinedErr = errors.Join(joinedErr, err)
		case <-finished:
			log.Println("Finished")
			return joinedErr
		}
	}
}
