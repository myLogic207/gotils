package buffer

import (
	"context"
	"errors"
	"gotils/config"
	"testing"
)

var (
	bufferTestStackConfig = map[string]interface{}{
		"MODE": int(MODE_STACK),
		"SIZE": 2,
	}
	bufferTestQueueConfig = map[string]interface{}{
		"MODE": int(MODE_QUEUE),
		"SIZE": 2,
	}
)

func TestStackBufferStoreRetrieve(t *testing.T) {
	bufferCtx := context.Background()
	buffer, err := NewBuffer[string](bufferCtx, config.NewConfigWithInitialValues(bufferCtx, bufferTestStackConfig))
	if err != nil {
		t.Log(err)
		t.Error("Buffer is not creating correctly")
		t.FailNow()
	}
	buffer.Add("test1")
	buffer.Add("test2")
	if val, err := buffer.Get(); err != nil || val != "test2" {
		t.Log(val)
		t.Log(err)
		t.Error("Buffer is not storing and retrieving correctly")
	}
	if val, err := buffer.Get(); err != nil || val != "test1" {
		t.Log(val)
		t.Log(err)
		t.Error("Buffer is not storing and retrieving correctly")
	}
	if val, err := buffer.Get(); err == nil {
		t.Log(val)
		t.Log(err)
		t.Error("Buffer is not storing and retrieving correctly")
	}
}

func TestQueueBufferStoreRetrieve(t *testing.T) {
	bufferCtx := context.Background()
	buffer, err := NewBuffer[string](bufferCtx, config.NewConfigWithInitialValues(bufferCtx, bufferTestQueueConfig))
	if err != nil {
		t.Log(err)
		t.Error("Buffer is not creating correctly")
		t.FailNow()
	}
	buffer.Add("test1")
	buffer.Add("test2")
	if val, err := buffer.Get(); err != nil || val != "test1" {
		t.Log(val)
		t.Log(err)
		t.Error("Buffer is not storing and retrieving correctly")
	}
	if val, err := buffer.Get(); err != nil || val != "test2" {
		t.Log(val)
		t.Log(err)
		t.Error("Buffer is not storing and retrieving correctly")
	}
	if val, err := buffer.Get(); err == nil {
		t.Log(val)
		t.Log(err)
		t.Error("Buffer is not storing and retrieving correctly")
	}
}

func TestStackBufferOverflow(t *testing.T) {
	bufferCtx := context.Background()
	buffer, err := NewBuffer[string](bufferCtx, config.NewConfigWithInitialValues(bufferCtx, bufferTestStackConfig))
	if err != nil {
		t.Log(err)
		t.Error("Buffer is not creating correctly")
		t.FailNow()
	}
	buffer.Add("test1")
	buffer.Add("test2")
	err = buffer.Add("test3")
	if err == nil {
		t.Error("Buffer is not overflowing correctly")
	}
	if !errors.Is(err, ErrAddElement) {
		t.Error("Buffer is not overflowing correctly")
	}
}

func TestQueueBufferOverflow(t *testing.T) {
	bufferCtx := context.Background()
	buffer, err := NewBuffer[string](bufferCtx, config.NewConfigWithInitialValues(bufferCtx, bufferTestQueueConfig))
	if err != nil {
		t.Log(err)
		t.Error("Buffer is not creating correctly")
		t.FailNow()
	}
	buffer.Add("test1")
	buffer.Add("test2")
	err = buffer.Add("test3")
	if err == nil {
		t.Error("Buffer is not overflowing correctly")
	}
	if !errors.Is(err, ErrAddElement) {
		t.Error("Buffer is not overflowing correctly")
	}
}
