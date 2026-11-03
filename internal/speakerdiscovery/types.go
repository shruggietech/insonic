// SPDX-License-Identifier: Apache-2.0
// Package speakerdiscovery defines current, non-authoritative speaker hypotheses.
package speakerdiscovery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

const Version = 1
const MaxBytes = 1 << 20
const MaxObservations = 512
const MaxReferences = 16

type Limits struct {
	MaxCues         int `json:"max_cues,omitempty"`
	MaxCalls        int `json:"max_calls,omitempty"`
	MaxObservations int `json:"max_observations,omitempty"`
}

func NormalizeLimits(v Limits) (Limits, error) {
	if v.MaxCues == 0 {
		v.MaxCues = 100
	}
	if v.MaxCalls == 0 {
		v.MaxCalls = 4
	}
	if v.MaxObservations == 0 {
		v.MaxObservations = 128
	}
	if v.MaxCues < 1 || v.MaxCues > 500 || v.MaxCalls < 1 || v.MaxCalls > 20 || v.MaxObservations < 1 || v.MaxObservations > MaxObservations {
		return v, contracts.Fail("invalid_request")
	}
	return v, nil
}

type Alias struct {
	Text     string `json:"text"`
	Language string `json:"language"`
	Scope    string `json:"scope"`
}
type Identity struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Aliases []Alias `json:"aliases"`
	Roster  bool    `json:"roster"`
}
type Cue struct {
	ID     string   `json:"id"`
	Text   string   `json:"text"`
	Voices []string `json:"voices"`
	Roots  []string `json:"-"`
}
type OwnerEvidence struct {
	MappingID string `json:"mapping_id"`
	Revision  int64  `json:"revision"`
	VoiceID   string `json:"voice_id"`
}
type Context struct {
	OwnerIdentities []Identity        `json:"-"`
	OwnerEvidence   []OwnerEvidence   `json:"-"`
	Cues            []Cue             `json:"cues"`
	Voices          []string          `json:"voices"`
	Identities      []Identity        `json:"identities"`
	Owners          map[string]string `json:"owner_mappings"`
	Acoustic        AcousticEvidence  `json:"-"`
}
type Reference struct {
	Start       int    `json:"start_byte"`
	End         int    `json:"end_byte"`
	CueID       string `json:"cue_id"`
	QuoteDigest string `json:"quote_digest"`
}
type Hypothesis struct {
	ID           string      `json:"id"`
	Class        string      `json:"class"`
	Name         string      `json:"name"`
	Aliases      []string    `json:"aliases"`
	Language     string      `json:"language"`
	Role         string      `json:"role"`
	Relationship string      `json:"relationship"`
	VoiceID      string      `json:"voice_id"`
	References   []Reference `json:"references"`
	Roots        []string    `json:"roots"`
	Uncertain    bool        `json:"uncertain"`
	Attributable bool        `json:"attributable"`
}
type AcousticDecision struct {
	VoiceID       string   `json:"voice_id"`
	SpeakerID     string   `json:"speaker_id"`
	State         string   `json:"state"`
	Score         *float64 `json:"score"`
	RunnerUpScore *float64 `json:"runner_up_score"`
	Independent   bool     `json:"independent"`
	Reason        string   `json:"reason"`
	ProfileID     string   `json:"profile_id"`
}
type AcousticEvidence struct {
	Availability     string             `json:"availability"`
	WorkID           string             `json:"work_id"`
	Digest           string             `json:"digest"`
	Decisions        []AcousticDecision `json:"decisions"`
	EligibleProfiles []string           `json:"eligible_profiles"`
}
type Coverage struct {
	Total    int `json:"total_cues"`
	Selected int `json:"selected_cues"`
	Omitted  int `json:"omitted_cues"`
	Calls    int `json:"calls"`
}
type Provenance struct {
	Adapter     string `json:"adapter"`
	Model       string `json:"model"`
	Route       string `json:"route"`
	InputDigest string `json:"input_digest"`
}
type Aggregate struct {
	OwnerEvidence []OwnerEvidence  `json:"owner_evidence,omitempty"`
	Version       int              `json:"version"`
	Coverage      Coverage         `json:"coverage"`
	Provenance    Provenance       `json:"provenance"`
	Hypotheses    []Hypothesis     `json:"hypotheses"`
	Acoustic      AcousticEvidence `json:"acoustic"`
	Policy        Policy           `json:"policy"`
	Decisions     []Decision       `json:"decisions"`
}
type Policy struct {
	MinimumRoots  int      `json:"minimum_contextual_roots,omitempty"`
	MinimumScore  *float64 `json:"minimum_acoustic_score,omitempty"`
	MinimumMargin *float64 `json:"minimum_acoustic_margin,omitempty"`
}

