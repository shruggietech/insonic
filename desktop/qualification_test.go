// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"encoding/json"
	"testing"

	"github.com/shruggietech/insonic/internal/app"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/schemas"
)

func TestNativePreparationInputsUsePublishedRequestContracts(t *testing.T) {
	for _, fixture := range []struct {
		operation string
		item      string
		data      any
	}{
		{"media.import", "", library.ImportRequest{Kind: "import-manifest", Version: contracts.Version, Items: []library.Item{{Source: "/controlled/speech.flac", Subtitle: "/controlled/speech.srt", NewEntry: true}}}},
		{"recordings.assemble", contracts.ID(), app.AssemblyInput{SubtitleFormat: "srt", Turns: []processing.Turn{{Label: "qualification voice", StartUS: 0, EndUS: 1_000_000}}}},
	} {
		raw, err := json.Marshal(fixture.data)
		if err != nil {
			t.Fatal(err)
		}
		request, err := json.Marshal(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: fixture.operation, ItemID: fixture.item, Data: raw})
		if err != nil {
			t.Fatal(err)
		}
		if err := schemas.ValidateRequest(request); err != nil {
			t.Fatalf("%s: %v", fixture.operation, err)
		}
	}
}

func TestQualificationProgressIsExplicitAndBounded(t *testing.T) {
	b := &Bridge{}
	if b.QualificationStep("ready") {
		t.Fatal("ordinary desktop activated qualification diagnostics")
	}
	b.qualification = map[string]any{}
	if b.QualificationStep("arbitrary fixture input") || !b.QualificationStep("ready") {
		t.Fatal("qualification accepted unbounded diagnostics or refused its stage")
	}
}
