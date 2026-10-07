//go:build !darwin || !cgo

// SPDX-License-Identifier: Apache-2.0
package main

func nativeFixture(string) (func() error, error) { return func() error { return nil }, nil }
