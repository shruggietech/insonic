// SPDX-License-Identifier: Apache-2.0
package evidence

import (
	"strings"
	"testing"
)

func TestModelUnicodeCharacterLimit(t *testing.T) {
	c := DefaultConfig()
	c.Adapter = "insonic-http"
	c.Endpoint = "https://example.org/extract"
	c.Model = strings.Repeat("漢", 256)
	if _, err := Normalize(c); err != nil {
		t.Fatal("256-character model", err)
	}
	c.Model += "a"
	if _, err := Normalize(c); err == nil {
		t.Fatal("257-character model accepted")
	}
}
