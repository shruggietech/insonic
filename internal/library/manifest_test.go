// SPDX-License-Identifier: Apache-2.0
package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestsResolveRelativeAndOverrideDateKind(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "batch.json")
	data := `{"kind":"import-manifest","schema_version":"0.0.0","defaults":{"originated_at":"2026-10-07T12:00:00","timezone":"UTC"},"items":[{"source":"a.wav","originated_on":"2026-10-06","subtitle":"a.srt"}]}`
	os.WriteFile(file, []byte(data), 0600)
	r, e := ReadManifest(file)
	if e != nil {
		t.Fatal(e)
	}
	item := r.Items[0]
	if item.Source != filepath.Join(dir, "a.wav") || item.Transcript != filepath.Join(dir, "a.srt") {
		t.Fatalf("paths %+v", item)
	}
	options := merged(r.Defaults, item.Options)
	if options.OriginatedAt != "" || options.OriginatedOn != "2026-10-06" {
		t.Fatalf("override %+v", options)
	}
	csv := filepath.Join(dir, "batch.csv")
	os.WriteFile(csv, []byte("source,copy,originated_on,timezone\na.wav,false,2026-10-07,UTC\n"), 0600)
	r, e = ReadManifest(csv)
	if e != nil || r.Items[0].Copy == nil || *r.Items[0].Copy {
		t.Fatalf("CSV %+v %v", r, e)
	}
	os.WriteFile(file, []byte(`{"items":[{"source":"a.wav","secret":"bad"}]}`), 0600)
	if _, e = ReadManifest(file); e == nil {
		t.Fatal("credential accepted")
	}
	os.WriteFile(csv, []byte("source,source\na.wav,b.wav\n"), 0600)
	if _, e = ReadManifest(csv); e == nil {
		t.Fatal("ambiguous duplicate CSV accepted")
	}
	os.WriteFile(file, []byte(`{"kind":"import-manifest","schema_version":"0.0.0","items":[{"source":"https://example.test/%zz?token=private"}]}`), 0600)
	if _, e = ReadManifest(file); e == nil {
		t.Fatal("manifest path resolution concealed malformed credential URL")
	}
}

func TestRosterManifestElectionPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rosters.json")
	os.WriteFile(path, []byte(`{"kind":"import-manifest","schema_version":"0.0.0","defaults":{"known_speakers":["Alice"],"copy":true},"items":[{"source":"a.wav","known_speakers":[],"replace_audio":false},{"source":"b.wav"}]}`), 0600)
	r, e := ReadManifest(path)
	if e != nil {
		t.Fatal(e)
	}
	first := merged(r.Defaults, r.Items[0].Options)
	if first.KnownSpeakers == nil || len(first.KnownSpeakers) != 0 || first.ReplaceAudio == nil || *first.ReplaceAudio {
		t.Fatal("explicit empty/false did not override defaults")
	}
	raw := marshal(r.Items[0])
	var item Item
	if json.Unmarshal(raw, &item) != nil || item.KnownSpeakers == nil {
		t.Fatal("empty roster election lost on durable serialization")
	}
	path = filepath.Join(dir, "rosters.csv")
	os.WriteFile(path, []byte("kind,source,record,replace_audio,existing_transcript,transcript_applies,existing_roster,known_speakers\nmedia,a.wav,Existing,true,keep,true,retain,\nmedia,b.wav,,,,,,[]\n"), 0600)
	r, e = ReadManifest(path)
	if e != nil || r.Items[0].Kind != "media" || r.Items[0].ExistingTranscript != "keep" || r.Items[0].TranscriptApplies == nil || !*r.Items[0].TranscriptApplies || r.Items[1].KnownSpeakers == nil {
		t.Fatalf("CSV elections %+v %v", r, e)
	}
}

func TestManifestModelReferencesShareJSONAndCSVGrammar(t *testing.T) {
	for _, reference := range []string{"voices", "base:11111111-1111-4111-8111-111111111111", "source:examples/voices-v1"} {
		dir := t.TempDir()
		jsonPath := filepath.Join(dir, "models.json")
		document := map[string]any{"kind": "import-manifest", "schema_version": "0.0.0", "defaults": map[string]string{"attribution": "diarize", "diarization_model_id": reference}, "items": []any{map[string]string{"source": "audio.wav"}}}
		data, _ := json.Marshal(document)
		if e := os.WriteFile(jsonPath, data, 0600); e != nil {
			t.Fatal(e)
		}
		parsed, e := ReadManifest(jsonPath)
		if e != nil || parsed.Defaults.DiarizationModelID != reference {
			t.Fatal("JSON reference grammar", reference, e)
		}
		csvPath := filepath.Join(dir, "models.csv")
		if e = os.WriteFile(csvPath, []byte("source,attribution,diarization_model_id\naudio.wav,diarize,"+reference+"\n"), 0600); e != nil {
			t.Fatal(e)
		}
		parsed, e = ReadManifest(csvPath)
		if e != nil || parsed.Items[0].DiarizationModelID != reference {
			t.Fatal("CSV reference grammar", reference, e)
		}
	}
}
