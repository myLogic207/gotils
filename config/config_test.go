package config

import (
	"context"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// func TestConfigNestFlatten(t *testing.T) {
// 	testConfig := map[string]interface{}{
// 		"test": "abcde",
// 		"nested": map[string]interface{}{
// 			"string": "nestedTestValue",
// 			"other": map[string]interface{}{
// 				"number": 123456,
// 			},
// 		},
// 	}
// 	flattenedConfig := map[string]interface{}{
// 		"test":                "abcde",
// 		"nested/string":       "nestedTestValue",
// 		"nested/other/number": 123456,
// 	}
// 	if flatMap := flattenMap(testConfig); !maps.Equal(flatMap, flattenedConfig) {
// 		t.Log(flatMap)
// 		t.Log(flattenedConfig)
// 		t.Error("Config is not flattened correctly")
// 	}
// 	nestedMap := nestStore(flattenedConfig)
// 	if unflatAgain := flattenMap(nestedMap); !maps.Equal(unflatAgain, flattenedConfig) {
// 		t.Log(unflatAgain)
// 		t.Log(flattenedConfig)
// 		t.Error("Config is not flattened correctly")
// 	}
// }

func TestConfigWithInitialValue(t *testing.T) {
	initialValues := map[string]interface{}{
		"test":                "abcde",
		"nested/string":       "nestedTestValue",
		"example/test/number": 234567,
	}
	t.Log("Testing Config Load with Initial Values")
	// config := NewConfig(initalValues, nil)
	ctx := context.TODO()
	config, err := WithInitialValues(ctx, initialValues)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, err := config.Get(ctx, "test"); err != nil || val != "abcde" {
		t.Fatal("Config is not loaded correctly (direct level 0)")
	}

	// step by step test
	nestedTest, err := config.GetConfig(ctx, "nested")
	if err != nil {
		t.Log(nestedTest.Sprint())
		t.Fatal(err)
	}

	if val, err := nestedTest.Get(ctx, "string"); err != nil || val != "nestedTestValue" {
		t.Log(val)
		t.Log(nestedTest)
		t.Fatal(err)
	}

	if val, err := config.Get(ctx, "example/test/number"); err != nil {
		t.Fatal(err)
	} else if v, err := strconv.Atoi(val); v != 234567 {
		t.Fatal(err)
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
	ctx := context.TODO()
	config, err := WithInitialValues(ctx, initialValues)
	if err != nil {
		t.Error("Config is nil")
	}
	t.Logf("Config:\n%v", config.Sprint())
	if val, err := config.Get(ctx, "test"); err != nil || val != "abcde" {
		t.Error("Config is not loaded correctly (direct level 0)")
	}

	// step by step test
	nestedTest, err := config.GetConfig(ctx, "nested")
	if err != nil {
		t.Log(nestedTest)
		t.Log(err)
		t.FailNow()
	}

	if val, err := nestedTest.Get(ctx, "string"); err != nil || val != "nestedTestValue" {
		t.Error("Config is not loaded correctly (level 1)")
	}
	if val, err := config.Get(ctx, "example/test/number"); err != nil && val != "234567" {
		t.Error("Config is not loaded correctly (level 2)")
	}
}

func TestConfigEmptyMerge(t *testing.T) {
	values := map[string]interface{}{
		"val": "abcde",
	}
	ctx1 := context.TODO()
	config, err := WithInitialValues(ctx1, values)
	if err != nil {
		t.Fatal(err)
	}
	if val, err := config.Get(ctx1, "val"); err != nil && val != "abcde" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	ctx2 := context.TODO()
	emptyEnvConfig, err := New(ctx2)
	if err != nil {
		t.Fatal(err)
	}

	if emptyEnvConfig.Has(ctx2, "val") {
		t.Error("Config is not empty")
	}

	if err := config.Merge(ctx1, emptyEnvConfig, false); err != nil {
		t.Log(err)
		t.Error("Config is not merged correctly")
	}

	if val, err := config.Get(ctx1, "val"); err != nil && val != "abcde" {
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
	ctx1 := context.TODO()
	config, err := WithInitialValues(ctx1, initialValues)
	if err != nil {
		t.Fatal(err)
	}
	ctx2 := context.TODO()
	configMerger, err := WithInitialValues(ctx2, initialValuesMergin)
	if err != nil {
		t.Fatal(err)
	}
	if val, err := config.Get(ctx1, "val"); err == nil && val != "" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 0)")
	}

	if val, err := config.Get(ctx1, "nested/string/triple"); err != nil && val != "nestedTestValue" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}

	if err := config.Merge(ctx1, configMerger, false); err != nil {
		t.Log(err)
		t.Error("Config is not merged correctly")
		t.FailNow()
	}

	if val, err := config.Get(ctx1, "val"); err != nil && val != "abcde" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 1)")
	}

	if val, err := config.Get(ctx1, "otherNest/number"); err != nil && val != "123456" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 2)")
		t.FailNow()
	}

	if ok, err := config.Get(ctx1, "otherNest/boolean"); err != nil {
		t.Fatal(err)
	} else if ok, err := strconv.ParseBool(ok); err != nil || !ok {
		t.Error("Config is not loaded correctly (level 2)")
		t.Fatal(err)
	}
	if val, err := config.Get(ctx1, "nested/string/triple"); err != nil && val != "nestedTestValue" {
		t.Log(err)
		t.Error("Config is not loaded correctly (level 3)")
		t.FailNow()
	}

	if val, err := config.Get(ctx1, "nested/string/other"); err != nil && val != "nestedOther" {
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
OTHERNEST/TRIPPLENEST/STRING=nestedOther
`

	ctx := context.TODO()
	config, err := WithInitialValues(ctx, initialValues)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if err := config.DumpToFile(ctx, "env", "test_dump.env"); err != nil {
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
	ctx1 := context.TODO()
	config, err := WithInitialValues(ctx1, initialValues)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("original:\n", config.Sprint())
	ctx2 := context.TODO()
	copy, err := config.Copy(ctx2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("copy:\n", copy.Sprint())
	if copy == nil {
		t.Fatal("Copy is nil")
	}
	// initial check if values are the same
	if val, err := config.Get(ctx1, "simple"); err != nil || val != "test" {
		t.Error(err)
		t.Error("Config is not loaded correctly (level 0)")
		t.FailNow()
	}
	if val, err := config.Get(ctx1, "nested/string"); err != nil || val != "nestedTestValue" {
		t.Error(err)
		t.Error("Config is not loaded correctly (level 1)")
		t.FailNow()
	}

	if val, err := copy.Get(ctx2, "simple"); err != nil || val != "test" {
		t.Error("Copy is not loaded correctly (level 0)")
		t.FailNow()
	}
	if val, err := copy.Get(ctx2, "nested/string"); err != nil || val != "nestedTestValue" {
		t.Error(err)
		t.Error("Copy is not loaded correctly (level 1)")
		t.FailNow()
	}

	// change value in copy
	if err := copy.Set(ctx1, "simple", "abcde", true); err != nil {
		t.Error(err)
		t.FailNow()
	}
	if err := copy.Set(ctx1, "nested/string", "abcde", true); err != nil {
		t.Error(err)
		t.FailNow()
	}
	// check if value in original is still the same
	if val, err := config.Get(ctx1, "simple"); err != nil || val != "test" {
		t.Error("Original config modified by copy")
		t.FailNow()
	}
	if val, err := config.Get(ctx1, "nested/string"); err != nil || val != "nestedTestValue" {
		t.Error("Original config modified by copy")
		t.FailNow()
	}
	t.Log(config.Sprint())

	// check if value in copy is changed
	if val, err := copy.Get(ctx1, "simple"); err != nil || val != "abcde" {
		t.Error("Copy not modified")
	}
	if val, err := copy.Get(ctx1, "nested/string"); err != nil || val != "abcde" {
		t.Error("Copy not modified")
	}

	t.Log(copy.Sprint())
}

func TestConfigCompare(t *testing.T) {
	ctx1 := context.TODO()
	baseConfig, err := WithInitialValues(ctx1, map[string]interface{}{
		"test":          "abcde",
		"simple":        "test",
		"nested/string": "nestedTestValue",
		"otherNest": map[string]interface{}{
			"string": "nestedOther",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx2 := context.TODO()
	otherConfig, err := WithInitialValues(ctx2, map[string]interface{}{
		"simple":        "test",
		"nested/string": "nestedTestValue",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := baseConfig.Compare(ctx1, otherConfig, true); err != nil {
		t.Error("Config compare is not working correctly")

		t.Log(err)

		t.Log(err)
		t.FailNow()
	}
	if err := baseConfig.Compare(ctx1, otherConfig, false); err != nil {
		t.Error("Config compare is not working correctly")
		t.Log("error was: ", err)
		t.FailNow()
	}
	if err := otherConfig.Compare(ctx2, baseConfig, true); err == nil {
		t.Error("Config compare is not working correctly")
		t.Log(err)
		t.FailNow()
	}
	if err := otherConfig.Compare(ctx2, baseConfig, false); err == nil {
		t.Error("Config compare is not working correctly")
		t.Log(err)
		t.FailNow()
	}
}
