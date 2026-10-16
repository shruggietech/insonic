// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
)

func TestTranscriptManifestRejectsMediaIntentBeforeDispatch(t *testing.T) {
	yes, no := true, false
	for _, test := range []struct {
		name      string
		item      library.Item
		defaults  library.Options
		overrides library.Options
	}{
		{name: "media replacement", item: library.Item{Kind: "media", Source: "audio.wav", Options: library.Options{ReplaceAudio: &yes}}},
		{name: "implicit audio replacement", item: library.Item{Source: "audio.wav", Options: library.Options{ReplaceAudio: &yes}}},
		{name: "media kind with transcript", item: library.Item{Kind: "media", Transcript: "caption.srt"}},
		{name: "new entry", item: library.Item{Transcript: "caption.srt", NewEntry: true}},
		{name: "disabled audio option", item: library.Item{Transcript: "caption.srt", Options: library.Options{ReplaceAudio: &no}}},
		{name: "default audio replacement", item: library.Item{Source: "audio.wav"}, defaults: library.Options{ReplaceAudio: &yes}},
		{name: "override audio replacement", item: library.Item{Source: "audio.wav"}, overrides: library.Options{ReplaceAudio: &yes}},
		{name: "preset with audio policy", item: library.Item{Transcript: "caption.srt", Options: library.Options{Preset: "saved", ExistingRoster: "clear"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.item.Record = contracts.ID()
			request := library.ImportRequest{Kind: "import-manifest", Version: contracts.Version, Defaults: test.defaults, Overrides: test.overrides,
				Items: []library.Item{{Record: contracts.ID(), Transcript: "valid.srt"}, test.item}}
			file := filepath.Join(t.TempDir(), "transcript batch.json")
			raw, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(file, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if op, _, _, err := parseDomain([]string{"transcript", "import", "--manifest", file}); err == nil {
				t.Fatal("transcript manifest admitted media intent", op)
			}
		})
	}
}

func TestTranscriptManifestPreservesTranscriptSourceAliases(t *testing.T) {
	for _, preset := range []string{"", "saved"} {
		file := filepath.Join(t.TempDir(), "transcripts.json")
		request := library.ImportRequest{Kind: "import-manifest", Version: contracts.Version,
			Items: []library.Item{{Kind: "transcript", Record: contracts.ID(), Source: "caption.srt", Options: library.Options{Preset: preset}}}}
		raw, _ := json.Marshal(request)
		if err := os.WriteFile(file, raw, 0600); err != nil {
			t.Fatal(err)
		}
		op, _, raw, err := parseDomain([]string{"transcript", "import", "--manifest", file})
		if err != nil || op != "media.import" {
			t.Fatal(op, err)
		}
		var parsed library.ImportRequest
		if json.Unmarshal(raw, &parsed) != nil || parsed.Items[0].Source != "" || parsed.Items[0].Transcript != filepath.Join(filepath.Dir(file), "caption.srt") || parsed.Items[0].Kind != "transcript" {
			t.Fatal("transcript source alias or intent lost", string(raw))
		}
	}
}
