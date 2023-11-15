package config

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	t.Log("Testing Config Load")
	os.Setenv("PATCHTESTCONF_SIMPLE", "test")
	os.Setenv("PATCHTESTCONF_MAP_TEST", "abcde")
	config, err := LoadConfig("PATCHTESTCONF", context.TODO())
	if err != nil {
		t.Error(err)
	}
	t.Logf("Config:\n%s", config.Sprint())
	if str, err := config.Get("SIMPLE"); err != nil || str.(string) != "test" {
		t.Log(str)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if confMap, err := config.Get("MAP"); err == nil {
		if val, err := confMap.(*Config).GetString("TEST"); err == nil {
			if val != "abcde" {
				t.Log(val)
				t.Log(err)
				t.Error("Config is not loaded correctly (level 3)")
			}
		} else {
			t.Log(val)
			t.Log(err)
			t.Error("Config is not loaded correctly (level 2)")
		}
	} else {
		t.Log(confMap)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}

	if directTest, err := config.GetString("MAP.TEST"); err != nil || directTest != "abcde" {
		t.Log(directTest)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 4)")
	}
}

func TestConfigDelete(t *testing.T) {
	os.Setenv("PATCHTESTCONF_SIMPLE", "test")
	config, err := LoadConfig("PATCHTESTCONF", context.TODO())
	if err != nil {
		t.Error(err)
	}
	t.Logf("Config:\n%v", config.Sprint())
	if str, err := config.Get("SIMPLE"); err != nil || str.(string) != "test" {
		t.Log(str)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if err := config.Set("SIMPLE", "", true); err != nil {
		t.Error(err)
	}

	if str, err := config.Get("SIMPLE"); err == nil || str != nil {
		t.Log(str)
		t.Log(err)
		t.Error("Config is not deleted correctly (level 0)")
	}
}

func TestConfigWithFile(t *testing.T) {
	testValue := "abcdefg1234567!"
	t.Log("Testing Config Load with File")
	if err := os.WriteFile("test_conf.env", []byte(testValue), 0644); err != nil {
		t.Error("Failed to create test file")
		t.FailNow()
	}
	os.Setenv("PATCHTESTCONF_SIMPLE_FILE", "test_conf.env")
	// os.Setenv("PATCHTEST_MAP_FILE", "abcde")
	config, err := LoadConfig("PATCHTESTCONF", context.TODO())
	if err != nil {
		t.Error(err)
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, err := config.Get("SIMPLE"); err != nil || val.(string) != testValue {
		t.Error("Config is not loaded correctly (level 1)")
	}

	if err := os.Remove("test_conf.env"); err != nil {
		t.Error("Failed to remove test file")
	}
}

func TestConfigWithInitialValue(t *testing.T) {
	initialValues := map[string]interface{}{
		"test":                "abcde",
		"nested.string":       "nestedTestValue",
		"example.test.number": 234567,
	}
	t.Log("Testing Config Load with Initial Values")
	// config := NewConfig(initalValues, nil)
	config := NewConfigWithInitialValues(context.TODO(), initialValues)
	if config == nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, err := config.GetString("test"); err != nil || val != "abcde" {
		t.Error("Config is not loaded correctly (direct level 0)")
	}

	// step by step test
	nestedTest, err := config.Get("nested")
	if err != nil {
		t.Log(nestedTest)
		t.Log(err)
		t.FailNow()
	}

	if nestedTest, ok := nestedTest.(*Config); ok {
		if val, err := nestedTest.GetString("string"); err != nil || val != "nestedTestValue" {
			t.Error("Config is not loaded correctly (level 1)")
		}
	} else {
		t.Log(err)
		t.Error("Config has non-parsable sub config")
	}

	if val, err := config.GetInt("example.test.number"); err != nil && val != 234567 {
		t.Error("Config is not loaded correctly (level 2)")
	}
}

func TestConfigWithInitialValueMap(t *testing.T) {
	initialValues := map[string]interface{}{
		"test": "abcde",
		"nested": map[string]interface{}{
			"string": "nestedTestValue",
		},
		"example": map[string]interface{}{
			"test": map[string]interface{}{
				"number": 234567,
			},
		},
	}
	t.Log("Testing Config Load with Initial Values")
	// config := NewConfig(initalValues, nil)
	config := NewConfigWithInitialValues(context.TODO(), initialValues)
	if config == nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, err := config.GetString("test"); err != nil || val != "abcde" {
		t.Error("Config is not loaded correctly (direct level 0)")
	}

	// step by step test
	nestedTest, err := config.Get("nested")
	if err != nil {
		t.Log(nestedTest)
		t.Log(err)
		t.FailNow()
	}

	if nestedTest, ok := nestedTest.(*Config); ok {
		if val, err := nestedTest.GetString("string"); err != nil || val != "nestedTestValue" {
			t.Error("Config is not loaded correctly (level 1)")
		}
	} else {
		t.Log(err)
		t.Error("Config has non-parsable sub config")
	}

	if val, err := config.GetInt("example.test.number"); err != nil && val != 234567 {
		t.Error("Config is not loaded correctly (level 2)")
	}
}

func TestConfigEmptyMerge(t *testing.T) {
	values := map[string]interface{}{
		"val": "abcde",
	}
	config := NewConfigWithInitialValues(context.TODO(), values)
	if val, err := config.GetString("val"); err != nil && val != "abcde" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	emptyEnvConfig, err := LoadConfig("ABCTEST123", context.Background())
	if err != nil {
		t.Log(err)
		t.Error("Config is not loaded")
	}
	if emptyEnvConfig.Has("val") {
		t.Error("Config is not empty")
	}

	if err := config.Merge(emptyEnvConfig, false); err != nil {
		t.Log(err)
		t.Error("Config is not merged correctly")
	}

	if val, err := config.GetString("val"); err != nil && val != "abcde" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}
}

func TestConfigMerge(t *testing.T) {
	initialValues := map[string]interface{}{
		"nested.string": "nestedTestValue",
	}
	initialValuesMergin := map[string]interface{}{
		"val":          "abcde",
		"nested.other": "nestedOther",
	}
	config := NewConfigWithInitialValues(context.TODO(), initialValues)
	configMerger := NewConfigWithInitialValues(context.TODO(), initialValuesMergin)

	if val, err := config.Get("val"); err == nil && val != nil {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if val, err := config.GetString("nested.string"); err != nil && val != "nestedTestValue" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}
	t.Logf("Config:\n%v", config.Sprint())

	err := config.Merge(configMerger, false)
	t.Log(err)

	t.Logf("Config:\n%v", config.Sprint())
	if val, err := config.GetString("val"); err != nil && val != "abcde" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if val, err := config.Get("nested.string"); err != nil && val != "nestedTestValue" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}

	if val, err := config.GetString("nested.other"); err != nil && val != "nestedOther" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}
}

