package config

import (
	"errors"
	"time"
)

var (
	ErrKeyNotFound    = errors.New("key not found")
	ErrFieldNotConfig = errors.New("field is not a config")
	ErrFieldNotString = errors.New("field is not a string")
	ErrFieldNotInt    = errors.New("field is not an int")
	ErrFieldNotFloat  = errors.New("field is not a float")
	ErrFieldNotBool   = errors.New("field is not a bool")
)

func (c *Config) GetString(key string) (string, error) {
	entry, err := c.Get(key)
	if err != nil {
		return "", err
	}
	if str, ok := entry.(string); ok {
		return str, nil
	}
	return "", ErrFieldNotString
}

func (c *Config) GetInt(key string) (int, error) {
	entry, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	if num, ok := entry.(int); ok {
		return num, nil
	}
	return 0, errors.Join(ErrFieldNotInt, errors.New(key))
}

func (c *Config) GetFloat(key string) (float64, error) {
	entry, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	if num, ok := entry.(float64); ok {
		return num, nil
	}
	return 0, errors.Join(ErrFieldNotFloat, errors.New(key))
}

func (c *Config) GetBool(key string) (bool, error) {
	entry, err := c.Get(key)
	if err != nil {
		return false, err
	}
	if boolean, ok := entry.(bool); ok {
		return boolean, nil
	}
	return false, errors.Join(ErrFieldNotBool, errors.New(key))
}

func (c *Config) GetDuration(key string) (time.Duration, error) {
	entry, err := c.GetString(key)
	if err != nil {
		return time.Duration(0), err
	}
	return time.ParseDuration(entry)
}

func (c *Config) GetConfig(key string) (*Config, error) {
	entry, err := c.Get(key)
	if err != nil {
		return nil, err
	}
	if config, ok := entry.(*Config); ok {
		return config, nil
	}
	return nil, errors.Join(ErrFieldNotConfig, errors.New(key))
}
