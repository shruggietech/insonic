// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"strings"
	"testing"
)

func TestTurnUnicodeCharacterLimit(t *testing.T) {
	turn := Turn{Label: strings.Repeat("漢", 256), StartUS: 0, EndUS: 1000}
	if !validTurn(turn, 1000) {
		t.Fatal("256-character turn label")
	}
	turn.Label += "a"
	if validTurn(turn, 1000) {
		t.Fatal("257-character turn label accepted")
	}
}
