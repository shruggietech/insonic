// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shruggietech/insonic/internal/contracts"
)

func HintsDigest(hints []string) string {
	if hints == nil {
		hints = []string{}
	}
	raw, _ := json.Marshal(hints)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// ValidateRecognitionOptions bounds exact joined context. Compilation must
// diagnose omissions before this boundary; adapters never truncate silently.
func ValidateRecognitionOptions(o RecognitionOptions, maxHintBytes int) (RecognitionOptions, error) {
	if o.ExpectedModelDigest != "" && !pinnedDigest.MatchString(o.ExpectedModelDigest) {
		return o, contracts.Fail("invalid_request")
	}
	if o.Device != "" && o.Device != "cpu" && o.Device != "cuda" || o.Language != "" && o.Language != "auto" && !languagePattern.MatchString(o.Language) || maxHintBytes < 0 || maxHintBytes > 8192 || len(o.Hints) > 1024 {
		return o, contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	for _, hint := range o.Hints {
		if hint == "" || hint != strings.TrimSpace(hint) || !utf8.ValidString(hint) || strings.ContainsFunc(hint, unicode.IsControl) || seen[hint] {
			return o, contracts.Fail("invalid_request")
		}
		seen[hint] = true
	}
	if len(strings.Join(o.Hints, ", ")) > maxHintBytes {
		return o, contracts.Fail("input_limit")
	}
	digest := HintsDigest(o.Hints)
	if o.ContextDigest != "" && o.ContextDigest != digest {
		return o, contracts.Fail("conflict")
	}
	o.ContextDigest = digest
	o.Hints = append([]string(nil), o.Hints...)
	return o, nil
}

func ValidateDiarizationOptions(o DiarizationOptions) error {
	if o.ExpectedModelDigest != "" && !pinnedDigest.MatchString(o.ExpectedModelDigest) {
		return contracts.Fail("invalid_request")
	}
	if o.Device != "" && o.Device != "cpu" && o.Device != "cuda" || o.MinSpeakers < 0 || o.MaxSpeakers < 0 || o.MinSpeakers > 64 || o.MaxSpeakers > 64 || o.MaxSpeakers > 0 && o.MinSpeakers > o.MaxSpeakers {
		return contracts.Fail("invalid_request")
	}
	if o.Quality != nil {
		return ValidateQuality(*o.Quality)
	}
	return nil
}
