// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"testing"
)

func TestRecordingOptionsAndDurablePayload(t *testing.T) {
	good := RecordingOptions{Transcription: "reuse", Diarization: "run", DiarizationModelID: contracts.ID()}
	if validateRecordingOptions(good) != nil {
		t.Fatal("independent rerun rejected")
	}
	for _, bad := range []RecordingOptions{
		{Transcription: "generate", Diarization: "run"},
		{Transcription: "reuse", Diarization: "reuse"},
		{Transcription: "supplied", Diarization: "run", DiarizationModelID: "model-name"},
		{Transcription: "reuse", Diarization: "run", DiarizationModelID: contracts.ID(), Audio: processing.AudioOptions{Channel: ptrInt(-1)}},
	} {
		if validateRecordingOptions(bad) == nil {
			t.Fatal("invalid options", bad)
		}
	}
	assembled := AssemblyInput{Turns: []processing.Turn{{Label: "fixture", StartUS: 0, EndUS: 1000}}}
	raw, _ := json.Marshal(assembled)
	if !containsAssignmentInput(raw) {
		t.Fatal("test fixture not an assignment")
	}
	payload := recordingPayload{MediaID: contracts.ID(), Expected: 0, SourceRevision: 1, Options: good}
	raw, _ = json.Marshal(payload)
	if containsAssignmentInput(raw) {
		t.Fatal("assignment copied into durable payload")
	}
}
func ptrInt(v int) *int { return &v }
func containsAssignmentInput(raw []byte) bool {
	var doc map[string]json.RawMessage
	json.Unmarshal(raw, &doc)
	_, turns := doc["turns"]
	_, document := doc["document"]
	return turns || document
}
