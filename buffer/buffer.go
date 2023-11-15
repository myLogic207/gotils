package buffer

import (
	"context"
	"errors"
	"fmt"
	"gotils/config"
	"gotils/logger"
	"sync"
)

type BufferMode int

const (
	MODE_STACK BufferMode = iota
	MODE_QUEUE
)

var (
	ErrInvalidMode   = errors.New("invalid mode")
	ErrGetElement    = errors.New("cannot get element from buffer")
	ErrAddElement    = errors.New("cannot add element to buffer")
	bufferConfigBase = map[string]interface{}{
		"LOGGER": map[string]interface{}{
			"PREFIX": "BUFFER",
		},
		"MODE": MODE_STACK,
		"SIZE": 100,
	}
)

type Buffer[T interface{}] struct {
	sync.Mutex
	prefix string
	size   int
	store  bufferStore
	ctx    context.Context
}

type bufferStore interface {
	add(element interface{}) bool
	get() interface{}
}

// message buffer is a FIFO buffer, can operate in stack or queue mode
func NewBuffer[storeType interface{}](ctx context.Context, options *config.Config) (*Buffer[storeType], error) {
	config := config.NewConfigWithInitialValues(ctx, bufferConfigBase)
	if err := config.Merge(options, true); err != nil {
		return nil, err
	}
	if err := config.CompareDefault(bufferConfigBase); err != nil {
		return nil, err
	}

	size, _ := config.GetInt("SIZE")
	rawMode, _ := config.GetInt("MODE")
	bufferStore, err := resolveRawModeToBuffer(rawMode, size)
	if err != nil {
		return nil, err
	}
	loggerConfig, _ := config.GetConfig("LOGGER")
	prefix, err := loggerConfig.GetString("PREFIX")
	if err != nil {
		return nil, err
	}

	logger.RegisterLogger(ctx, loggerConfig)
	buffer := Buffer[storeType]{
		prefix: prefix,
		ctx:    ctx,
		size:   size,
		store:  bufferStore,
	}
	logger.Info(prefix, fmt.Sprintf("Created new buffer with size %d", size))
	go buffer.contextDone()
	return &buffer, nil
}

func (b *Buffer[_]) contextDone() {
	<-b.ctx.Done()
	logger.Info(b.prefix, "Shutting down buffer")
	b.Lock()
	defer b.Unlock()
	b.store = nil
}

func resolveRawModeToBuffer(rawMode int, size int) (bufferStore, error) {
	mode := BufferMode(int(rawMode))
	rawStore := make([]interface{}, size)
	switch mode {
	case MODE_STACK:
		return &stackBuffer{
			store: rawStore,
			head:  0,
		}, nil
	case MODE_QUEUE:
		return &queueBuffer{
			store: rawStore,
		}, nil
	default:
		return nil, ErrInvalidMode
	}
}

func (b *Buffer[elementType]) Add(element interface{}) error {
	if b.store == nil {
		return ErrAddElement
	}
	parsed, ok := element.(elementType)
	if !ok {
		return ErrAddElement
	}
	b.Lock()
	defer b.Unlock()
	if ok := b.store.add(parsed); !ok {
		return ErrAddElement
	}
	logger.Info(b.prefix, "Added element to buffer")
	return nil
}

func (b *Buffer[elementType]) Get() (elementType, error) {
	var element elementType
	if b.store == nil {
		return element, ErrGetElement
	}
	b.Lock()
	defer b.Unlock()
	if parsed, ok := b.store.get().(elementType); ok {
		return parsed, nil
	}
	logger.Info(b.prefix, "Get element from buffer")
	return element, ErrGetElement
}
