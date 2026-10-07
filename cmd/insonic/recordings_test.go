// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordingCLIInput(t *testing.T) {
	id := contracts.ID()
	path := filepath.Join(t.TempDir(), "request.json")
	if e := os.WriteFile(path, []byte(`{"transcription":"reuse","diarization":"run","diarization_model_id":"10000000-0000-4000-8000-000000000001"}`), 0600); e != nil {
		t.Fatal(e)
	}
	op, item, raw, e := parseRecording([]string{"recordings", "process", id, "--input", path})
	if e != nil || op != "recordings.process" || item != id || len(raw) == 0 {
		t.Fatal(op, item, e)
	}
	if _, _, _, e = parseRecording([]string{"recordings", "process", id}); e == nil {
		t.Fatal("missing input admitted")
	}
	if _, _, _, e = parseRecording([]string{"recordings", "unknown", id, "--input", path}); e == nil {
		t.Fatal("unknown operation admitted")
	}
	if _, _, _, e = parseRecording([]string{"recordings", "show", id}); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, []byte(`{"transcription":"reuse","transcription":"generate"}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, _, _, e = parseRecording([]string{"recordings", "process", id, "--input", path}); e == nil {
		t.Fatal("duplicate payload keys admitted")
	}
}
