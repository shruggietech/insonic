// SPDX-License-Identifier: Apache-2.0
package answers

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
)

const SynthesisContract = "answer-synthesis-v1"
const GroundedKind = "grounded-answer"
const MaxSynthesisClaims = 128
const MaxSynthesisReferences = 128

// ProviderEvidence is only the exact cue prefix included in the elected context.
type ProviderEvidence struct {
	ID        string    `json:"id"`
	Reference Reference `json:"reference"`
	Text      string    `json:"text"`
	Family    string    `json:"family"`
}
type SynthesisRequest struct {
	Contract      string             `json:"contract"`
	Version       string             `json:"version"`
	PackageID     string             `json:"package_id"`
	Question      string             `json:"question"`
	Evidence      []ProviderEvidence `json:"evidence"`
	Conflicts     []Conflict         `json:"conflicts"`
	Limitations   []string           `json:"limitations"`
	RepairReasons []string           `json:"repair_reasons,omitempty"`
}
type Attempt struct {
	Number         int    `json:"number"`
	ResponseDigest string `json:"response_digest,omitempty"`
	Outcome        string `json:"outcome"`
	Reason         string `json:"reason,omitempty"`
}
type Grounded struct {
	Kind                string    `json:"kind"`
	Version             string    `json:"schema_version"`
	ID                  string    `json:"id"`
	WorkspaceID         string    `json:"workspace_id"`
	WorkID              string    `json:"work_id"`
	PackageID           string    `json:"package_id"`
	SourceEpoch         string    `json:"source_epoch"`
	Answer              Envelope  `json:"answer"`
	Provider            Provider  `json:"provider"`
	Route               string    `json:"route"`
	IndependentFamilies int       `json:"independent_families"`
	EvidenceCount       int       `json:"evidence_count"`
	Attempts            []Attempt `json:"attempts"`
	Diagnostics         []string  `json:"diagnostics"`
	Validation          string    `json:"validation"`
}

func SynthesisInput(p Package, reasons []string) (SynthesisRequest, error) {
	out := SynthesisRequest{Contract: SynthesisContract, Version: contracts.Version, PackageID: p.ID, Question: p.Request.Question, Evidence: []ProviderEvidence{}, Conflicts: []Conflict{}, Limitations: append([]string{}, p.Diagnostics...), RepairReasons: reasons}
	if ValidateContext(p.Context, p.Items, p.Request.Budget) != nil || len(reasons) > 2 {
		return out, contracts.Fail("invalid_request")
	}
	byID := map[string]Item{}
	for _, item := range p.Items {
		byID[item.ID] = item
	}
	for _, node := range p.Context.Nodes {
		if node.Level != "cue" {
			continue
		}
		item, ok := byID[node.ID]
		if !ok {
			return out, contracts.Fail("invalid_request")
		}
		out.Evidence = append(out.Evidence, ProviderEvidence{ID: item.ID, Reference: item.Reference, Text: node.Text, Family: item.Family})
	}
	// Conflict records are package observations, never additional evidence.
	for _, c := range p.Conflicts {
		out.Conflicts = append(out.Conflicts, c)
	}
	if p.State != "complete" {
		out.Limitations = append(out.Limitations, "evidence_incomplete")
	}
	if p.Context.Omitted > 0 {
		out.Limitations = append(out.Limitations, "context_evidence_omitted")
	}
	for _, reason := range reasons {
		if !contracts.ValidTextLength(reason, 1, 128) {
			return out, contracts.Fail("invalid_request")
		}
	}
	return out, nil
}
func ValidateSynthesis(e Envelope, p Package) error {
	if len(e.Claims) > MaxSynthesisClaims || e.State == "cancelled" {
		return contracts.Fail("invalid_request")
	}
	if err := ValidateEnvelope(e, p); err != nil {
		return err
	}
	visible := map[string]string{}
	for _, node := range p.Context.Nodes {
		if node.Level == "cue" {
			visible[node.ID] = node.Text
		}
	}
	refs := map[string]bool{}
	for _, claim := range e.Claims {
		if claim.Kind != "quote" && claim.Kind != "paraphrase" {
			return contracts.Fail("invalid_request")
		}
		quoted := false
		for _, cite := range claim.Citations {
			text, ok := visible[cite.EvidenceID]
			if !ok || !strings.Contains(text, cite.Quote) {
				return contracts.Condition("conflict", "citation_authority")
			}
			refs[cite.EvidenceID] = true
			quoted = quoted || cite.Quote == claim.Text
		}
		if claim.Kind == "quote" && !quoted {
			return contracts.Condition("conflict", "citation_authority")
		}
	}
	if len(refs) > MaxSynthesisReferences {
		return contracts.Fail("input_limit")
	}
	if e.State == "complete" && (p.Context.Omitted > 0 || len(p.Conflicts) > 0) {
		return contracts.Condition("conflict", "citation_authority")
	}
	return nil
}
func ParseSynthesis(raw []byte, p Package) (Envelope, error) {
	var e Envelope
	if len(raw) > 512<<10 || contracts.DecodeExplore(raw, &e) != nil {
		return e, contracts.Fail("invalid_request")
	}
	return e, ValidateSynthesis(e, p)
}
func Insufficient(p Package, reason string) Envelope {
	return Envelope{Version: contracts.Version, PackageID: p.ID, State: "insufficient", Claims: []Claim{{ID: "insufficient", Kind: "paraphrase", Text: "The available evidence does not establish a verified answer.", Support: "unsupported", Uncertainty: reason}}, Limitations: []string{reason}}
}

// SerializedSize is used before retaining or sending a structured object.
func SerializedSize(v any) int {
	raw, err := json.Marshal(v)
	if err != nil {
		return MaxPackageBytes + 1
	}
	return len(raw)
}
