package config

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

const (
	CONFIG_TREE_SEPARATOR = "/"
	KEY_ALLOWED_CHARS     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
)

type Config struct {
	loader Loader
	ConfigStore
}

func New(ctx context.Context) (*Config, error) {
	store, err := DefaultConfigStore(ctx)
	if err != nil {
		return nil, err
	}
	config := &Config{
		loader:      &ConfigLoader{},
		ConfigStore: store,
	}
	return config, nil
}

func WithInitialValues(ctx context.Context, initialValues map[string]interface{}) (*Config, error) {
	store, err := DefaultConfigStore(ctx)
	if err != nil {
		return nil, err
	}
	errGroup, eCtx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				errGroup.Go(func() error {
					return &ErrKeyValueInvalid{key: "", value: initialValues, nested: errors.New("panic during initial value set")}
				})
			}
		}()
		recursiveSet(eCtx, store, "", initialValues, errGroup)
		return nil
	})
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}
	config := &Config{
		loader:      &ConfigLoader{},
		ConfigStore: store,
	}
	return config, errGroup.Wait()
}

func recursiveSet(ctx context.Context, store ConfigStore, baseKey string, valueMap map[string]interface{}, errGroup *errgroup.Group) {
	for key, val := range valueMap {
		k := strings.Join([]string{baseKey, key}, CONFIG_TREE_SEPARATOR)
		k = strings.TrimPrefix(k, CONFIG_TREE_SEPARATOR)
		switch v := val.(type) {
		case map[string]interface{}:
			recursiveSet(ctx, store, k, v, errGroup)
		case string:
			errGroup.Go(func() error { return store.Set(ctx, k, v, true) })
		case int:
			errGroup.Go(func() error { return store.Set(ctx, k, strconv.Itoa(v), true) })
		case bool:
			errGroup.Go(func() error { return store.Set(ctx, k, strconv.FormatBool(v), true) })
		case float64:
			errGroup.Go(func() error { return store.Set(ctx, k, strconv.FormatFloat(v, 'f', -1, 64), true) })
		default:
			errGroup.Go(func() error { return &ErrKeyValueInvalid{key: k, value: v} })
		}
	}
}

func WithInitialValuesAndOptions(ctx context.Context, initialValues map[string]interface{}, options *Config) (*Config, error) {
	store, err := DefaultConfigStore(ctx)
	if err != nil {
		return nil, err
	}
	errGroup, eCtx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				errGroup.Go(func() error {
					return &ErrKeyValueInvalid{key: "", value: initialValues, nested: errors.New("panic during initial value set")}
				})
			}
		}()
		recursiveSet(eCtx, store, "", initialValues, errGroup)
		return nil
	})
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}
	config := &Config{
		loader:      &ConfigLoader{},
		ConfigStore: store,
	}

	if err := config.Merge(ctx, options, true); err != nil {
		return nil, err
	}

	return config, nil
}

func NewLoadedConfig(ctx context.Context, envPrefixList []string, fileList []string) (*Config, error) {
	config, err := New(ctx)
	if err != nil {
		return nil, err
	}
	if err := config.Load(ctx, envPrefixList, fileList); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Config) Load(ctx context.Context, envPrefixList []string, fileList []string) error {
	if c.loader == nil {
		return ErrNoConfigSource
	}
	errGroup, eCtx := errgroup.WithContext(ctx)
	errGroup.Go(func() error { return c.loader.LoadEnv(eCtx, c.ConfigStore, envPrefixList) })
	errGroup.Go(func() error { return c.loader.LoadFile(eCtx, c.ConfigStore, fileList) })
	return errGroup.Wait()
}

// filters out simple values and nested values
// returns two maps, one of simple values, one of keys to nested values
// func filterNestedValues(m map[string]interface{}) (map[string]interface{}, map[string][]string) {
// 	simpleMap := make(map[string]interface{})
// 	keyBuffer := make(map[string][]string)
// 	for key, value := range m {
// 		parts := strings.Split(key, CONFIG_TREE_SEPARATOR)
// 		if len(parts) == 1 {
// 			simpleMap[key] = value
// 			continue
// 		}
// 		if keyBuffer[parts[0]] == nil {
// 			keyBuffer[parts[0]] = []string{}
// 		}
// 		keyBuffer[parts[0]] = append(keyBuffer[parts[0]], strings.Join(parts[1:], CONFIG_TREE_SEPARATOR))
// 	}
// 	return simpleMap, keyBuffer
// }

// check if base config has all keys defined in cmp
// if firstError is true, return first error encountered
func (c *Config) Compare(ctx context.Context, cmp *Config, valueCompare bool) error {
	errGroup, eCtx := errgroup.WithContext(ctx)
	for _, key := range cmp.Keys(ctx) {
		k := key
		errGroup.Go(func() error {
			ourValue, err := c.Get(eCtx, k)
			if err != nil {
				return &ErrKeyNotFound{key: k}
			} else if !valueCompare {
				return nil
			}
			theirValue, err := cmp.Get(eCtx, k)
			if err != nil {
				return &ErrKeyNotFound{key: k}
			} else if strings.Compare(ourValue, theirValue) != 0 {
				return &ErrValueMismatch{key: k, expected: theirValue, actual: ourValue}
			}
			return nil
		})
	}
	return errGroup.Wait()
}

// check if base config has all keys defined in cmp
// if firstError is true, return first error encountered
func (c *Config) CompareMap(ctx context.Context, cmpMap map[string]string, valueCompare bool) error {
	errGroup, eCtx := errgroup.WithContext(ctx)
	for key := range cmpMap {
		k := key
		errGroup.Go(func() error {
			value, err := c.Get(eCtx, k)
			if err != nil {
				return &ErrKeyNotFound{key: k}
			} else if !valueCompare {
				return nil
			} else if strings.Compare(value, cmpMap[k]) != 0 {
				return &ErrValueMismatch{key: k, expected: cmpMap[k], actual: value}
			}
			return nil
		})
	}
	return errGroup.Wait()
}

