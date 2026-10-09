// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"testing"
)

func TestRosterAndReplacementGrammar(t *testing.T) {
	id := contracts.ID()
	op, target, raw, e := parseRecording([]string{"recordings", "roster", "replace", id, "--speaker", "one", "--speaker", "two", "--expected-revision", "0"})
	if e != nil || op != "recordings.roster.replace" || target != id {
		t.Fatalf("%s %s %v", op, target, e)
	}
	var input struct {
		Speakers []string `json:"speakers"`
	}
	json.Unmarshal(raw, &input)
	if len(input.Speakers) != 2 {
		t.Fatal("repeat speakers")
	}
	op, _, raw, e = parseDomain([]string{"media", "import", "source.wav", "--record", id, "--replace-audio", "--existing-transcript", "clear", "--existing-roster", "retain"})
	if e != nil || op != "media.import" {
		t.Fatalf("media %s %v", op, e)
	}
	var request library.ImportRequest
	if json.Unmarshal(raw, &request) != nil || request.Items[0].Kind != "media" || request.Items[0].Record != id || request.Defaults.ReplaceAudio == nil {
		t.Fatal("replacement contract")
	}
	_, _, raw, e = parseDomain([]string{"media", "import", "source.wav", "--known-speaker", "one", "--known-speaker", "two"})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &request)
	if len(request.Defaults.KnownSpeakers) != 2 {
		t.Fatal("repeat known speakers")
	}
}
