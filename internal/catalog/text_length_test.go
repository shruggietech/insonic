// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestIdentityUnicodeCharacterLimits(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	for _, alphabet := range []string{"a", "é", "漢", "😀", "e\u0301"} {
		// A combining sequence counts twice; no normalization changes its bytes.
		unit := len([]rune(alphabet))
		name := strings.Repeat(alphabet, 512/unit)
		identity := SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: name}, Aliases: []SpeakerAlias{{ID: contracts.ID(), Text: name}}}
		identity.Aliases[0].SpeakerID = identity.Speaker.ID
		got, err := s.PutSpeaker(ctx, contracts.ID(), 0, identity)
		if err != nil {
			t.Errorf("%q at 512 characters: %v", alphabet, err)
			continue
		}
		if got.Speaker.Name != name || got.Aliases[0].Text != name {
			t.Fatal("text normalized")
		}
		term, err := s.PutTerm(ctx, contracts.ID(), 0, Term{ID: contracts.ID(), Canonical: name, Variants: []byte("[]")})
		if err != nil || term.Canonical != name {
			t.Fatalf("term scalar limit: %v", err)
		}
		if identityText(name+"a", 512) {
			t.Fatal("513 characters accepted")
		}
	}
	if identityText(string([]byte{0xff}), 512) {
		t.Fatal("damaged UTF-8 accepted")
	}
}
