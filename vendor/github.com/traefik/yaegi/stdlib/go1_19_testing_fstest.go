// Code generated by 'yaegi extract testing/fstest'. DO NOT EDIT.

//go:build go1.19 && !go1.20
// +build go1.19,!go1.20

package stdlib

import (
	"reflect"
	"testing/fstest"
)

func init() {
	Symbols["testing/fstest/fstest"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"TestFS": reflect.ValueOf(fstest.TestFS),

		// type definitions
		"MapFS":   reflect.ValueOf((*fstest.MapFS)(nil)),
		"MapFile": reflect.ValueOf((*fstest.MapFile)(nil)),
	}
}
