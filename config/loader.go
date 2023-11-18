package config

import (
	"bufio"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	ENV_SPLIT_CHAR = "_"
)

var (
	ErrNoConfigSource = errors.New("no config source provided")
)

type Entry struct {
	key   string
	value interface{}
}

// loads config from multiple sources
// currently only supports env with single prefix, is wip
func LoadConfig(ctx context.Context, envPrefixList []string, fileList []string, firstError bool) (*Config, error) {
	if len(envPrefixList) == 0 && len(fileList) == 0 {
		return nil, ErrNoConfigSource
	}

	config := NewConfig()
	errChan := make(chan error, 2)
	entryChan := make(chan *Entry, 32)
	finishChan := make(chan bool, 2)
	// load from env
	finishCounter := 0
	if len(envPrefixList) != 0 {
		go config.LoadEnv(envPrefixList, entryChan, errChan, finishChan)
		finishCounter += 1
	}
	// load from file
	if len(fileList) == 0 {
		for _, file := range fileList {
			go config.LoadFile(file, entryChan, errChan, finishChan)
			finishCounter += 1
		}
	}
	// handle finish
	var joinedErr error
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errChan:
			if err != nil && firstError {
				return nil, err
			} else {
				errors.Join(err, joinedErr)
			}
		case <-finishChan:
			finishCounter--
			if finishCounter == 0 {
				close(errChan)
				close(entryChan)
				close(finishChan)
				return config, joinedErr
			}
		case entry := <-entryChan:
			if strings.HasSuffix(entry.key, CONFIG_TREE_SEPARATOR+"FILE") {
				var err error
				entry, err = handleFileEntry(entry)
				if err != nil {
					log.Println(err)
				}
			}
			if err := config.Set(entry.key, entry.value, true); err != nil {
				log.Println(err)
			}
		}
	}
}

func (c *Config) LoadEnv(envPrefixes []string, entryChan chan<- *Entry, errChan chan<- error, finishChan chan<- bool) {
	waitGroup := &sync.WaitGroup{}
	for _, envVar := range os.Environ() {
		waitGroup.Add(1)
		go func(rawVar string) {
			defer waitGroup.Done()

			println("checking env var: " + rawVar)

			if envPrefix, ok := checkHasPrefix(rawVar, envPrefixes); !ok {
				return
			} else {
				rawVar = strings.TrimPrefix(rawVar, envPrefix+ENV_SPLIT_CHAR)
			}
			if entry, err := parseEnvVar(rawVar); err != nil {
				errChan <- err
			} else if entry != nil {
				entryChan <- entry
			}
		}(envVar)
	}
	waitGroup.Wait()
	finishChan <- true
}

func checkHasPrefix(envVar string, prefixes []string) (string, bool) {
	for _, prefix := range prefixes {
		if strings.HasPrefix(envVar, prefix+ENV_SPLIT_CHAR) {
			return prefix, true
		}
	}
	return "", false
}

func parseEnvVar(envVar string) (*Entry, error) {
	parts := strings.SplitN(envVar, "=", 2)
	if len(parts) != 2 {
		return nil, &ErrValueInvalid{
			value: envVar,
			key:   parts[0],
		}
	}

	key := strings.ReplaceAll(parts[0], ENV_SPLIT_CHAR, CONFIG_TREE_SEPARATOR)
	value := parts[1]

	return &Entry{
		key:   key,
		value: value,
	}, nil
}

func handleFileEntry(entry *Entry) (*Entry, error) {
	entry.key = strings.TrimSuffix(entry.key, CONFIG_TREE_SEPARATOR+"FILE")
	var err error
	if filePath := entry.value.(string); filePath == "" {
		return nil, &ErrValueInvalid{
			key:   entry.key,
			value: entry.value,
		}
	} else if entry.value, err = readFromFile(filePath); err != nil {
		return nil, &ErrValueInvalid{
			key:    entry.key,
			value:  entry.value,
			nested: err,
		}
	}
	return entry, nil
}

func readFromFile(filePath string) (string, error) {
	log.Println("Loading config from file: " + filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var buffer strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		buffer.WriteString(scanner.Text())
	}
	return strings.Trim(buffer.String(), "\r\n"), nil
}

func (c *Config) LoadFile(path string, entryChan chan<- *Entry, errChan chan<- error, finishChan chan<- bool) {
	file, err := os.Open(path)
	if err != nil {
		errChan <- err
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	nestedKey := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") && strings.HasSuffix(line, ":") {
			nestedKey = append(nestedKey, strings.TrimSuffix(line, ":"))
			continue
		}
		if !strings.HasPrefix(line, "\t") || !strings.HasPrefix(line, "    ") {
			nestedKey = []string{}
		}
		if entry, err := parseFileLine(line, nestedKey); err != nil {
			errChan <- err
		} else if entry != nil {
			entryChan <- entry
		}
	}
	finishChan <- true
}

func parseFileLine(line string, nestedKey []string) (*Entry, error) {
	for strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ") {
		line = strings.TrimPrefix(line, "\t")
		line = strings.TrimPrefix(line, "    ")
	}
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil, &ErrValueInvalid{
			value: line,
			key:   parts[0],
		}
	}
	key := parts[0]
	value := parts[1]

	joinedKey := strings.Join(nestedKey, CONFIG_TREE_SEPARATOR) + CONFIG_TREE_SEPARATOR + key

	return &Entry{
		key:   joinedKey,
		value: value,
	}, nil
}
