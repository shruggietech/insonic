// SPDX-License-Identifier: Apache-2.0
package models

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Compatibility struct {
	Adapter         string `json:"adapter"`
	ContractVersion string `json:"contract_version"`
	Architecture    string `json:"architecture"`
	RuntimeFormat   string `json:"runtime_format"`
	SampleRates     []int  `json:"sample_rates,omitempty"`
	Channels        []int  `json:"channels,omitempty"`
}

func modelText(s string, n int) bool {
	return len(s) > 0 && len(s) <= n && utf8.ValidString(s) && s == strings.TrimSpace(s) && !strings.ContainsFunc(s, unicode.IsControl)
}
func (c Compatibility) Validate() error {
	if !modelText(c.Adapter, 128) || !modelText(c.ContractVersion, 64) || !modelText(c.Architecture, 128) || !modelText(c.RuntimeFormat, 128) || len(c.SampleRates) > 32 || len(c.Channels) > 32 {
		return contracts.Fail("invalid_request")
	}
	for _, set := range []struct {
		items []int
		max   int
	}{{c.SampleRates, 768000}, {c.Channels, 64}} {
		seen := map[int]bool{}
		for _, v := range set.items {
			if v < 1 || v > set.max || seen[v] {
				return contracts.Fail("invalid_request")
			}
			seen[v] = true
		}
	}
	return nil
}

var modelRolePattern = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._/-]*$`)

func ValidRole(role string) bool {
	if len(role) == 0 || len(role) > 128 || !modelRolePattern.MatchString(role) || path.Clean(role) != role || strings.HasSuffix(role, "/") {
		return false
	}
	for _, part := range strings.Split(role, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".") {
			return false
		}
		n := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if n == "CON" || n == "PRN" || n == "AUX" || n == "NUL" || len(n) == 4 && (strings.HasPrefix(n, "COM") || strings.HasPrefix(n, "LPT")) && n[3] >= '1' && n[3] <= '9' {
			return false
		}
	}
	return true
}
func DefaultAdapter(operation string) (string, string) {
	switch operation {
	case "transcription":
		return "faster-whisper", "1"
	case "diarization":
		return "pyannote", "1"
	}
	return "", ""
}
func CheckCompatibility(m Manifest, operation, adapter, version string) error {
	if m.Validate() != nil {
		return contracts.Fail("invalid_request")
	}
	capable := false
	for _, v := range m.Capabilities {
		capable = capable || v == operation
	}
	if !capable {
		return contracts.Fail("unsupported_capability")
	}
	if adapter == "" {
		adapter, version = DefaultAdapter(operation)
	}
	if version != "1" || adapter != "faster-whisper" && adapter != "pyannote" || adapter == "faster-whisper" && operation != "transcription" || adapter == "pyannote" && operation != "diarization" {
		return contracts.Fail("unsupported_capability")
	}
	required := []string{"config.json", "model.bin", "tokenizer.json", "vocabulary.txt"}
	arch, format := "whisper", "ctranslate2"
	if adapter == "pyannote" {
		required = []string{"config.yaml", "embedding/pytorch_model.bin", "segmentation/pytorch_model.bin", "plda/plda.npz", "plda/xvec_transform.npz"}
		arch, format = "pyannote", "torch"
	}
	roles := map[string]string{}
	for _, f := range m.Files {
		key := strings.ToLower(f.Role)
		if !ValidRole(f.Role) || roles[key] != "" {
			return contracts.Fail("invalid_request")
		}
		roles[key] = f.Role
	}
	for _, v := range required {
		if roles[v] != v {
			return contracts.Fail("model_unavailable")
		}
	}
	if c := m.Compatibility; c != nil {
		if c.Adapter != adapter || c.ContractVersion != version || c.Architecture != arch || c.RuntimeFormat != format {
			return contracts.Fail("unsupported_capability")
		}
		for _, set := range []struct {
			items    []int
			required int
		}{{c.SampleRates, 16000}, {c.Channels, 1}} {
			if len(set.items) == 0 {
				continue
			}
			found := false
			for _, v := range set.items {
				found = found || v == set.required
			}
			if !found {
				return contracts.Fail("unsupported_capability")
			}
		}
	}
	return nil
}
