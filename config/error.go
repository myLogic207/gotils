package config

import (
	"errors"
	"fmt"
)

var ErrConfigOp = errors.New("configuration error")

var (
	ErrDumpFailed  = errors.New("dump failed")
	ErrMergeFailed = errors.New("merge failed")
)

type ErrValueInvalid struct {
	key    string
	value  interface{}
	nested error
}

func (v *ErrValueInvalid) Error() string {
	return fmt.Sprintf("invalid value: '%v' for key %s", v.value, v.key)
}

func (v *ErrValueInvalid) Wrap(err error) {
	v.nested = err
}

func (v *ErrValueInvalid) Unwrap() error {
	if v.nested != nil {
		return v.nested
	}
	return ErrConfigOp
}

type ErrKeyInvalid struct {
	key    string
	nested error
}

func (k *ErrKeyInvalid) Error() string {
	return "invalid key: " + k.key
}

func (k *ErrKeyInvalid) Wrap(err error) {
	k.nested = err
}

func (k *ErrKeyInvalid) Unwrap() error {
	if k.nested != nil {
		return k.nested
	}
	return ErrConfigOp
}

type ErrKeyInStore struct {
	key string
}

func (k *ErrKeyInStore) Error() string {
	return "key already in store: " + k.key
}

type ErrKeyNotFound struct {
	key string
}

func (k *ErrKeyNotFound) Error() string {
	return "key not found: " + k.key
}

func (k *ErrKeyNotFound) Unwrap() error {
	return ErrConfigOp
}

type KeyCharInvalid struct {
	key  string
	char rune
}

func (k *KeyCharInvalid) Error() string {
	return fmt.Sprintf("invalid character in key: %s (%s)", k.key, string(k.char))
}

func (k *KeyCharInvalid) Unwrap() error {
	return ErrConfigOp
}

type ErrValueNotFound struct {
	key string
}

func (v *ErrValueNotFound) Error() string {
	return "no value for key: " + v.key
}

func (v *ErrValueNotFound) Unwrap() error {
	return &ErrKeyInvalid{
		key: v.key,
	}
}

type ErrValueMismatch struct {
	key      string
	value    interface{}
	mismatch interface{}
}

func (v *ErrValueMismatch) Error() string {
	return fmt.Sprintf("value mismatch: %s (%v != %v)", v.key, v.value, v.mismatch)
}

func (v *ErrValueMismatch) Unwrap() error {
	return ErrConfigOp
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
