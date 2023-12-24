package config

import (
	"context"
	"sync"
	"testing"
)

func TestDelete(t *testing.T) {
	store := &ConfigStoreImpl{
		mu:    sync.RWMutex{},
		store: make(map[string]string),
	}
	store.store["SIMPLE"] = "test"

	ctx := context.TODO()

	if str, err := store.Get(ctx, "SIMPLE"); err != nil || str != "test" {
		t.Log(str)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if err := store.Set(ctx, "SIMPLE", "", true); err != nil {
		t.Error(err)
	}

	if str, err := store.Get(ctx, "SIMPLE"); err == nil || str != "" {
		t.Log(str)
		t.Log(err)
		t.Error("Config is not deleted correctly (level 0)")
	}
}

func TestSet(t *testing.T) {
	store := &ConfigStoreImpl{
		mu:    sync.RWMutex{},
		store: make(map[string]string),
	}

	err := store.Set(context.Background(), "TEST_KEY", "TEST_VALUE", false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	value, ok := store.store["TEST_KEY"]
	if !ok || value != "TEST_VALUE" {
		t.Errorf("Expected 'TEST_VALUE', got '%v'", value)
	}

	err = store.Set(context.Background(), "TEST_KEY", "", false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	_, ok = store.store["TEST_KEY"]
	if ok {
		t.Errorf("Expected key to be deleted")
	}
}

func TestKeys(t *testing.T) {
	store := &ConfigStoreImpl{
		mu: sync.RWMutex{},
		store: map[string]string{
			"KEY1": "VALUE1",
			"KEY2": "VALUE2",
		},
	}

	keys := store.Keys(context.Background())
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %v", len(keys))
	}

	// Check that keys are correct
	for _, key := range keys {
		if _, ok := store.store[key]; !ok {
			t.Errorf("Unexpected key: %v", key)
		}
	}
}

func TestGet(t *testing.T) {
	store := &ConfigStoreImpl{
		mu: sync.RWMutex{},
		store: map[string]string{
			"KEY1": "VALUE1",
			"KEY2": "VALUE2",
		},
	}

	value, err := store.Get(context.Background(), "KEY1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if value != "VALUE1" {
		t.Errorf("Expected 'VALUE1', got '%v'", value)
	}

	_, err = store.Get(context.Background(), "NON_EXISTENT_KEY")
	if err == nil {
		t.Errorf("Expected error for non-existent key, got nil")
	}
}
