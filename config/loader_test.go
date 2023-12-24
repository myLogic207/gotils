package config

import (
	"context"
	"os"
	"sync"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	// Set up environment variables for testing
	os.Setenv("TEST_KEY1", "VALUE1")
	os.Setenv("TEST_KEY2", "VALUE2")
	os.Setenv("OTHER_KEY", "OTHER_VALUE")

	loader := &ConfigLoader{}

	// Test with "TEST_" prefix
	ctx := context.Background()

	store := &ConfigStoreImpl{
		mu:    sync.RWMutex{},
		store: map[string]string{},
	}
	err := loader.LoadEnv(ctx, store, []string{"TEST"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(store.store) != 2 {
		t.Errorf("Expected 2 store.store, got %v", len(store.store))
	}

	if store.store["KEY1"] != "VALUE1" || store.store["KEY2"] != "VALUE2" {
		t.Errorf("Unexpected store.store: %v", store.store)
	}

	// Test with "OTHER_" prefix
	if err = loader.LoadEnv(ctx, store, []string{"OTHER"}); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(store.store) != 3 {
		t.Errorf("Expected 3 entry, got %v", len(store.store))
	}

	if store.store["KEY"] != "OTHER_VALUE" {
		t.Errorf("Unexpected store.store: %v", store.store)
	}
}

func TestLoadEnvFromFile(t *testing.T) {
	testValue := "abcdefg1234567!"
	t.Log("Testing Config Load with File")
	if err := os.WriteFile("test_conf.env", []byte(testValue), 0644); err != nil {
		t.Fatal("Failed to create test file")
	}
	t.Cleanup(func() {
		os.Remove("test_conf.env")
	})
	ctx := context.TODO()
	os.Setenv("PATCHTESTCONF_SIMPLE_FILE", "test_conf.env")
	// os.Setenv("PATCHTEST_MAP_FILE", "abcde")
	store := &ConfigStoreImpl{
		mu:    sync.RWMutex{},
		store: make(map[string]string),
	}
	loader := &ConfigLoader{}
	if err := loader.LoadEnv(ctx, store, []string{"PATCHTESTCONF"}); err != nil {
		t.Fatal(err)
	}
	if val, err := store.Get(ctx, "SIMPLE"); err != nil || val != testValue {
		t.Fatal("Config is not loaded correctly (level 0)")
	}
}
