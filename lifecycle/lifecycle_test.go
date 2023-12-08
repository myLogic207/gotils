package lifecycle

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/myLogic207/gotils/config"
)

var (
	ErrNotInit     = errors.New("system not initialized")
	testInitConfig = map[string]interface{}{
		"TIMEOUT": "1s",
		"PREFIX":  "LIFECYCLE-TEST",
		"LOGGER": map[string]interface{}{
			"WRITERS": map[string]interface{}{
				"STDOUT": true,
				"FILE": map[string]interface{}{
					"ACTIVE": false,
				},
			},
		},
	}
	testSystemConfig = map[string]interface{}{
		"TESTKEY": "TESTVAL",
	}
)

type TestSystem struct {
	SubSystem
	initialized bool
	val         interface{}
}

func (t *TestSystem) Init(ctx context.Context, cfg config.Config) error {
	log.Println("Initializing Test Stub")

	testVal, err := cfg.Get("TESTKEY")
	if err != nil {
		return err
	}
	t.val = testVal
	t.initialized = true
	return nil
}

func (t *TestSystem) Shutdown() error {
	log.Println("Shutting down test stub")
	t.initialized = false
	return nil
}

func (t *TestSystem) TestVal() (interface{}, error) {
	if !t.initialized {
		log.Println("Not initialized")
		return nil, ErrNotInit
	}
	log.Println("Val is: ", t.val)
	return t.val, nil
}

func TestSystemSelfTest(t *testing.T) {
	system := &TestSystem{}
	if err := system.Init(context.Background(), config.NewConfigWithInitialValues(testSystemConfig)); err != nil {
		t.Log(err)
		t.Error("Test system is not initializing correctly")
		t.FailNow()
	}
	if val, err := system.TestVal(); err != nil || val != testSystemConfig["TESTKEY"] {
		t.Log(err)
		t.Error("Test system is not responding correctly")
		t.FailNow()
	}
	if err := system.Shutdown(); err != nil {
		t.Log(err)
		t.Error("Test system is not shutting down correctly")
		t.FailNow()
	}
	if _, err := system.TestVal(); !errors.Is(err, ErrNotInit) {
		t.Log(err)
		t.Error("Test system is initialized after shutdown")
		t.FailNow()
	}
}

func TestSimpleLifecycle(t *testing.T) {
	ctx := context.Background()
	initSystem, err := NewInitializer(ctx, config.NewConfigWithInitialValues(testInitConfig))
	if err != nil {
		t.Log(err)
		t.Error("Initializer is not creating correctly")
		t.FailNow()
	}
	testSystem := &TestSystem{}
	if err := initSystem.AddSystem("TEST", testSystem, config.NewConfigWithInitialValues(testSystemConfig)); err != nil {
		t.Log(err)
		t.Error("Initializer is not adding systems correctly")
		t.FailNow()
	}

	if _, err := testSystem.TestVal(); err == nil || !errors.Is(err, ErrNotInit) {
		t.Log(err)
		t.Error("Test system is initialized before initializer is initialized")
		t.FailNow()
	}
	t.Log("Initializing")
	if err := initSystem.Init(ctx, nil); err != nil {
		t.Log(err)
		t.Error("Initializer is not initializing correctly")
		t.FailNow()
	}

	if val, err := testSystem.TestVal(); err != nil || val != testSystemConfig["TESTKEY"] {
		t.Log(err)
		t.Error("Test system is not responding correctly")
		t.FailNow()
	}

	t.Log("Shutting down")
	if err := initSystem.Shutdown(); err != nil {
		t.Log(err)
		t.Error("Initializer is not shutting down correctly")
		t.FailNow()
	}

	if val, err := testSystem.TestVal(); !errors.Is(err, ErrNotInit) || val == testSystemConfig["TESTKEY"] {
		t.Log(err)
		t.Log(val)
		t.Error("Test system is initialized after initializer is shutdown")
		t.FailNow()
	}
}
