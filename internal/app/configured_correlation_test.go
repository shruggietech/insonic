// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

type correlationFixtureCatalog struct {
	catalog.Catalog
	Evidence catalog.ResolvedEvidence
	Failure  error
}

func (c *correlationFixtureCatalog) CurrentSpeakerReferences(context.Context, catalog.SpeakerSelection) (catalog.SpeakerSelectionPage, error) {
	return catalog.SpeakerSelectionPage{References: []catalog.CurrentReference{c.Evidence.Reference}, Revision: 1}, nil
}
func (c *correlationFixtureCatalog) ResolveEvidence(context.Context, catalog.CurrentReference) (catalog.ResolvedEvidence, error) {
	return c.Evidence, c.Failure
}
func (c *correlationFixtureCatalog) ComparePriorSpeakerEvidence(context.Context, catalog.SpeakerSelection, catalog.CurrentReference, string) ([]catalog.PriorEvidenceComparison, error) {
	return []catalog.PriorEvidenceComparison{{AssignmentOrdinal: 0, LocalVoices: 1, FirstReference: true}, {AssignmentOrdinal: 1, LocalVoices: 1, FirstReference: true}}, c.Failure
}

func TestCorrelationQualityDefaultDisabledAndMandatoryIntegrity(t *testing.T) {
	a := &App{ctx: context.Background()}
	speakerID, localID := contracts.ID(), contracts.ID()
	fixture := &correlationFixtureCatalog{Catalog: a.Catalog, Evidence: catalog.ResolvedEvidence{Reference: catalog.CurrentReference{RecordingID: contracts.ID(), CueID: "cue", SpeakerID: speakerID, LocalSpeakerID: localID}, Mapping: catalog.SpeakerMapping{SpeakerID: speakerID}, SourceMap: json.RawMessage(`{"start_numerator":"0","start_denominator":"1","stream_index":0,"channel":null}`), Cue: json.RawMessage(`{"id":"cue","speaker_attributions":[{"speaker_id":"` + localID + `","start_milliseconds":100,"end_milliseconds":500},{"speaker_id":"` + localID + `"}]}`)}}
	a.Catalog = fixture
	call := func(operation string, data any) contracts.Response {
		t.Helper()
		var raw json.RawMessage
		if data != nil {
			raw, _ = json.Marshal(data)
		}
		result, err := a.configuredDispatch(contracts.Request{Operation: operation, ItemID: speakerID, Data: raw})
		response := contracts.Response{Result: result}
		if err != nil {
			var ok bool
			response.Error, ok = err.(*contracts.Error)
			if !ok {
				t.Fatal(err)
			}
		}
		return response
	}
	for _, operation := range []string{"speakers.select", "speakers.diagnostics"} {
		enabled := configuredResult(t, call(operation, nil))
		quality, ok := enabled["quality"].(map[string]any)
		if !ok || quality["enabled"] != true {
			t.Fatal("default effective quality missing", enabled)
		}
		selection := enabled["selection"].(map[string]any)
		if len(selection["diagnostics"].([]any)) == 0 {
			t.Fatal("default correlation diagnostics missing")
		}
		disabled := configuredResult(t, call(operation, map[string]any{"quality": map[string]any{"enabled": false}}))
		if disabled["quality"].(map[string]any)["enabled"] != false {
			t.Fatal("effective disabled setting missing")
		}
		selection = disabled["selection"].(map[string]any)
		if len(selection["diagnostics"].([]any)) != 0 || len(selection["items"].([]any)) != 1 || selection["items"].([]any)[0].(map[string]any)["untimed"] != true {
			t.Fatal("disabled diagnostics changed evidence selection", disabled)
		}
		unknown := call(operation, map[string]any{"quality": map[string]any{"threshold": 1}})
		if unknown.Error == nil || unknown.Error.Code != "invalid_request" {
			t.Fatal("unknown correlation quality admitted")
		}
		fixture.Failure = contracts.Fail("conflict")
		invalid := call(operation, map[string]any{"quality": map[string]any{"enabled": false}})
		if invalid.Error == nil || invalid.Error.Code != "conflict" {
			t.Fatal("disabled quality bypassed current-reference resolution")
		}
		fixture.Failure = nil
	}
}
