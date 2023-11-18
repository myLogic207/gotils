package config

import (
	"context"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestConfigNestFlatten(t *testing.T) {
	testConfig := map[string]interface{}{
		"test": "abcde",
		"nested": map[string]interface{}{
			"string": "nestedTestValue",
			"other": map[string]interface{}{
				"number": 123456,
			},
		},
	}
	flattenedConfig := map[string]interface{}{
		"test":                "abcde",
		"nested/string":       "nestedTestValue",
		"nested/other/number": 123456,
	}
	if flatMap := flattenMap(testConfig); !maps.Equal(flatMap, flattenedConfig) {
		t.Log(flatMap)
		t.Log(flattenedConfig)
		t.Error("Config is not flattened correctly")
	}
	nestedMap := nestStore(flattenedConfig)
	if unflatAgain := flattenMap(nestedMap); !maps.Equal(unflatAgain, flattenedConfig) {
		t.Log(unflatAgain)
		t.Log(flattenedConfig)
		t.Error("Config is not flattened correctly")
	}
}

func TestConfigLoad(t *testing.T) {
	t.Log("Testing Config Load")
	os.Setenv("PATCHTESTCONF_SIMPLE", "test")
	os.Setenv("PATCHTESTCONF_MAP_TEST", "abcde")
	config, err := LoadConfig(context.TODO(), []string{"PATCHTESTCONF"}, nil, true)
	if err != nil {
		t.Error(err)
	}
	t.Logf("Config:\n%s", config.Sprint())
	if str, err := config.Get("sImpLe"); err != nil || str.(string) != "test" {
		t.Log(str)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if confMap, err := config.GetConfig("MaP"); err != nil {
		t.Log(confMap)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	} else {
		if val, err := confMap.GetString("tEsT"); err != nil {
			t.Log(val)
			t.Log(err)
			t.Error("Config is not loaded correctly (level 2)")
		} else {
			if val != "abcde" {
				t.Log(val)
				t.Log(err)
				t.Error("Config is not loaded correctly (level 3)")
			}
		}
	}

	if directTest, err := config.GetString("MaP/teST"); err != nil || directTest != "abcde" {
		t.Log(directTest)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 4)")
	}
}

func TestConfigDelete(t *testing.T) {
	os.Setenv("PATCHTESTCONF_SIMPLE", "test")
	config, err := LoadConfig(context.TODO(), []string{"PATCHTESTCONF"}, nil, true)
	if err != nil {
		t.Error(err)
	}
	t.Logf("Config:\n%v", config.Sprint())
	if str, err := config.Get("SIMPLE"); err != nil || str.(string) != "test" {
		t.Log(str)
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if err := config.Set("SIMPLE", nil, true); err != nil {
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
	config, err := LoadConfig(context.TODO(), []string{"PATCHTESTCONF"}, nil, true)
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
		"nested/string":       "nestedTestValue",
		"example/test/number": 234567,
	}
	t.Log("Testing Config Load with Initial Values")
	// config := NewConfig(initalValues, nil)
	config := NewConfigWithInitialValues(initialValues)
	if config == nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, err := config.GetString("test"); err != nil || val != "abcde" {
		t.Error("Config is not loaded correctly (direct level 0)")
	}

	// step by step test
	nestedTest, err := config.GetConfig("nested")
	if err != nil {
		t.Log(nestedTest)
		t.Log(err)
		t.FailNow()
	}

	if val, err := nestedTest.GetString("string"); err != nil || val != "nestedTestValue" {
		t.Log(val)
		t.Log(nestedTest)
		t.Error("Config is not loaded correctly (level 1)")
		t.Log(err)
	}

	if val, err := config.GetInt("example/test/number"); err != nil && val != 234567 {
		t.Log(err)
		t.Log(val)
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
	config := NewConfigWithInitialValues(initialValues)
	if config == nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, err := config.GetString("test"); err != nil || val != "abcde" {
		t.Error("Config is not loaded correctly (direct level 0)")
	}

	// step by step test
	nestedTest, err := config.GetConfig("nested")
	if err != nil {
		t.Log(nestedTest)
		t.Log(err)
		t.FailNow()
	}

	if val, err := nestedTest.GetString("string"); err != nil || val != "nestedTestValue" {
		t.Error("Config is not loaded correctly (level 1)")
	}
	if val, err := config.GetInt("example/test/number"); err != nil && val != 234567 {
		t.Error("Config is not loaded correctly (level 2)")
	}
}

func TestConfigEmptyMerge(t *testing.T) {
	values := map[string]interface{}{
		"val": "abcde",
	}
	config := NewConfigWithInitialValues(values)
	if val, err := config.GetString("val"); err != nil && val != "abcde" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	emptyEnvConfig := NewConfig()
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
		"nested/string/triple": "nestedTestValue",
		"otherNest": map[string]interface{}{
			"boolean": true,
		},
	}
	initialValuesMergin := map[string]interface{}{
		"val":                 "abcde",
		"nested/string/other": "nestedOther",
		"otherNest": map[string]interface{}{
			"number": 123456,
		},
	}
	config := NewConfigWithInitialValues(initialValues)
	configMerger := NewConfigWithInitialValues(initialValuesMergin)

	if val, err := config.Get("val"); err == nil && val != nil {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if val, err := config.GetString("nested/string/triple"); err != nil && val != "nestedTestValue" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}

	err := config.Merge(configMerger, false)
	if err != nil {
		t.Log(err)
		t.Error("Config is not merged correctly")
		t.FailNow()
	}

	if val, err := config.GetString("val"); err != nil && val != "abcde" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}

	if val, err := config.GetInt("otherNest/number"); err != nil && val != 123456 {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 2)")
		t.FailNow()
	}

	if ok, err := config.GetBool("otherNest/boolean"); err != nil && !ok {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 2)")
		t.FailNow()
	}
	if val, err := config.Get("nested/string/triple"); err != nil && val != "nestedTestValue" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 3)")
		t.FailNow()
	}

	if val, err := config.GetString("nested/string/other"); err != nil && val != "nestedOther" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 3)")
		t.FailNow()
	}
}

