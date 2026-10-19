// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"strings"
	"testing"
)

func TestNativeUnicodeCharacterLimit(t *testing.T) {
	const prefix = "RETURN '"
	const suffix = "' AS value"
	text := prefix + strings.Repeat("漢", 65536-len(prefix)-len(suffix)) + suffix
	if err := ValidateNative("ladybug-cypher", text, nil); err != nil {
		t.Fatal("65536-character read query", err)
	}
	if err := ValidateNative("ladybug-cypher", text+" ", nil); err == nil {
		t.Fatal("65537-character read query accepted")
	}
	if err := ValidateNative("ladybug-cypher", "RETURN '"+string([]byte{0xff})+"'", nil); err == nil {
		t.Fatal("damaged UTF-8 read accepted")
	}
}
