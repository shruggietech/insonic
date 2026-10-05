//go:build cgo

package main

import (
	"path/filepath"
	"testing"
)

func TestMissingLookupDistinguishesLockedKeychain(t *testing.T) {
	if !lockedFixture(filepath.Join(t.TempDir(), "qualification.keychain")) {
		t.Fatal("native noninteractive missing/locked keychain qualification failed")
	}
}
