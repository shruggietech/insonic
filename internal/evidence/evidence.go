// SPDX-License-Identifier: Apache-2.0
package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"sort"
	"strings"
	"unicode/utf8"
)

type Cue struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	StartMS int64    `json:"start_milliseconds"`
	EndMS   int64    `json:"end_milliseconds"`
	Timed   bool     `json:"timed"`
	Voices  []string `json:"local_speaker_ids"`
}
type Config struct {
	Adapter          string `json:"adapter"`
	ContractVersion  string `json:"contract_version"`
	Endpoint         string `json:"endpoint,omitempty"`
	CredentialID     string `json:"credential_id,omitempty"`
	Model            string `json:"model,omitempty"`
	WindowCues       int    `json:"window_cues,omitempty"`
	OverlapCues      int    `json:"overlap_cues,omitempty"`
	MaxChunkBytes    int    `json:"max_chunk_bytes,omitempty"`
	TimeoutMS        int    `json:"timeout_ms,omitempty"`
	MaxResponseBytes int    `json:"max_response_bytes,omitempty"`
}
type Chunk struct {
	Index int   `json:"index"`
	Cues  []Cue `json:"cues"`
}
type Assertion struct {
	ID                string   `json:"id,omitempty"`
	Subject           string   `json:"subject"`
	Relation          string   `json:"relation"`
	Object            string   `json:"object"`
	Polarity          string   `json:"polarity"`
	Modality          string   `json:"modality"`
	Conditions        []string `json:"conditions"`
	CueIDs            []string `json:"cue_ids"`
	LocalSpeakerID    string   `json:"local_speaker_id,omitempty"`
	QuotedAttribution string   `json:"quoted_attribution,omitempty"`
}
type Diagnostic struct {
	Chunk int    `json:"chunk"`
	Code  string `json:"code"`
	Count int    `json:"count"`
}
type Result struct {
	Assertions  []Assertion  `json:"assertions"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Chunks      int          `json:"chunks"`
}
type Adapter interface {
	Extract(context.Context, Chunk, Config) ([]Assertion, error)
}

func DefaultConfig() Config {
	return Config{Adapter: "cue-statement", ContractVersion: "1", WindowCues: 16, OverlapCues: 2, MaxChunkBytes: 65536, TimeoutMS: 30000, MaxResponseBytes: 524288}
}
func Normalize(c Config) (Config, error) {
	if c.Adapter == "" {
		c.Adapter = "cue-statement"
	}
	if c.ContractVersion == "" {
		c.ContractVersion = "1"
	}
	if c.WindowCues == 0 {
		c.WindowCues = 16
	}
	if c.MaxChunkBytes == 0 {
		c.MaxChunkBytes = 65536
	}
	if c.TimeoutMS == 0 {
		c.TimeoutMS = 30000
	}
	if c.MaxResponseBytes == 0 {
		c.MaxResponseBytes = 524288
	}
	if c.ContractVersion != "1" || (c.Adapter != "cue-statement" && c.Adapter != "insonic-http") || c.WindowCues < 1 || c.WindowCues > 128 || c.OverlapCues < 0 || c.OverlapCues >= c.WindowCues || c.MaxChunkBytes < 1 || c.MaxChunkBytes > 1<<20 || c.TimeoutMS < 1 || c.TimeoutMS > 600000 || c.MaxResponseBytes < 1 || c.MaxResponseBytes > 1<<20 || len(c.Model) > 256 {
		return c, contracts.Fail("invalid_request")
	}
	if c.Adapter == "cue-statement" && (c.Endpoint != "" || c.CredentialID != "" || c.Model != "") {
		return c, contracts.Fail("invalid_request")
	}
	if c.Adapter == "insonic-http" {
		if e := ValidateHTTP(c); e != nil {
			return c, e
		}
	}
	return c, nil
}
func ParseCues(document json.RawMessage) ([]Cue, error) {
	var doc struct {
		Cues []struct {
			ID      string `json:"id"`
			Payload struct {
				Text string `json:"plain_text"`
			} `json:"payload"`
			Timing *struct {
				Start int64 `json:"start_milliseconds"`
				End   int64 `json:"end_milliseconds"`
			} `json:"timing"`
			Attributions []struct {
				ID string `json:"speaker_id"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
	}
	if json.Unmarshal(document, &doc) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	out := []Cue{}
	seen := map[string]bool{}
	for _, c := range doc.Cues {
		if c.ID == "" || seen[c.ID] || !utf8.ValidString(c.Payload.Text) {
			return nil, contracts.Fail("invalid_request")
		}
		seen[c.ID] = true
		v := Cue{ID: c.ID, Text: c.Payload.Text, Voices: []string{}}
		if c.Timing != nil {
			v.Timed = true
			v.StartMS = c.Timing.Start
			v.EndMS = c.Timing.End
		}
		voices := map[string]bool{}
		for _, a := range c.Attributions {
			if !voices[a.ID] {
				v.Voices = append(v.Voices, a.ID)
				voices[a.ID] = true
			}
		}
		out = append(out, v)
	}
	return out, nil
}
func Chunks(cues []Cue, c Config) ([]Chunk, error) {
	out := []Chunk{}
	for start := 0; start < len(cues); {
		end := start
		size := 0
		for end < len(cues) && end-start < c.WindowCues {
			raw, _ := json.Marshal(cues[end])
			if len(raw) > c.MaxChunkBytes {
				return nil, contracts.Fail("input_limit")
			}
			if size+len(raw) > c.MaxChunkBytes {
				break
			}
			if end > start+c.OverlapCues && strings.Join(cues[end].Voices, ",") != strings.Join(cues[end-1].Voices, ",") {
				break
			}
			size += len(raw)
			end++
		}
		if end == start {
			return nil, contracts.Fail("input_limit")
		}
		out = append(out, Chunk{Index: len(out), Cues: append([]Cue{}, cues[start:end]...)})
		if end == len(cues) {
			break
		}
		start = max(start+1, end-c.OverlapCues)
	}
	return out, nil
}
func boundedText(s string) bool {
	return strings.TrimSpace(s) != "" && len(s) <= 4096 && utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}
