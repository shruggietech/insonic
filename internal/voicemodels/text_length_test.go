// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"strings"
	"testing"
)

func TestAdapterUnicodeCharacterLimits(t *testing.T) {
	a := Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "hosted", Architecture: strings.Repeat("漢", 128), OutputKinds: []string{"weights"}, Consumers: []string{"custom"}, Endpoint: "https://example.com/train", RemoteModel: strings.Repeat("😀", 256), UpstreamRevision: strings.Repeat("é", 512), License: strings.Repeat("漢", 1024)}
	if err := ValidateAdapter(a); err != nil {
		t.Fatal("Unicode adapter at public limits", err)
	}
	a.RemoteModel += "a"
	if ValidateAdapter(a) == nil {
		t.Fatal("257-character hosted identity accepted")
	}
	if text(string([]byte{0xff}), 128) {
		t.Fatal("damaged UTF-8 accepted")
	}
}
