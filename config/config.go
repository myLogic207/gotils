package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	CONFIG_TREE_SEPARATOR = "/"
	KEY_ALLOWED_CHARS     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
)

type Config interface {
	Has(key string) bool
	Get(key string) (interface{}, error)
	GetInt(key string) (int, error)
	GetString(key string) (string, error)
	GetBool(key string) (bool, error)
	GetFloat(key string) (float64, error)
	GetConfig(key string) (Config, error)
	GetDuration(key string) (time.Duration, error)
	Set(key string, value interface{}, force bool) error
	CompareDefault(cmp map[string]interface{}) error
	Compare(cmp Config, valueCompare bool) error
	Merge(merger Config, overwrite bool) error
	Copy() Config
	Keys() []string
	GetMap() map[string]interface{}
	Sprint() string
	DumpToFile(format string, outFile string) error
}

// config is an infinitely nested tree of Entries defined above
// keys are strings, ints, floats, or bools, arrays of these types, or maps(named) of these types
type ConfigStore struct {
	sync.RWMutex
	Config
	store map[string]interface{}
}

func NewConfig() *ConfigStore {
	config := ConfigStore{
		store: make(map[string]interface{}),
	}
	return &config
}

func NewConfigWithInitialValues(initialValues map[string]interface{}) *ConfigStore {
	config := NewConfig()
	for key, val := range flattenMap(initialValues) {
		config.Set(key, val, true)
	}
	return config
}

func flattenMap(m map[string]interface{}) map[string]interface{} {
	flatMap := make(map[string]interface{})
	for key, value := range m {
		switch entry := value.(type) {
		case map[string]interface{}:
			for k, v := range flattenMap(entry) {
				flatMap[key+CONFIG_TREE_SEPARATOR+k] = v
			}
		default:
			flatMap[key] = value
		}
	}
	return flatMap
}

func nestStore(m map[string]interface{}) map[string]interface{} {
	nestedMap, nestedKeys := filterNestedValues(m)

	// for all collected subkeys, collect into flatmap and nest
	bufferMaps := map[string]map[string]interface{}{}
	for key, subKeyList := range nestedKeys {
		bufferMap := make(map[string]interface{})
		for _, subKey := range subKeyList {
			bufferMap[subKey] = m[key+CONFIG_TREE_SEPARATOR+subKey]
		}
		bufferMap = flattenMap(bufferMap)
		bufferMaps[key] = bufferMap
	}

	// nest all collected subkeys
	for key, bufferMap := range bufferMaps {
		nestedMap[key] = nestStore(bufferMap)
	}
	return nestedMap
}

// filters out simple values and nested values
// returns two maps, one of simple values, one of keys to nested values
func filterNestedValues(m map[string]interface{}) (map[string]interface{}, map[string][]string) {
	simpleMap := make(map[string]interface{})
	keyBuffer := make(map[string][]string)
	for key, value := range m {
		parts := strings.Split(key, CONFIG_TREE_SEPARATOR)
		if len(parts) == 1 {
			simpleMap[key] = value
			continue
		}
		if keyBuffer[parts[0]] == nil {
			keyBuffer[parts[0]] = []string{}
		}
		keyBuffer[parts[0]] = append(keyBuffer[parts[0]], strings.Join(parts[1:], CONFIG_TREE_SEPARATOR))
	}
	return simpleMap, keyBuffer
}

func isKeyValid(raw_key string) error {
	if raw_key == "" {
		return &ErrKeyInvalid{key: raw_key}
	}
	waitGroup := &sync.WaitGroup{}
	errChan := make(chan error, len(raw_key))
	parts := strings.Split(raw_key, CONFIG_TREE_SEPARATOR)
	for _, part := range parts {
		waitGroup.Add(1)
		go func(segment string) {
			defer waitGroup.Done()
			if err := checkSegment(segment); err != nil {
				errChan <- err
			}
		}(part)
	}
	waitGroup.Wait()
	close(errChan)
	for err, ok := <-errChan; ok; err, ok = <-errChan {
		if err != nil {
			return &ErrKeyInvalid{key: raw_key, nested: err}
		}
	}
	return nil
}

func checkSegment(segment string) error {
	if segment == "" {
		return &ErrKeyInvalid{key: ""}
	}
	for _, char := range segment {
		if !strings.Contains(KEY_ALLOWED_CHARS, string(char)) {
			return &KeyCharInvalid{char: char, key: segment}
		}
	}
	return nil
}

func (c *ConfigStore) Has(key string) bool {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)
	if err := isKeyValid(key); err != nil { // check key is valid
		return false
	}
	c.RLock()
	defer c.RUnlock()
	for k := range c.store {
		if strings.HasPrefix(k, key) {
			return true
		}
	}
	return false
}

func (c *ConfigStore) Get(key string) (interface{}, error) {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)
	if err := isKeyValid(key); err != nil { // check key is valid
		return nil, err
	}
	if !c.Has(key) {
		return nil, &ErrKeyNotFound{key: key}
	}
	matchedValues := c.getMatching(key)
	if matchedValues[""] != nil {
		return matchedValues[""], nil
	}
	return nestStore(matchedValues), nil
}

