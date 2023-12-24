package config

import (
	"errors"
	"fmt"
)

var (
	ErrDumpFailed     = errors.New("dump failed")
	ErrMergeFailed    = errors.New("merge failed")
	ErrConfigKey      = errors.New("config key error")
	ErrTypeMismatch   = errors.New("type mismatch")
	ErrCopyConfig     = errors.New("copy config error")
	ErrNoConfigSource = errors.New("no config source provided")
	ErrLoadingConfig  = errors.New("loading config failed")
	ErrValueInvalid   = errors.New("value invalid")
)

type ErrKeyValueInvalid struct {
	key    string
	value  interface{}
	nested error
}

func (v *ErrKeyValueInvalid) Error() string {
	return fmt.Sprintf("invalid value: '%v' for key %s", v.value, v.key)
}

func (v *ErrKeyValueInvalid) Wrap(err error) {
	v.nested = err
}

func (v *ErrKeyValueInvalid) Unwrap() error {
	if v.nested != nil {
		return v.nested
	}
	return ErrValueInvalid
}

type ErrKeyInStore struct {
	key string
}

func (k *ErrKeyInStore) Error() string {
	return "key already in store: " + k.key
}

func (k *ErrKeyInStore) Unwrap() error {
	return ErrConfigKey
}

type ErrKeyNotFound struct {
	key string
}

func (k *ErrKeyNotFound) Error() string {
	return "key not in store: " + k.key
}

func (k *ErrKeyNotFound) Unwrap() error {
	return ErrConfigKey
}

type KeyCharInvalid struct {
	key  string
	char rune
}

func (k *KeyCharInvalid) Error() string {
	return fmt.Sprintf("invalid character in key: %s (%s)", k.key, string(k.char))
}

func (k *KeyCharInvalid) Unwrap() error {
	return ErrValueInvalid
}

type ErrValueNotFound struct {
	key string
}

func (v *ErrValueNotFound) Error() string {
	return "no value for key: " + v.key
}

func (v *ErrValueNotFound) Unwrap() error {
	return ErrConfigKey
}

type ErrValueMismatch struct {
	key      string
	expected interface{}
	actual   interface{}
}

func (v *ErrValueMismatch) Error() string {
	return fmt.Sprintf("value mismatch: %s (%v != %v)", v.key, v.expected, v.actual)
}

func (v *ErrValueMismatch) Unwrap() error {
	return ErrValueInvalid
}

type ErrDumpToFile struct {
	file   string
	reason error
}

func (c *ErrDumpToFile) Error() string {
	return "writing config to file " + c.file + " failed: " + c.reason.Error()
}

func (c *ErrDumpToFile) Unwrap() error {
	return ErrDumpFailed
}

type ErrFieldNotConfig struct {
	key string
}

func (f *ErrFieldNotConfig) Error() string {
	return "field is not a config: " + f.key
}

func (f *ErrFieldNotConfig) Unwrap() error {
	return ErrTypeMismatch
}

type ErrFieldNotString struct {
	key string
}

func (f *ErrFieldNotString) Error() string {
	return "field is not a string: " + f.key
}

func (f *ErrFieldNotString) Unwrap() error {
	return ErrTypeMismatch
}

type ErrFieldNotInt struct {
	key string
}

func (f *ErrFieldNotInt) Error() string {
	return "field is not an int: " + f.key
}

func (f *ErrFieldNotInt) Unwrap() error {
	return ErrTypeMismatch
}

type ErrFieldNotFloat struct {
	key string
}

func (f *ErrFieldNotFloat) Error() string {
	return "field is not a float: " + f.key
}

func (f *ErrFieldNotFloat) Unwrap() error {
	return ErrTypeMismatch
}

type ErrFieldNotBool struct {
	key string
}

func (f *ErrFieldNotBool) Error() string {
	return "field is not a bool: " + f.key
}

func (f *ErrFieldNotBool) Unwrap() error {
	return ErrTypeMismatch
}

type ErrFieldNotDuration struct {
	key string
}

func (f *ErrFieldNotDuration) Error() string {
	return "field is not a duration: " + f.key
}

func (f *ErrFieldNotDuration) Unwrap() error {
	return ErrTypeMismatch
}

type ErrCopyConfigReason struct {
	err error
}

func (e ErrCopyConfigReason) Error() string {
	return "error copying config: " + e.err.Error()
}

func (e ErrCopyConfigReason) Unwrap() error {
	return ErrCopyConfig
}

type ErrMergeConfigReason struct {
	err error
}

func (e ErrMergeConfigReason) Error() string {
	return "error merging config: " + e.err.Error()
}

func (e ErrMergeConfigReason) Unwrap() error {
	return ErrMergeFailed
}

type ErrKeyAmbiguous struct {
	key string
}

func (e *ErrKeyAmbiguous) Error() string {
	return "key is ambiguous: " + e.key
}

func (e *ErrKeyAmbiguous) Unwrap() error {
	return ErrConfigKey
}

type ErrParsingEnvVar struct {
	nested error
}

func (e *ErrParsingEnvVar) Error() string {
	return "error parsing env var: " + e.nested.Error()
}

func (e *ErrParsingEnvVar) Unwrap() error {
	return ErrLoadingConfig
}
