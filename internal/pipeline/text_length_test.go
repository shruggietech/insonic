// SPDX-License-Identifier: Apache-2.0
package pipeline

import (
	"strings"
	"testing"
)

func TestHostedModelUnicodeCharacterLimit(t *testing.T) {
	s := remoteStage("https://example.test/process")
	s.RemoteModel = strings.Repeat("😀", 256)
	if err := ValidateStage(s, "transcription"); err != nil {
		t.Fatal("256-character hosted model", err)
	}
	s.RemoteModel += "a"
	if ValidateStage(s, "transcription") == nil {
		t.Fatal("257-character hosted model accepted")
	}
}