func (c *Config) Merge(ctx context.Context, merger ConfigStore, overwrite bool) error {
	errGroup, eCtx := errgroup.WithContext(ctx)
	for _, key := range merger.Keys(ctx) {
		k := key
		errGroup.Go(func() error {
			if c.Has(eCtx, k) && !overwrite {
				return &ErrKeyInStore{key: k}
			}
			value, err := merger.Get(eCtx, k)
			if err != nil {
				return err
			}
			return c.Set(eCtx, k, value, overwrite)
		})
	}
	return errGroup.Wait()
}

func (c *Config) MergeIn(ctx context.Context, baseKey string, value ConfigStore, force bool) error {
	errGroup, eCtx := errgroup.WithContext(ctx)
	for _, key := range value.Keys(ctx) {
		k := key
		errGroup.Go(func() error {
			joinedKey := baseKey + CONFIG_TREE_SEPARATOR + k
			val, _ := value.Get(eCtx, k)
			return c.Set(eCtx, joinedKey, val, force)
		})
	}
	return errGroup.Wait()
}

func (c *Config) GetConfig(ctx context.Context, key string) (*Config, error) {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)
	if err := IsValidKey(key); err != nil { // check key is valid
		return nil, err
	}
	store, err := DefaultConfigStore(ctx)
	if err != nil {
		return nil, err
	}
	errGroup, eCtx := errgroup.WithContext(ctx)
	for key, val := range c.GetAll(ctx, key) {
		if key == "" {
			// this would be the case if the key is a simple value
			continue
		}
		k := key
		v := val
		errGroup.Go(func() error {
			return store.Set(eCtx, k, v, true)
		})
	}
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}
	return &Config{
		loader:      &ConfigLoader{},
		ConfigStore: store,
	}, nil
}

func (c *Config) Copy(ctx context.Context) (*Config, error) {
	buffer, err := DefaultConfigStore(ctx)
	if err != nil {
		return nil, err
	}
	for _, key := range c.Keys(ctx) {
		value, err := c.Get(ctx, key)
		if err != nil {
			return nil, ErrCopyConfigReason{err}
		}
		buffer.Set(ctx, key, value, true)
	}
	return &Config{
		loader:      &ConfigLoader{},
		ConfigStore: buffer,
	}, nil
}

func (c *Config) Sprint() string {
	buffer := &strings.Builder{}
	ctx := context.Background()
	for _, key := range c.Keys(ctx) {
		val, _ := c.Get(ctx, key)
		buffer.WriteString(key + ": " + val + "\n")
	}
	return buffer.String()
}

func (c *Config) DumpToFile(ctx context.Context, format string, outFile string) error {
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
	case "env":
		formattedConfig = c.toEnv(ctx)
	default:
		return &ErrDumpToFile{reason: errors.New("invalid or unknown format")}
	}

	_, err = file.WriteString(formattedConfig)
	if err != nil {
		return &ErrDumpToFile{reason: err}
	}
	return nil
}

func (c *Config) toEnv(ctx context.Context) string {
	var builder strings.Builder
	for _, key := range c.Keys(ctx) {
		val, err := c.Get(ctx, key)
		if err != nil {
			continue
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(val)
		builder.WriteString("\n")
	}
	return builder.String()
}

func IsValidKey(raw_key string) error {
	if raw_key == "" {
		return &ErrKeyValueInvalid{key: raw_key}
	}
	errGroup := &errgroup.Group{}
	parts := strings.Split(raw_key, CONFIG_TREE_SEPARATOR)
	for _, part := range parts {
		segment := part
		errGroup.Go(func() error { return checkSegment(segment) })
	}
	if err := errGroup.Wait(); err != nil {
		return &ErrKeyValueInvalid{key: raw_key, nested: err}
	}
	return nil
}

func checkSegment(segment string) error {
	if segment == "" {
		return &ErrKeyValueInvalid{key: ""}
	}
	for _, char := range segment {
		if !strings.Contains(KEY_ALLOWED_CHARS, string(char)) {
			return &KeyCharInvalid{char: char, key: segment}
		}
	}
	return nil
}

// func flattenMap(m map[string]interface{}) map[string]interface{} {
// 	flatMap := make(map[string]interface{})
// 	for key, value := range m {
// 		switch entry := value.(type) {
// 		case map[string]interface{}:
// 			for k, v := range flattenMap(entry) {
// 				flatMap[key+CONFIG_TREE_SEPARATOR+k] = v
// 			}
// 		default:
// 			flatMap[key] = value
// 		}
// 	}
// 	return flatMap
// }

// func nestStore(m map[string]interface{}) map[string]interface{} {
// 	nestedMap, nestedKeys := filterNestedValues(m)

// 	// for all collected subkeys, collect into flatmap and nest
// 	bufferMaps := map[string]map[string]interface{}{}
// 	for key, subKeyList := range nestedKeys {
// 		bufferMap := make(map[string]interface{})
// 		for _, subKey := range subKeyList {
// 			bufferMap[subKey] = m[key+CONFIG_TREE_SEPARATOR+subKey]
// 		}
// 		bufferMap = flattenMap(bufferMap)
// 		bufferMaps[key] = bufferMap
// 	}

// 	// nest all collected subkeys
// 	for key, bufferMap := range bufferMaps {
// 		nestedMap[key] = nestStore(bufferMap)
// 	}
// 	return nestedMap
// }