func (c *ConfigStore) getMatching(key string) map[string]interface{} {
	values := make(map[string]interface{})
	c.RLock()
	defer c.RUnlock()
	for k, v := range c.store {
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

func (c *ConfigStore) Set(key string, value interface{}, force bool) error {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)
	var flatMapVal map[string]interface{}
	switch entry := value.(type) {
	case *ConfigStore:
		flatMapVal = entry.copyStore()
	case map[string]interface{}:
		flatMapVal = flattenMap(entry)
	case string:
		raw := strings.Trim(entry, " ")
		c.set(key, raw)
		return nil
	default:
		if err := isKeyValid(key); err != nil { // check key is valid
			return err
		}
		c.set(key, value)
		return nil
	}

	for k, val := range flatMapVal {
		if err := isKeyValid(k); err != nil { // check key is valid
			return err
		}
		joinedKey := key + CONFIG_TREE_SEPARATOR + k

		if c.store[joinedKey] != nil && !force { // check key is not in store
			return &ErrKeyInStore{key: joinedKey}
		}
		c.set(joinedKey, val)
	}

	return nil
}

// set is used to directly set a key in the config store, without checking for validity
func (c *ConfigStore) set(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()
	if value == nil {
		delete(c.store, key) // delete key if value is nil or empty
	} else if raw, ok := value.(string); ok && raw == "" {
		delete(c.store, key)
	} else {
		c.store[key] = value
	}
}

func (c *ConfigStore) CompareDefault(cmp map[string]interface{}) error {
	flatCmp := flattenMap(cmp)
	errChan := make(chan error, len(flatCmp))
	go c.compare(flatCmp, false, errChan)
	return handleFinishStream(errChan)
}

// check if base config has all keys defined in cmp
// if firstError is true, return first error encountered
func (c *ConfigStore) Compare(cmp Config, valueCompare bool) error {
	cmpMap := flattenMap(cmp.GetMap())

	errChan := make(chan error, len(cmpMap))
	// check if all keys in cmp are in base
	go c.compare(cmpMap, valueCompare, errChan)

	// handle errors
	return handleFinishStream(errChan)
}

// recursive compare finds all keys in cmp and checks if they are in base
// reports all keys in cmp that are not in base as err to errChan
// terminates when terminationStream is closed
func (c *ConfigStore) compare(cmpMap map[string]interface{}, valueCompare bool, errChan chan<- error) {
	waitGroup := &sync.WaitGroup{}
	for key := range cmpMap {
		waitGroup.Add(1)
		go func(key string) {
			defer waitGroup.Done()
			c.RLock()
			if c.store[key] == nil {
				errChan <- &ErrKeyNotFound{key: key}
			} else if valueCompare && c.store[key] != cmpMap[key] {
				errChan <- &ErrValueInvalid{key: key, value: cmpMap[key]}
			}
			c.RUnlock()
		}(key)
	}
	waitGroup.Wait()
	close(errChan)
}

func handleFinishStream(errChan <-chan error) error {
	for err, ok := <-errChan; ok; err, ok = <-errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ConfigStore) Merge(merger Config, overwrite bool) error {
	for key, value := range merger.GetMap() {
		if c.Has(key) && !overwrite {
			return &ErrKeyInStore{key: key}
		}
		c.set(key, value)
	}
	return nil
}

func (c *ConfigStore) Copy() Config {
	storeCopy := c.copyStore()
	return NewConfigWithInitialValues(storeCopy)
}

func (c *ConfigStore) copyStore() map[string]interface{} {
	mapCopy := make(map[string]interface{})
	for key, value := range c.store {
		mapCopy[key] = value
	}
	return mapCopy
}

func (c *ConfigStore) Keys() []string {
	keys := []string{}
	for key := range c.store {
		keys = append(keys, key)
	}
	return keys
}

func (c *ConfigStore) GetMap() map[string]interface{} {
	return c.copyStore()
}

func (c *ConfigStore) toEnv() string {
	var buffer []string
	c.Lock()
	for key, value := range c.store {
		buffer = append(buffer, fmt.Sprintf("%s=%v", key, value))
	}
	c.Unlock()
	return strings.Join(buffer, "\n")
}

func (c *ConfigStore) Sprint() string {
	storeCopy := c.copyStore()
	unflat := nestStore(storeCopy)
	return nestedMapToString(unflat)
}

func nestedMapToString(m map[string]interface{}) string {
	buffer := strings.Builder{}
	for key, value := range m {
		switch entry := value.(type) {
		case map[string]interface{}:
			buffer.WriteString(fmt.Sprintf("%s:\n", key))
			for _, line := range strings.Split(nestedMapToString(entry), "\n") {
				buffer.WriteString(fmt.Sprintf("\t%s\n", line))
			}
		default:
			buffer.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
	}
	return buffer.String()
}

func (c *ConfigStore) DumpToFile(format string, outFile string) error {
	file, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return &ErrDumpToFile{reason: err}
	}
	defer file.Close()
	var formattedConfig string

	switch format {
	// case "json":
	// 	bytes, err := json.Marshal(c.toInitialMap())
	// 	if err != nil {
	// 		return err
	// 	}
	// 	formattedConfig = string(bytes)
	case "txt":
		formattedConfig = c.Sprint()
	case "env":
		formattedConfig = c.toEnv()
	default:
		return &ErrDumpToFile{reason: errors.New("invalid or unknown format")}
	}

	_, err = file.WriteString(formattedConfig)
	if err != nil {
		return &ErrDumpToFile{reason: err}
	}
	return nil
}
