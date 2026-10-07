// SPDX-License-Identifier: Apache-2.0
// Package speakers resolves current evidence and compiles explicit context.
package speakers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Diagnostic struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}
type OmittedHint struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}
type CompiledContext struct {
	Hints            []string      `json:"hints"`
	Joined           string        `json:"joined"`
	Bytes            int           `json:"bytes"`
	Budget           int           `json:"budget"`
	Digest           string        `json:"digest"`
	SnapshotDigest   string        `json:"snapshot_digest"`
	Revision         int64         `json:"revision"`
	Diagnostics      []Diagnostic  `json:"diagnostics"`
	Omitted          []OmittedHint `json:"omitted"`
	OmittedTotal     int           `json:"omitted_total"`
	OmittedTruncated bool          `json:"omitted_truncated"`
}
type hintCandidate struct{ id, kind, text, state, language, scope string }

func digest(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func CompileContext(snapshot catalog.ContextSnapshot, budget int, supported bool) (out CompiledContext, e error) {
	if budget < 0 || budget > 8192 || len(snapshot.ExtraHints) > 1024 {
		return out, contracts.Fail("invalid_request")
	}
	out = CompiledContext{Hints: []string{}, Budget: budget, Revision: snapshot.Revision, Diagnostics: []Diagnostic{}, Omitted: []OmittedHint{}}
	// Sorting copies leaves the caller's frozen snapshot unchanged.
	snapshot.Filter.SpeakerIDs = append([]string{}, snapshot.Filter.SpeakerIDs...)
	sort.Strings(snapshot.Filter.SpeakerIDs)
	snapshot.Speakers = append([]catalog.SpeakerIdentity{}, snapshot.Speakers...)
	sort.Slice(snapshot.Speakers, func(i, j int) bool { return snapshot.Speakers[i].Speaker.ID < snapshot.Speakers[j].Speaker.ID })
	snapshot.Terms = append([]catalog.Term{}, snapshot.Terms...)
	sort.Slice(snapshot.Terms, func(i, j int) bool { return snapshot.Terms[i].ID < snapshot.Terms[j].ID })
	candidates := []hintCandidate{}
	for i, text := range snapshot.ExtraHints {
		candidates = append(candidates, hintCandidate{id: integer(i), kind: "configured", text: text, state: "active"})
	}
	for i, identity := range snapshot.Speakers {
		sp := identity.Speaker
		candidates = append(candidates, hintCandidate{sp.ID, "speaker", sp.Name, sp.State, "", ""})
		aliases := append([]catalog.SpeakerAlias{}, identity.Aliases...)
		sort.Slice(aliases, func(i, j int) bool { return aliases[i].ID < aliases[j].ID })
		snapshot.Speakers[i].Aliases = aliases
		for _, alias := range aliases {
			state := alias.State
			if sp.State != "active" {
				state = "inactive"
			}
			candidates = append(candidates, hintCandidate{alias.ID, "alias", alias.Text, state, alias.Language, alias.Scope})
		}
	}
	for _, term := range snapshot.Terms {
		state := term.State
		if term.SpeakerID != nil {
			for _, identity := range snapshot.Speakers {
				if identity.Speaker.ID == *term.SpeakerID && identity.Speaker.State != "active" {
					state = "inactive"
				}
			}
		}
		if term.AliasID != nil {
			for _, identity := range snapshot.Speakers {
				for _, a := range identity.Aliases {
					if a.ID == *term.AliasID && (a.State != "active" || identity.Speaker.State != "active") {
						state = "inactive"
					}
				}
			}
		}
		candidates = append(candidates, hintCandidate{term.ID, "term", term.Canonical, state, term.Language, term.Context})
		var variants []string
		if json.Unmarshal(term.Variants, &variants) != nil {
			return out, contracts.Fail("invalid_request")
		}
		for _, text := range variants {
			candidates = append(candidates, hintCandidate{term.ID, "variant", text, state, term.Language, term.Context})
		}
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return out, contracts.Fail("invalid_request")
	}
	if len(raw) > 1<<20 {
		return out, contracts.Fail("input_limit")
	}
	out.SnapshotDigest = digest(raw)
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, c := range candidates {
		reason := ""
		switch {
		case c.state != "active":
			reason = "inactive"
		case c.language != "" && c.language != snapshot.Filter.Language:
			reason = "language"
		case c.scope != "" && c.scope != snapshot.Filter.Context:
			reason = "scope"
		case !utf8.ValidString(c.text) || strings.TrimSpace(c.text) == "" || strings.IndexFunc(c.text, unicode.IsControl) >= 0:
			reason = "invalid"
		case seen[c.text]:
			reason = "duplicate"
		case !supported:
			reason = "unsupported"
		default:
			needed := len(c.text)
			if len(out.Hints) > 0 {
				needed += 2
			}
			if out.Bytes+needed > budget {
				reason = "budget"
			} else {
				out.Hints = append(out.Hints, c.text)
				out.Bytes += needed
				seen[c.text] = true
			}
		}
		if reason != "" {
			out.OmittedTotal++
			if len(out.Omitted) < 2048 {
				out.Omitted = append(out.Omitted, OmittedHint{c.id, c.kind, reason})
			} else {
				out.OmittedTruncated = true
			}
			counts[reason]++
		}
	}
	reasons := []string{}
	for reason := range counts {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	for _, reason := range reasons {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{"context_" + reason, counts[reason]})
	}
	out.Joined = strings.Join(out.Hints, ", ")
	raw, _ = json.Marshal(out.Hints)
	out.Digest = digest(raw)
	return
}
