package config

import (
	"errors"
	"strings"
	"time"
)

type ConfigGetter interface {
	GetInt(key string) (int, error)
	GetString(key string) (string, error)
	GetBool(key string) (bool, error)
	GetFloat(key string) (float64, error)
	GetConfig(key string) (Config, error)
	GetDuration(key string) (time.Duration, error)
}

var (
	ErrTypeMismatch = errors.New("type mismatch")
)

func (c *ConfigStore) GetString(key string) (string, error) {
	entry, err := c.Get(key)
	if err != nil {
		return "", err
	}
	if str, ok := entry.(string); ok {
		return str, nil
	}
	return "", &ErrFieldNotString{
		key: key,
	}
}

func (c *ConfigStore) GetInt(key string) (int, error) {
	entry, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	if num, ok := entry.(int); ok {
		return num, nil
	}
	return 0, &ErrFieldNotInt{
		key: key,
	}
}

func (c *ConfigStore) GetFloat(key string) (float64, error) {
	entry, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	if num, ok := entry.(float64); ok {
		return num, nil
	}
	return 0, &ErrFieldNotFloat{
		key: key,
	}
}

func (c *ConfigStore) GetBool(key string) (bool, error) {
	entry, err := c.Get(key)
	if err != nil {
		return false, err
	}
	return resolveBool(key, entry)
}

func resolveBool(key string, raw any) (bool, error) {
	err := &ErrFieldNotBool{
		key: key,
	}
	switch entry := raw.(type) {
	case string:
		return resolveStringBool(key, entry)
	case bool:
		return entry, nil
	case int:
		return entry != 0, nil
	default:
		return false, err
	}
}

func resolveStringBool(key string, raw string) (bool, error) {
	raw = strings.ToLower(raw)
	switch raw {
	case "yes":
		fallthrough
	case "y":
		fallthrough
	case "1":
		fallthrough
	case "true":
		return true, nil
	case "no":
		fallthrough
	case "n":
		fallthrough
	case "0":
		fallthrough
	case "false":
		return false, nil
	default:
		return false, &ErrFieldNotBool{
			key: key,
		}
	}
}

func (c *ConfigStore) GetDuration(key string) (time.Duration, error) {
	raw, err := c.Get(key)
	if err != nil {
		return time.Duration(0), err
	}
	durErr := &ErrFieldNotDuration{
		key: key,
	}
	switch entry := raw.(type) {
	case string:
		duration, err := time.ParseDuration(entry)
		if err != nil {
			return time.Duration(0), durErr
		}
		return duration, nil
	case int:
		return time.Duration(entry) * time.Millisecond, nil
	default:
		return time.Duration(0), durErr
	}
}

func (c *ConfigStore) GetConfig(key string) (Config, error) {
	entry, err := c.Get(key)
	if err != nil {
		return nil, err
	}
	if config, ok := entry.(map[string]interface{}); ok {
		return NewWithInitialValues(config), nil
	}
	return NewWithInitialValues(map[string]interface{}{
		key: entry,
	}), nil
}
