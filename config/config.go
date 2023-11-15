package config

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	CONFIG_TREE_SEPARATOR = "."
	KEY_ALLOWED_CHARS     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
)

var (
	ErrKeyInvalid     = errors.New("invalid key")
	ErrKeyInStore     = errors.New("key already in store")
	ErrKeyCharInvalid = errors.New("invalid character in key")
	ErrValueNotFound  = errors.New("no value")
	ErrValueInvalid   = errors.New("invalid value")
	ErrContextDone    = errors.New("context done")
	ErrMergeFailed    = errors.New("merge failed")
)

type Entry struct {
	key   string
	value interface{}
}

// config is an infinitely nested tree of Entries defined above
// keys are strings, ints, floats, or bools, arrays of these types, or maps(named) of these types
type Config struct {
	ctx context.Context
	sync.RWMutex
	store map[string]interface{}
}

func NewConfig(ctx context.Context) *Config {
	config := Config{
		ctx:   ctx,
		store: make(map[string]interface{}),
	}
	go func() {
		<-ctx.Done()
		config.delete()
	}()
	return &config
}

func (c *Config) delete() {
	for key, value := range c.store {
		if config, ok := value.(*Config); ok {
			go config.delete()
		} else {
			c.Lock()
			delete(c.store, key)
			c.Unlock()
		}
	}
}

func NewConfigWithInitialValues(ctx context.Context, initialValues map[string]interface{}) *Config {
	config := NewConfig(ctx)
	for key, val := range initialValues {
		config.Set(key, val, true)
	}
	return config
}

func isKeyValid(raw_key string) error {
	if raw_key == "" {
		return errors.Join(ErrKeyInvalid, errors.New(raw_key))
	}
	parts := strings.Split(raw_key, CONFIG_TREE_SEPARATOR)
	for _, part := range parts {
		for _, char := range part {
			if !strings.Contains(KEY_ALLOWED_CHARS, string(char)) {
				return errors.Join(ErrKeyInvalid, errors.New(raw_key))
			}
		}
	}
	return nil
}

func (c *Config) Has(key string) bool {
	// split the key into its components
	split_key := strings.Split(key, CONFIG_TREE_SEPARATOR)

	// keylength of one means end, return bool of key in store
	if len(split_key) == 1 {
		c.RLock()
		keyInStore := c.store[key] != nil
		c.RUnlock()
		return keyInStore
	}

	// keylength of more than one means we need to check the next level
	// if the next level is a config, recurse
	// if the next level is not a config, return false
	if config, ok := c.store[split_key[0]].(*Config); ok {
		return config.Has(strings.Join(split_key[1:], CONFIG_TREE_SEPARATOR))
	}
	return false
}

func (c *Config) Get(key string) (interface{}, error) {
	if err := isKeyValid(key); err != nil { // check key is valid
		return nil, err
	}
	if !c.Has(key) {
		return nil, errors.Join(ErrKeyNotFound, errors.New(key))
	}

	return c.getRecursive(strings.Split(key, CONFIG_TREE_SEPARATOR))
}

func (c *Config) getRecursive(key []string) (interface{}, error) {
	c.RLock()
	defer c.RUnlock()
	cur_key := key[0]
	if len(key) == 1 {
		if val, ok := c.store[cur_key]; ok {
			return val, nil
		}
		return nil, errors.Join(ErrValueNotFound, errors.New(cur_key))
	}

	if config, ok := c.store[cur_key].(*Config); ok {
		return config.getRecursive(key[1:])
	}
	return nil, errors.Join(ErrFieldNotConfig, errors.New(cur_key))
}

func (c *Config) Set(key string, value interface{}, force bool) error {
	if err := isKeyValid(key); err != nil { // check key is valid
		return errors.Join(ErrKeyInvalid, errors.New(key))
	}

	if c.Has(key) && !force {
		return errors.Join(ErrKeyInStore, errors.New(key))
	}

	// log.Println("Setting key:", key, "to value:", fmt.Sprintf("%v", value))

	errChan := make(chan error, 1)

	go func() {
		go c.setRecursive(strings.Split(key, CONFIG_TREE_SEPARATOR), value, force, errChan)
	}()

	select {
	case <-c.ctx.Done():
		return ErrContextDone
	case err, ok := <-errChan:
		if err != nil && ok {
			return err
		}
		return nil
	}
}