func TestFileDump(t *testing.T) {
	initialValues := map[string]interface{}{
		"simple":        "test",
		"nested/string": "nestedTestValue",
		"otherNest": map[string]interface{}{
			"string":  "nestedOther",
			"boolean": true,
			"number":  123456,
			"tripplenest": map[string]interface{}{
				"string": "nestedOther",
			},
		},
	}

	shouldFileContent := `SIMPLE=test
NESTED/STRING=nestedTestValue
OTHERNEST/STRING=nestedOther
OTHERNEST/BOOLEAN=true
OTHERNEST/NUMBER=123456
OTHERNEST/TRIPPLENEST/STRING=nestedOther`

	config := NewConfigWithInitialValues(initialValues)
	if err := config.DumpToFile("env", "test_dump.env"); err != nil {
		t.Error(err)
		t.FailNow()
	}

	// check if lines are in file
	var fileLines []string
	if rawFile, err := os.ReadFile("test_dump.env"); err != nil {
		t.Error(err)
	} else {
		fileLines = strings.Split(string(rawFile), "\n")
		slices.Sort(fileLines)
	}

	slice := strings.Split(shouldFileContent, "\n")
	slices.Sort(slice)

	for i, line := range slice {
		if strings.Compare(line, fileLines[i]) != 0 {
			t.Errorf("File content is not correct:\n'%s' != '%s'", line, fileLines[i])
		} else {
			t.Log("File content is correct:\n", line)
		}
	}

	if err := os.Remove("test_dump.env"); err != nil {
		t.Error(err)
	}
}

func TestCopy(t *testing.T) {
	initialValues := map[string]interface{}{
		"simple":        "test",
		"nested/string": "nestedTestValue",
		"otherNest": map[string]interface{}{
			"string":  "nestedOther",
			"boolean": true,
			"number":  123456,
		},
	}
	config := NewConfigWithInitialValues(initialValues)
	t.Log("original:\n", config.Sprint())
	copy := config.Copy()
	t.Log("copy:\n", copy.Sprint())
	if copy == nil {
		t.Error("Copy is nil")
		t.FailNow()
	}
	// initial check if values are the same
	if val, err := config.GetString("simple"); err != nil || val != "test" {
		t.Error(err)
		t.Error("Config is not loaded correctly (level 0)")
		t.FailNow()
	}
	if val, err := config.GetString("nested/string"); err != nil || val != "nestedTestValue" {
		t.Error(err)
		t.Error("Config is not loaded correctly (level 1)")
		t.FailNow()
	}

	if val, err := copy.GetString("simple"); err != nil || val != "test" {
		t.Error("Copy is not loaded correctly (level 0)")
		t.FailNow()
	}
	if val, err := copy.GetString("nested/string"); err != nil || val != "nestedTestValue" {
		t.Error(err)
		t.Error("Copy is not loaded correctly (level 1)")
		t.FailNow()
	}

	// change value in copy
	if err := copy.Set("simple", "abcde", true); err != nil {
		t.Error(err)
		t.FailNow()
	}
	if err := copy.Set("nested/string", "abcde", true); err != nil {
		t.Error(err)
		t.FailNow()
	}
	// check if value in original is still the same
	if val, err := config.GetString("simple"); err != nil || val != "test" {
		t.Error("Original config modified by copy")
		t.FailNow()
	}
	if val, err := config.GetString("nested/string"); err != nil || val != "nestedTestValue" {
		t.Error("Original config modified by copy")
		t.FailNow()
	}
	t.Log(config.Sprint())

	// check if value in copy is changed
	if val, err := copy.GetString("simple"); err != nil || val != "abcde" {
		t.Error("Copy not modified")
	}
	if val, err := copy.GetString("nested/string"); err != nil || val != "abcde" {
		t.Error("Copy not modified")
	}

	t.Log(copy.Sprint())
}

func TestConfigCompare(t *testing.T) {
	baseConfig := NewConfigWithInitialValues(map[string]interface{}{
		"test":          "abcde",
		"simple":        "test",
		"nested/string": "nestedTestValue",
		"otherNest": map[string]interface{}{
			"string": "nestedOther",
		},
	})
	otherConfig := NewConfigWithInitialValues(map[string]interface{}{
		"simple":        "test",
		"nested/string": "nestedTestValue",
	})
	if err := baseConfig.Compare(otherConfig, true); err != nil {
		t.Error("Config compare is not working correctly")

		t.Log(err)

		t.Log(err)
		t.FailNow()
	}
	if err := baseConfig.Compare(otherConfig, false); err != nil {
		t.Error("Config compare is not working correctly")
		t.Log("error was: ", err)
		t.FailNow()
	}
	if err := otherConfig.Compare(baseConfig, true); err == nil {
		t.Error("Config compare is not working correctly")
		t.Log(err)
		t.FailNow()
	}
	if err := otherConfig.Compare(baseConfig, false); err == nil {
		t.Error("Config compare is not working correctly")
		t.Log(err)
		t.FailNow()
	}
}