func ValidateAssertion(a Assertion, cues []Cue) error {
	if !boundedText(a.Subject) || !boundedText(a.Relation) || !boundedText(a.Object) || len(a.CueIDs) < 1 || len(a.CueIDs) > 128 || len(a.Conditions) > 64 || len(a.QuotedAttribution) > 4096 || !utf8.ValidString(a.QuotedAttribution) || strings.ContainsRune(a.QuotedAttribution, 0) {
		return contracts.Fail("invalid_request")
	}
	if !strings.Contains("|positive|negative|unspecified|", "|"+a.Polarity+"|") || !strings.Contains("|asserted|possible|conditional|question|verbatim|", "|"+a.Modality+"|") {
		return contracts.Fail("invalid_request")
	}
	for _, c := range a.Conditions {
		if !boundedText(c) {
			return contracts.Fail("invalid_request")
		}
	}
	byID := map[string]Cue{}
	for _, c := range cues {
		byID[c.ID] = c
	}
	seen := map[string]bool{}
	voice := a.LocalSpeakerID == ""
	for _, id := range a.CueIDs {
		c, ok := byID[id]
		if !ok || seen[id] {
			return contracts.Fail("invalid_request")
		}
		seen[id] = true
		for _, v := range c.Voices {
			if v == a.LocalSpeakerID {
				voice = true
			}
		}
	}
	if !voice {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func Deduplicate(assertions []Assertion) []Assertion {
	out := []Assertion{}
	seen := map[string]bool{}
	for _, a := range assertions {
		a.ID = ""
		a.CueIDs = append([]string{}, a.CueIDs...)
		sort.Strings(a.CueIDs)
		if a.Conditions == nil {
			a.Conditions = []string{}
		}
		raw, _ := json.Marshal(a)
		h := sha256.Sum256(raw)
		key := hex.EncodeToString(h[:])
		if !seen[key] {
			a.ID = key
			out = append(out, a)
			seen[key] = true
		}
	}
	return out
}

type literalAdapter struct{}

func (literalAdapter) Extract(_ context.Context, ch Chunk, _ Config) ([]Assertion, error) {
	out := []Assertion{}
	for _, cue := range ch.Cues {
		if strings.TrimSpace(cue.Text) == "" {
			continue
		}
		voices := cue.Voices
		if len(voices) == 0 {
			voices = []string{""}
		}
		for _, voice := range voices {
			subject := "unattributed"
			if voice != "" {
				subject = "local-voice:" + voice
			}
			out = append(out, Assertion{Subject: subject, Relation: "stated", Object: "cue:" + cue.ID, Polarity: "unspecified", Modality: "verbatim", Conditions: []string{}, CueIDs: []string{cue.ID}, LocalSpeakerID: voice})
		}
	}
	return out, nil
}
func Extract(ctx context.Context, cues []Cue, c Config, adapter Adapter) (Result, error) {
	c, e := Normalize(c)
	out := Result{Assertions: []Assertion{}, Diagnostics: []Diagnostic{}}
	if e != nil {
		return out, e
	}
	chunks, e := Chunks(cues, c)
	if e != nil {
		return out, e
	}
	out.Chunks = len(chunks)
	if adapter == nil {
		if c.Adapter != "cue-statement" {
			return out, contracts.Fail("unavailable")
		}
		adapter = literalAdapter{}
	}
	for _, chunk := range chunks {
		if ctx.Err() != nil {
			return out, contracts.Fail("cancelled")
		}
		assertions, e := adapter.Extract(ctx, chunk, c)
		if e != nil {
			out.Diagnostics = append(out.Diagnostics, Diagnostic{chunk.Index, "extraction_failed", 1})
			continue
		}
		if len(assertions) > 1000 {
			out.Diagnostics = append(out.Diagnostics, Diagnostic{chunk.Index, "invalid_assertions", len(assertions)})
			continue
		}
		rejected := 0
		for _, a := range assertions {
			if ValidateAssertion(a, chunk.Cues) != nil {
				rejected++
			} else {
				out.Assertions = append(out.Assertions, a)
			}
		}
		if rejected > 0 {
			out.Diagnostics = append(out.Diagnostics, Diagnostic{chunk.Index, "invalid_assertions", rejected})
		}
		out.Assertions = Deduplicate(out.Assertions)
		raw, _ := json.Marshal(out)
		if len(raw) > 512<<10 {
			return out, contracts.Fail("output_limit")
		}
	}
	return out, nil
}