func (c *Config) setRecursive(key []string, value interface{}, force bool, errChan chan<- error) error {
	cur_key := key[0]
	// if key is longer than one, we are not at the end of the tree
	if len(key) > 1 {
		config, ok := c.store[cur_key].(*Config)
		if !ok && (c.store[cur_key] == nil || force) {
			c.Lock()
			c.store[cur_key] = NewConfig(c.ctx)
			c.Unlock()
			config = c.store[cur_key].(*Config)
		}
		go config.setRecursive(key[1:], value, force, errChan)
		return nil
	}
	// if key is one, we are at the end of the tree

	// delete operation when value is empty string
	if stringVal, ok := value.(string); ok && stringVal == "" {
		c.Lock()
		delete(c.store, cur_key)
		c.Unlock()
		close(errChan)
		return nil
	}

	if rawConfVal, ok := value.(map[string]interface{}); ok {
		value = NewConfigWithInitialValues(c.ctx, rawConfVal)
	}

	// value should be primitive/like
	c.Lock()
	c.store[cur_key] = value
	c.Unlock()

	close(errChan)
	return nil
}

// check if config has all keys in default map
func (c *Config) CompareDefault(defaultMap map[string]interface{}) error {
	for key := range defaultMap {
		if !c.Has(key) {
			return fmt.Errorf("key %s not in default", key)
		}
	}
	return nil
}

func (c *Config) Merge(merger *Config, overwrite bool) error {
	for _, key := range merger.Keys() {
		value, err := merger.Get(key)
		if err != nil {
			return err
		}
		c.RLock()
		baseConfig, ok := c.store[key].(*Config)
		c.RUnlock()
		if !ok {
			// value in base is not config, set with merger value
			if err := c.Set(key, value, overwrite); err != nil {
				if err == ErrKeyInStore && overwrite {
					return ErrMergeFailed
				} else if err == ErrKeyInStore && !overwrite {
					continue
				}
				return err
			}
			continue
		}

		// value in base is config, check if merger value is config
		mergerConfig, ok := value.(*Config)
		if !ok {
			return ErrValueInvalid
		}
		baseConfig.Merge(mergerConfig, overwrite)
	}
	return nil
}

func (c *Config) Copy() *Config {
	config := NewConfig(c.ctx)
	for key, value := range c.store {
		switch entry := value.(type) {
		case *Config:
			config.Set(key, entry.Copy(), true)
		default:
			config.Set(key, entry, true)
		}
	}
	return config
}

func (c *Config) Keys() []string {
	keys := []string{}
	for key := range c.store {
		keys = append(keys, key)
	}
	return keys
}

func (c *Config) GetMap() map[string]interface{} {
	return c.toInitialMap()
}

func (c *Config) toInitialMap() map[string]interface{} {
	initialMap := make(map[string]interface{})
	for key, value := range c.store {
		switch entry := value.(type) {
		case *Config:
			nestedEntry := entry.toInitialMap()
			initialMap[key] = nestedEntry
		default:
			c.Lock()
			initialMap[key] = entry
			c.Unlock()
		}
	}

	return initialMap
}

func (c *Config) toEnv() string {
	var buffer []string
	c.Lock()
	for key, value := range c.store {
		switch entry := value.(type) {
		case *Config:
			c.Unlock()
			nestedEntry := entry.toEnv()
			c.Lock()
			for _, line := range strings.Split(nestedEntry, "\n") {
				log.Printf("adding: %s_%s", key, line)
				buffer = append(buffer, fmt.Sprintf("%s_%s", key, line))
			}
		default:
			log.Printf("adding: %s=%v", key, value)
			buffer = append(buffer, fmt.Sprintf("%s=%v", key, value))
		}
	}
	c.Unlock()
	return strings.Join(buffer, "\n")
}

