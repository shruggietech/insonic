// SPDX-License-Identifier: Apache-2.0
package pipeline

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestTrainingOnlyProfileDoesNotElectRecognition(t *testing.T) {
	raw := json.RawMessage(`{"adapter":{"id":"pyannote-profile","contract_version":"1","mode":"local","architecture":"pyannote","output_kinds":["voice-embedding"],"consumers":["voice-matching"]},"base_model_id":"` + contracts.ID() + `","output_kind":"voice-embedding"}`)
	definition := Definition{SpeakerTraining: raw}
	if err := Validate("local", definition); err != nil {
		t.Fatal(err)
	}
	if _, err := Elect("local", definition, Override{}); err == nil {
		t.Fatal("training-only profile elected processing")
	}
	if err := Validate("connected", definition); err == nil {
		t.Fatal("hosted preset accepted local adapter")
	}
}
