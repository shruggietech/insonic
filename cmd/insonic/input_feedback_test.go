// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"testing"
)

func TestMissingAndMalformedInputHaveSupportedCorrections(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.json")
	_, err := modelInput(missing)
	if typed, ok := err.(*contracts.Error); !ok || typed.Code != "unavailable" || typed.Reason != "input_missing" {
		t.Fatal(err)
	}
	malformed := filepath.Join(t.TempDir(), "bad.json")
	if err = os.WriteFile(malformed, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = modelInput(malformed)
	if typed, ok := err.(*contracts.Error); !ok || typed.Code != "invalid_request" || typed.Reason != "input_invalid" {
		t.Fatal(err)
	}
}