func (c *Config) Sprint() string {
	c.RLock()
	defer c.RUnlock()
	var buffer strings.Builder
	for k, v := range c.store {
		switch entry := v.(type) {
		case *Config:
			buffer.WriteString(fmt.Sprintf("%s:\n", k))
			for _, line := range strings.Split(entry.Sprint(), "\n") {
				buffer.WriteString(fmt.Sprintf("\t%s\n", line))
			}
			// trim last newline
		default:
			buffer.WriteString(fmt.Sprintf("%s: %v", k, v))
			buffer.WriteString("\n")
		}
	}
	return buffer.String()
}

func (c *Config) Print() {
	fmt.Printf("Config:\n%+v\n", c.Sprint())
}

func (c *Config) DumpToFile(format string, outFile string) error {
	file, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	var formattedConfig string

	switch format {
	case "json":
		bytes, err := json.Marshal(c.toInitialMap())
		if err != nil {
			return err
		}
		formattedConfig = string(bytes)
	case "txt":
		formattedConfig = c.Sprint()
	case "env":
		formattedConfig = c.toEnv()
	default:
		return ErrValueInvalid
	}

	_, err = file.WriteString(formattedConfig)
	return err
}

// ConfigLoaders to load config from different sources, with managed priority

func LoadConfig(envPrefix string, ctx context.Context) (*Config, error) {
	finishChan := make(chan error, 2)
	config := NewConfig(ctx)
	// load from env
	go config.LoadEnv(envPrefix, finishChan)
	// load from file
	// go config.LoadFile(file, finishChan)

	select {
	case <-ctx.Done():
		return nil, ErrContextDone
	case err := <-finishChan:
		return config, err
	}
}

func (c *Config) LoadEnv(envPrefix string, finishChan chan<- error) error {
	waitGroup := &sync.WaitGroup{}
	defer close(finishChan)

	variableStream := getVarStream(envPrefix, waitGroup)
	entryStream := parseVarStream(variableStream, waitGroup)
	var recent error
	for entry := range entryStream {
		if err := c.Set(entry.key, entry.value, true); err != nil {
			finishChan <- err
			recent = err
		}
	}
	waitGroup.Wait()
	return recent
}

func getVarStream(prefix string, wg *sync.WaitGroup) <-chan string {
	variableStream := make(chan string, len(os.Environ()))
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(variableStream)
		wg.Add(len(os.Environ()))
		for _, envVar := range os.Environ() {
			defer wg.Done()
			if !strings.HasPrefix(envVar, prefix) {
				continue
			}
			if strings.Split(envVar, "_")[0] != prefix {
				continue
			}
			variableStream <- strings.TrimPrefix(envVar, prefix+"_")
		}
	}()
	return variableStream
}

func parseVarStream(variableStream <-chan string, wg *sync.WaitGroup) <-chan *Entry {
	entries := make(chan *Entry, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(entries)
		for envVar := range variableStream {
			if val, err := parseEnvVar(envVar); err == nil {
				entries <- val
			} else {
				panic(err)
			}
		}
	}()
	return entries
}

func parseEnvVar(envVar string) (*Entry, error) {
	parts := strings.SplitN(envVar, "=", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid env var")
	}
	key := parts[0]
	value := parts[1]

	if strings.HasSuffix(key, "_FILE") {
		log.Println("Loading config from file: " + value)
		if file, err := os.Open(value); err == nil {
			defer file.Close()
			value = readFromFile(file)
			key = strings.TrimSuffix(key, "_FILE")
		} else {
			return nil, err
		}
	}

	return &Entry{
		key:   strings.Join(strings.Split(key, "_"), CONFIG_TREE_SEPARATOR),
		value: value,
	}, nil
}

func readFromFile(file *os.File) string {
	var buffer strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		buffer.WriteString(scanner.Text())
	}
	return strings.Trim(buffer.String(), "\r\n")
}

func (c *Config) LoadFile(path string) error {
	return nil
}
