package config

import (
	"context"
	"slices"
	"strings"
	"sync"
)

type ConfigStore interface {
	Get(ctx context.Context, key string) (string, error)
	GetAll(ctx context.Context, key string) map[string]string
	Set(ctx context.Context, key string, value string, force bool) error
	Has(ctx context.Context, key string) bool
	// HasAllKeys(cmp map[string]interface{}) error
	Keys(ctx context.Context) []string
}

type ConfigStoreNew func(context.Context) (ConfigStore, error)

var DefaultConfigStore ConfigStoreNew = NewConfigStore

func NewConfigStore(ctx context.Context) (ConfigStore, error) {
	return &ConfigStoreImpl{
		mu:    sync.RWMutex{},
		store: make(map[string]string),
	}, nil
}

// type ConfigStoreImpl map[string]interface{}
type ConfigStoreImpl struct {
	mu    sync.RWMutex
	store map[string]string
}

func (c *ConfigStoreImpl) Has(ctx context.Context, key string) bool {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)
	if err := IsValidKey(key); err != nil { // check key is valid
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for k := range c.store {
		if err := ctx.Err(); err != nil {
			// context Cancelled
			return false
		}
		if strings.HasPrefix(k, key) {
			return true
		}
	}
	return false
}

func (c *ConfigStoreImpl) HasAllKeys(ctx context.Context, cmp []string) error {
	keys := c.Keys(ctx)
	slices.Sort(keys)
	slices.Sort(cmp)
	if slices.Compare(keys, cmp) >= 0 {
		return &ErrKeyNotFound{}
	}
	return nil
}

func (c *ConfigStoreImpl) Get(ctx context.Context, key string) (string, error) {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)
	if err := IsValidKey(key); err != nil { // check key is valid
		return "", err
	}
	if !c.Has(ctx, key) {
		return "", &ErrKeyNotFound{key: key}
	}
	matchedValues := c.GetAll(ctx, key)
	if len(matchedValues) == 1 {
		return matchedValues[""], nil
	} else {
		return "", &ErrKeyAmbiguous{key: key}
	}
}

// GetAll returns all values that match the given key.
// If the key is not found, nil is returned.
// // If the key is found, but no values match, an empty map is returned.
// If the key is found, and values match, a map of values is returned, index by the key suffix
// (the part of the key after the last CONFIG_TREE_SEPARATOR)
// Single values are therefore indexed by an empty string.
func (c *ConfigStoreImpl) GetAll(ctx context.Context, key string) map[string]string {
	values := make(map[string]string)
	c.mu.RLock()
	defer c.mu.RUnlock()
	for k, v := range c.store {
		if err := ctx.Err(); err != nil {
			// context Cancelled, return what we have
			return values
		}
		if !strings.HasPrefix(k, key) {
			continue
		}
		trimKey := strings.TrimPrefix(k, key)
		// if trimKey == "" || len(trimKey) == 1 && string(trimKey[0]) == CONFIG_TREE_SEPARATOR {
		// 	println("direct match:", k)
		// 	return v
		// }
		trimKey = strings.TrimPrefix(trimKey, CONFIG_TREE_SEPARATOR)
		values[trimKey] = v
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (c *ConfigStoreImpl) Set(ctx context.Context, key string, value string, force bool) error {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)

	c.mu.Lock()
	defer c.mu.Unlock()
	if value == "" {
		delete(c.store, key) // delete key if value is nil or empty
	} else {
		c.store[key] = value
	}
	return nil
}

func (c *ConfigStoreImpl) Keys(ctx context.Context) []string {
	keys := []string{}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for key := range c.store {
		if err := ctx.Err(); err != nil {
			// context Cancelled, return what we have
			return keys
		}
		keys = append(keys, key)
	}
	return keys
}