func TestFileDump(t *testing.T) {
	initialValues := map[string]interface{}{
		"simple":        "test",
		"nested.string": "nestedTestValue",
		"otherNest": map[string]interface{}{
			"string":  "nestedOther",
			"boolean": true,
			"number":  123456,
			"tripplenest": map[string]interface{}{
				"string": "nestedOther",
			},
		},
	}

	shouldFileContent := `simple=test
nested_string=nestedTestValue
otherNest_string=nestedOther
otherNest_boolean=true
otherNest_number=123456
otherNest_tripplenest_string=nestedOther`

	config := NewConfigWithInitialValues(context.TODO(), initialValues)
	if err := config.DumpToFile("env", "test_dump.env"); err != nil {
		t.Error(err)
	}

	// check if lines are in file
	var fileLines []string
	if rawFile, err := os.ReadFile("test_dump.env"); err != nil {
		t.Error(err)
	} else {
		fileLines = strings.Split(strings.Trim(string(rawFile), "\n"), "\n")
		slices.Sort(fileLines)
	}

	slice := strings.Split(shouldFileContent, "\n")
	slices.Sort(slice)

	for i, line := range slice {
		if line != fileLines[i] {
			t.Errorf("File content is not correct: '%s' != '%s'", line, fileLines[i])
		} else {
			t.Logf("File content is correct: '%s'", line)
		}
	}

	if err := os.Remove("test_dump.env"); err != nil {
		t.Error(err)
	}
}

func TestCopy(t *testing.T) {
	initialValues := map[string]interface{}{
		"simple":        "test",
		"nested.string": "nestedTestValue",
		"otherNest": map[string]interface{}{
			"string":  "nestedOther",
			"boolean": true,
			"number":  123456,
		},
	}
	config := NewConfigWithInitialValues(context.Background(), initialValues)
	t.Log("original:\n", config.Sprint())
	copy := config.Copy()
	t.Log("copy:\n", copy.Sprint())
	if copy == nil {
		t.Error("Copy is nil")
		t.FailNow()
	}
	// initial check if values are the same
	if val, err := config.GetString("simple"); err != nil || val != "test" {
		t.Error("Config is not loaded correctly (level 0)")
		t.FailNow()
	}
	if val, err := config.GetString("nested.string"); err != nil || val != "nestedTestValue" {
		t.Error("Config is not loaded correctly (level 1)")
		t.FailNow()
	}

	if val, err := copy.GetString("simple"); err != nil || val != "test" {
		t.Error("Copy is not loaded correctly (level 0)")
		t.FailNow()
	}
	if val, err := copy.GetString("nested.string"); err != nil || val != "nestedTestValue" {
		t.Error(err)
		t.Error("Copy is not loaded correctly (level 1)")
		t.FailNow()
	}

	// change value in copy
	if err := copy.Set("simple", "abcde", true); err != nil {
		t.Error(err)
		t.FailNow()
	}
	if err := copy.Set("nested.string", "abcde", true); err != nil {
		t.Error(err)
		t.FailNow()
	}
	// check if value in original is still the same
	if val, err := config.GetString("simple"); err != nil || val != "test" {
		t.Error("Original config modified by copy")
		t.FailNow()
	}
	if val, err := config.GetString("nested.string"); err != nil || val != "nestedTestValue" {
		t.Error("Original config modified by copy")
		t.FailNow()
	}
	t.Log(config.Sprint())

	// check if value in copy is changed
	if val, err := copy.GetString("simple"); err != nil || val != "abcde" {
		t.Error("Copy not modified")
	}
	if val, err := copy.GetString("nested.string"); err != nil || val != "abcde" {
		t.Error("Copy not modified")
	}

	t.Log(copy.Sprint())
}
