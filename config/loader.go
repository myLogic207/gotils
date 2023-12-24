package config

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"strings"

	"golang.org/x/sync/errgroup"
)

const (
	ENV_SPLIT_CHAR = "_"
	ENTRY_SPLIT    = "="
)

type Loader interface {
	LoadEnv(ctx context.Context, store ConfigStore, prefixList []string) error
	LoadFile(ctx context.Context, store ConfigStore, paths []string) error
}

type ConfigLoader struct{}

func (cl *ConfigLoader) LoadEnv(ctx context.Context, store ConfigStore, prefixList []string) error {
	eg, eCtx := errgroup.WithContext(ctx)

	for _, envVar := range os.Environ() {
		ev := envVar
		eg.Go(func() error {
			key, val, err := parseEnvVar(ev, prefixList)
			if err != nil {
				return &ErrParsingEnvVar{err}
			}
			return store.Set(eCtx, key, val, false)
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func parseEnvVar(envVar string, prefixList []string) (string, string, error) {
	var foundPrefix string
	for _, prefix := range prefixList {
		if !strings.HasPrefix(envVar, prefix) {
			continue
		} else {
			// matching prefix found
			foundPrefix = prefix
			break
		}
	}
	if foundPrefix == "" {
		return "", "", nil
	}
	return handleEntry(strings.TrimPrefix(envVar, foundPrefix+ENV_SPLIT_CHAR), ENTRY_SPLIT)
}

func (cl *ConfigLoader) LoadFile(ctx context.Context, store ConfigStore, filePaths []string) error {
	eg, eCtx := errgroup.WithContext(ctx)
	for _, filePath := range filePaths {
		fp := filePath
		eg.Go(func() error {
			return cl.loadFile(eCtx, eg, fp, store)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (cl *ConfigLoader) loadFile(ctx context.Context, errGroup *errgroup.Group, filePath string, store ConfigStore) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Text()
		errGroup.Go(func() error {
			key, value, err := parseFileLine(line)
			if err != nil {
				return err
			}
			return store.Set(ctx, key, value, false)
		})
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil

}

func parseFileLine(line string) (string, string, error) {
	if strings.HasPrefix(line, "#") {
		return "", "", nil
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	var split string
	if strings.Contains(line, ENTRY_SPLIT) {
		split = ENTRY_SPLIT
	} else if strings.Contains(line, ": ") {
		split = ": "
	} else {
		return "", "", &ErrKeyValueInvalid{
			key: line,
		}
	}
	return handleEntry(line, split)
}

func handleEntry(rawString string, split string) (string, string, error) {
	parts := strings.SplitN(rawString, split, 2)
	if len(parts) != 2 {
		return "", "", &ErrKeyValueInvalid{
			value: rawString,
			key:   parts[0],
		}
	}

	key := strings.ReplaceAll(parts[0], ENV_SPLIT_CHAR, CONFIG_TREE_SEPARATOR)
	value := parts[1]
	var err error
	if strings.HasSuffix(key, CONFIG_TREE_SEPARATOR+"FILE") {
		key = strings.TrimSuffix(key, CONFIG_TREE_SEPARATOR+"FILE")
		v, err := handleFileEntry(value)
		if err != nil {
			return "", "", err
		}
		value = string(v)
	}

	return key, value, err
}

func handleFileEntry(path string) ([]byte, error) {
	var err error
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buffer bytes.Buffer
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		buffer.Write(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	bytes := buffer.Bytes()
	return bytes, nil
}

// func nestedMapToString(m map[string]interface{}) string {
// 	buffer := strings.Builder{}
// 	for key, value := range m {
// 		switch entry := value.(type) {
// 		case map[string]interface{}:
// 			buffer.WriteString(fmt.Sprintf("%s:\n", key))
// 			for _, line := range strings.Split(nestedMapToString(entry), "\n") {
// 				buffer.WriteString(fmt.Sprintf("\t%s\n", line))
// 			}
// 		default:
// 			buffer.WriteString(fmt.Sprintf("%s: %v\n", key, value))
// 		}
// 	}
// 	return buffer.String()
// }
