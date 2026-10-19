// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"strings"
	"testing"
)

func TestModelUnicodeCharacterLimit(t *testing.T) {
	c := DefaultConfig()
	c.Enabled = true
	c.Endpoint = "http://localhost:1234/query"
	c.Model = strings.Repeat("😀", 512)
	if _, err := Normalize(c); err != nil {
		t.Fatal("512-character model", err)
	}
	c.Model += "a"
	if _, err := Normalize(c); err == nil {
		t.Fatal("513-character model accepted")
	}
}
