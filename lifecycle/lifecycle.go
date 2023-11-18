package lifecycle

import (
	"context"
	"errors"
	"gotils/config"
	log "gotils/logger"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	KEY_SYSTEMS = "SYSTEMCONFIGS"
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
	initialized bool
	logger      *log.Logger
	systems     map[string]SubSystem
	configTree  *config.Config
}

func NewInitializer(ctx context.Context, options *config.Config) (*Initializer, error) {
	cfg := config.NewConfigWithInitialValues(defaultConfig)
	cfg.Merge(options, true)
	if err := cfg.CompareDefault(defaultConfig); err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}

	logger, err := log.NewLogger(cfg)
	if err != nil {
		return nil, errors.Join(ErrInitConfig, err)
	}

	initSystem := &Initializer{
		logger:      logger,
		systems:     make(map[string]SubSystem),
		initialized: false,
		configTree:  cfg,
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
	i.logger.Info("Adding system: ", name)
	systemWrapper, err := NewSystemWrapper(name, validSystem)
	if err != nil {
		return ErrWrappingSystem
	}
	systemKey := KEY_SYSTEMS + config.CONFIG_TREE_SEPARATOR + name
	if err := i.configTree.Set(systemKey, configOptions, true); err != nil {
		return errors.Join(ErrInitConfig, err)
	}
	i.systems[name] = systemWrapper
	i.logger.Debug("System added: ", name)
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
	i.logger.Info("Removing system", name)
	delete(i.systems, name)
	return nil
}

func (i *Initializer) Init(ctx context.Context, envPrefixes []string) error {
	if len(i.systems) == 0 {
		return ErrNothingToInit
	}
	if i.initialized {
		return ErrAlreadyInitialized
	}
	i.initialized = true

	i.logger.Info("Initializing system")
	loadedOptions, err := config.LoadConfig(ctx, envPrefixes, nil, false)
	if err != nil && !errors.Is(err, config.ErrNoConfigSource) {
		return err
	} else if errors.Is(err, config.ErrNoConfigSource) {
		i.logger.Warn("No config source found")
	} else if err := i.configTree.Merge(loadedOptions, true); err != nil {
		return errors.Join(ErrInitConfig, err)
	}
	print(i.configTree.Sprint())

	return i.unrestrictedInit(ctx)
}

func (i *Initializer) unrestrictedInit(ctx context.Context) error {
	systemConfig, err := i.configTree.GetConfig(KEY_SYSTEMS)
	if err != nil {
		return errors.Join(ErrInitConfig, err)
	}

	errChan := make(chan error)
	finishChan := make(chan bool)
	waitGroup := sync.WaitGroup{}
	print(systemConfig.Sprint())

	for systemName, subSystem := range i.systems {
		waitGroup.Add(1)
		i.logger.Info("Initializing", systemName)
		systemConfig, err := systemConfig.GetConfig(systemName)
		if err != nil || systemConfig == nil {
			return errors.Join(ErrInitConfig, err)
		}
		i.logger.Info("Config loaded")
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

	return i.handleFinishStream(errChan, finishChan)
}

func (i *Initializer) Shutdown() error {
	waitGroup := sync.WaitGroup{}
	errChan := make(chan error)
	finishChan := make(chan bool)

	for name, system := range i.systems {
		i.logger.Info("Shutting down", name)
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

	return i.handleFinishStream(errChan, finishChan)
}

func (i *Initializer) handleFinishStream(errChan <-chan error, finished <-chan bool) error {
	timeout, _ := i.configTree.GetDuration("TIMEOUT")
	var joinedErr error
	for {
		select {
		case <-time.After(timeout):
			i.logger.Warn("Operation timed out")
			return ErrTimeout
		case err := <-errChan:
			if err != nil {
				i.logger.Error("Error received: ", err.Error())
				joinedErr = errors.Join(joinedErr, err)
			}
		case <-finished:
			i.logger.Info("Finished")
			return joinedErr
		}
	}
}
