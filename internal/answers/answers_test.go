// SPDX-License-Identifier: Apache-2.0
package answers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func testItem() Item {
	return Item{ID: "cue:a:document:large-9007199254740993", Reference: Reference{
		WorkspaceID: contracts.ID(), RecordingID: contracts.ID(), RecordingRevision: "9007199254740993",
		DocumentDigest: strings.Repeat("a", 64), SourceDigest: strings.Repeat("b", 64), SourceMapDigest: strings.Repeat("c", 64),
		IdentityDigest: strings.Repeat("d", 64), CueID: "large-9007199254740993", Ordinal: "0",
	}, Text: "A source statement", Family: "source:" + strings.Repeat("b", 64), Lineage: "source-digest"}
}

func TestRequestAndExactCitationAuthority(t *testing.T) {
	r, err := Normalize(Request{Version: contracts.Version, Question: "What changed?"})
	if err != nil || r.Budget.Rows != 4096 || r.Provider.SummaryContract != SummaryContract {
		t.Fatal(r, err)
	}
	item := testItem()
	p := Package{Version: contracts.Version, WorkspaceID: item.Reference.WorkspaceID, State: "complete", Items: []Item{item}}
	e := Envelope{Version: contracts.Version, State: "complete", Claims: []Claim{{ID: "c1", Text: "A source statement", Support: "supported", Citations: []Citation{{EvidenceID: item.ID, Reference: item.Reference, Quote: item.Text}}}}}
	raw, _ := json.Marshal(e)
	var decoded Envelope
	if err = contracts.DecodeExplore(raw, &decoded); err != nil || ValidateEnvelope(decoded, p) != nil {
		t.Fatal(string(raw), err)
	}
	if !strings.Contains(string(raw), `"9007199254740993"`) || strings.Contains(string(raw), "start_us") {
		t.Fatal("lost exact identity or invented timing", string(raw))
	}
	for _, mutate := range []func(*Citation){
		func(c *Citation) { c.Reference.DocumentDigest = strings.Repeat("e", 64) },
		func(c *Citation) { c.Reference.RecordingRevision = "9007199254740992" },
		func(c *Citation) { c.Reference.Source = &Interval{StartUS: "0", EndUS: "1"} },
		func(c *Citation) { c.Quote = "The source says something else" },
		func(c *Citation) { c.EvidenceID = "missing" },
	} {
		copy := decoded
		copy.Claims = append([]Claim{}, decoded.Claims...)
		copy.Claims[0].Citations = append([]Citation{}, decoded.Claims[0].Citations...)
		mutate(&copy.Claims[0].Citations[0])
		if ValidateEnvelope(copy, p) == nil {
			t.Fatal("forged citation accepted", copy)
		}
	}
	for _, state := range []string{"partial", "insufficient", "cancelled"} {
		e.State = state
		e.Claims = []Claim{{ID: "c1", Text: "Unknown", Support: "unsupported", Uncertainty: "No current source"}}
		if ValidateEnvelope(e, p) != nil {
			t.Fatal(state)
		}
	}
	e.State = "complete"
	if ValidateEnvelope(e, p) == nil {
		t.Fatal("unsupported complete answer accepted")
	}
}

func TestRequestLimitsAndStrictDecoding(t *testing.T) {
	for _, r := range []Request{
		{Version: "0.0.0", Question: "q"}, {Version: contracts.Version, Question: " "},
		{Version: contracts.Version, Question: "q", Budget: Budget{Rows: -1}},
		{Version: contracts.Version, Question: "q", Budget: Budget{Rows: 100001}},
		{Version: contracts.Version, Question: "q", Filters: Filters{SourceInterval: &Interval{StartUS: "9007199254740993", EndUS: "9007199254740992"}}},
		{Version: contracts.Version, Question: "q", Provider: Provider{SummaryContract: "invented"}},
	} {
		if _, err := Normalize(r); err == nil {
			t.Fatal("accepted", r)
		}
	}
	var r Request
	if contracts.DecodeExplore([]byte(`{"version":"1.3.0","question":"q","credential":"secret"}`), &r) == nil {
		t.Fatal("unknown credential field")
	}
}