func NormalizePolicy(p Policy) (Policy, error) {
	if p.MinimumRoots == 0 {
		p.MinimumRoots = 1
	}
	if p.MinimumScore == nil {
		v := 0.8
		p.MinimumScore = &v
	}
	if p.MinimumMargin == nil {
		v := 0.1
		p.MinimumMargin = &v
	}
	if p.MinimumRoots < 1 || p.MinimumRoots > 16 || !unit(*p.MinimumScore) || !unit(*p.MinimumMargin) {
		return p, contracts.Fail("invalid_request")
	}
	return p, nil
}
func unit(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0 && n <= 1 }

type Candidate struct {
	SpeakerID *string `json:"speaker_id"`
	Name      string  `json:"name"`
}
type Decision struct {
	VoiceID        string      `json:"voice_id"`
	Outcome        string      `json:"outcome"`
	SpeakerID      *string     `json:"speaker_id"`
	IdentityStatus string      `json:"identity_status"`
	Name           string      `json:"name"`
	Alternatives   []Candidate `json:"alternatives"`
	Reasons        []string    `json:"reasons"`
	EvidenceIDs    []string    `json:"evidence_ids"`
	Roots          []string    `json:"roots"`
}

func Digest(v any) string  { b, _ := json.Marshal(v); return Hash(b) }
func Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func Name(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
func textOK(s string, required bool) bool {
	return utf8.ValidString(s) && len(s) <= 512 && (!required || strings.TrimSpace(s) != "") && !strings.ContainsRune(s, 0)
}
func unique(v []string) []string {
	out := []string{}
	set := map[string]bool{}
	for _, x := range v {
		if !set[x] {
			out = append(out, x)
			set[x] = true
		}
	}
	sort.Strings(out)
	return out
}
func classOK(s string) bool {
	switch s {
	case "self-identification", "context-association", "mention", "quotation":
		return true
	}
	return false
}
func ValidHypothesis(h Hypothesis) bool {
	if h.ID == "" || !classOK(h.Class) || !textOK(h.Name, true) || !textOK(h.Language, false) || !textOK(h.Role, false) || !textOK(h.Relationship, false) || len(h.Aliases) > 16 || len(h.References) < 1 || len(h.References) > MaxReferences || len(h.Roots) > MaxReferences {
		return false
	}
	for _, a := range h.Aliases {
		if !textOK(a, true) {
			return false
		}
	}
	copy := h
	copy.ID = ""
	return h.ID == Digest(copy)
}
func ValidAggregate(a Aggregate) bool {
	if a.Version != Version || a.Coverage.Total < 0 || a.Coverage.Selected < 0 || a.Coverage.Selected > 500 || a.Coverage.Selected+a.Coverage.Omitted != a.Coverage.Total || a.Coverage.Omitted < 0 || a.Coverage.Calls < 0 || a.Coverage.Calls > 20 || len(a.Hypotheses) > MaxObservations || len(a.Decisions) > 1000 {
		return false
	}
	seen := map[string]bool{}
	for _, h := range a.Hypotheses {
		if !ValidHypothesis(h) || seen[h.ID] {
			return false
		}
		seen[h.ID] = true
	}
	p, e := NormalizePolicy(a.Policy)
	if e != nil || Digest(p) != Digest(a.Policy) {
		return false
	}
	b, e := json.Marshal(a)
	return e == nil && len(b) <= MaxBytes
}
