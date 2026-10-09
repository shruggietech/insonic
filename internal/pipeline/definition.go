// SPDX-License-Identifier: Apache-2.0
// Package pipeline validates frozen processing elections and configured hosted
// workers. Inspecting definitions performs no network requests or inference.
package pipeline

import (
	"encoding/json"
	"net"
	"net/url"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/voicemodels"
)

type Definition struct {
	Recognition     Stage                    `json:"recognition,omitzero"`
	Diarization     Stage                    `json:"diarization,omitzero"`
	Quality         processing.QualityConfig `json:"quality,omitzero"`
	SpeakerTraining json.RawMessage          `json:"speaker_training,omitempty"`
}
type Stage struct {
	Adapter       string                        `json:"adapter"`
	Version       string                        `json:"contract_version"`
	Mode          string                        `json:"mode"`
	ModelID       string                        `json:"model_id,omitempty"`
	RemoteModel   string                        `json:"remote_model,omitempty"`
	Endpoint      string                        `json:"endpoint,omitempty"`
	CredentialID  string                        `json:"credential_id,omitempty"`
	Capabilities  []string                      `json:"capabilities,omitempty"`
	SupportsHints *bool                         `json:"supports_hints,omitempty"`
	MaxHintBytes  int                           `json:"max_hint_bytes,omitempty"`
	Limits        Limits                        `json:"limits"`
	Recognition   processing.RecognitionOptions `json:"recognition,omitempty"`
	Diarization   processing.DiarizationOptions `json:"diarization,omitempty"`
}
type Limits struct {
	MaxAudioBytes    int64 `json:"max_audio_bytes,omitempty"`
	MaxResponseBytes int64 `json:"max_response_bytes,omitempty"`
	TimeoutMS        int64 `json:"timeout_ms,omitempty"`
}
type Override struct {
	Recognition *Stage                    `json:"recognition,omitempty"`
	Diarization *Stage                    `json:"diarization,omitempty"`
	Quality     *processing.QualityConfig `json:"quality,omitempty"`
}
type Capability struct {
	Adapter       string   `json:"adapter"`
	Version       string   `json:"contract_version"`
	Operations    []string `json:"operations"`
	SupportsHints bool     `json:"supports_hints"`
	MaxHintBytes  int      `json:"max_hint_bytes"`
}

func NormalizeLimits(l Limits) (Limits, error) {
	if l.MaxAudioBytes == 0 {
		l.MaxAudioBytes = 128 << 20
	}
	if l.MaxResponseBytes == 0 {
		l.MaxResponseBytes = 16 << 20
	}
	if l.TimeoutMS == 0 {
		l.TimeoutMS = 600000
	}
	if l.MaxAudioBytes < 1 || l.MaxAudioBytes > 64<<30 || l.MaxResponseBytes < 1 || l.MaxResponseBytes > 16<<20 || l.TimeoutMS < 1 || l.TimeoutMS > 86400000 {
		return l, contracts.Fail("invalid_request")
	}
	return l, nil
}
func endpoint(raw, credential string) error {
	u, e := url.Parse(raw)
	if e != nil || !u.IsAbs() || u.Hostname() == "" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsFunc(raw, unicode.IsSpace) || u.Scheme != "https" && u.Scheme != "http" {
		return contracts.Fail("invalid_request")
	}
	if credential != "" && u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if !strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
			return contracts.Fail("invalid_request")
		}
	}
	if credential != "" && !contracts.ValidID(credential) {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func Capabilities(s Stage) (Capability, error) {
	c := Capability{Adapter: s.Adapter, Version: s.Version, Operations: []string{}}
	if s.Version != "1" {
		return c, contracts.Fail("incompatible_version")
	}
	switch s.Adapter {
	case "faster-whisper":
		if s.Mode != "local" {
			return c, contracts.Fail("unsupported_capability")
		}
		c.Operations = []string{"transcription"}
		c.SupportsHints = true
		c.MaxHintBytes = 200
	case "pyannote":
		if s.Mode != "local" {
			return c, contracts.Fail("unsupported_capability")
		}
		c.Operations = []string{"diarization"}
	case "insonic-http":
		if s.Mode != "hosted" || len(s.Capabilities) < 1 || len(s.Capabilities) > 2 {
			return c, contracts.Fail("unsupported_capability")
		}
		seen := map[string]bool{}
		for _, op := range s.Capabilities {
			if op != "transcription" && op != "diarization" || seen[op] {
				return c, contracts.Fail("unsupported_capability")
			}
			seen[op] = true
			c.Operations = append(c.Operations, op)
		}
		c.SupportsHints = s.SupportsHints != nil && *s.SupportsHints
		c.MaxHintBytes = s.MaxHintBytes
		if c.SupportsHints && (c.MaxHintBytes < 1 || c.MaxHintBytes > 8192 || !seen["transcription"]) || !c.SupportsHints && c.MaxHintBytes != 0 {
			return c, contracts.Fail("invalid_request")
		}
	default:
		return c, contracts.Fail("unsupported_capability")
	}
	return c, nil
}
func ValidateStage(s Stage, operation string) error {
	// Definition.Quality is the sole pipeline diagnostic election. A second
	// stage-level quality block would otherwise be overwritten by the parent.
	if s.Diarization.Quality != nil {
		return contracts.Fail("invalid_request")
	}
	if s.Recognition.ExecutionLimits != nil || s.Diarization.ExecutionLimits != nil {
		return contracts.Fail("invalid_request")
	}
	if s.Recognition.ExpectedModelDigest != "" || s.Diarization.ExpectedModelDigest != "" {
		return contracts.Fail("invalid_request")
	}
	c, e := Capabilities(s)
	if e != nil {
		return e
	}
	found := false
	for _, op := range c.Operations {
		found = found || op == operation
	}
	if !found {
		return contracts.Fail("unsupported_capability")
	}
	if _, e = NormalizeLimits(s.Limits); e != nil {
		return e
	}
	if s.Mode == "local" {
		if !models.ValidateReference(s.ModelID) || s.RemoteModel != "" || s.Endpoint != "" || s.CredentialID != "" || len(s.Capabilities) > 0 || s.SupportsHints != nil || s.MaxHintBytes != 0 {
			return contracts.Fail("invalid_request")
		}
	} else {
		if s.ModelID != "" || s.RemoteModel == "" || len(s.RemoteModel) > 256 || !utf8.ValidString(s.RemoteModel) || strings.ContainsFunc(s.RemoteModel, unicode.IsControl) || s.RemoteModel != strings.TrimSpace(s.RemoteModel) || endpoint(s.Endpoint, s.CredentialID) != nil || s.Recognition.Device != "" || s.Diarization.Device != "" {
			return contracts.Fail("invalid_request")
		}
	}
	if operation == "transcription" {
		if s.Diarization != (processing.DiarizationOptions{}) {
			return contracts.Fail("invalid_request")
		}
		if _, e = processing.ValidateRecognitionOptions(s.Recognition, c.MaxHintBytes); e != nil {
			return e
		}
	} else if operation == "diarization" {
		if s.Recognition.Device != "" || s.Recognition.Language != "" || len(s.Recognition.Hints) > 0 || s.Recognition.ContextDigest != "" {
			return contracts.Fail("invalid_request")
		}
		if e = processing.ValidateDiarizationOptions(s.Diarization); e != nil {
			return e
		}
	} else {
		return contracts.Fail("unsupported_capability")
	}
	return nil
}
func Validate(preset string, d Definition) error {
	if preset != "local" && preset != "connected" && preset != "custom" {
		return contracts.Fail("invalid_request")
	}
	if len(d.SpeakerTraining) > 0 {
		if voicemodels.ValidateTrainingConfiguration(d.SpeakerTraining) != nil {
			return contracts.Fail("invalid_request")
		}
		var training struct {
			Adapter struct {
				Mode string `json:"mode"`
			} `json:"adapter"`
		}
		if json.Unmarshal(d.SpeakerTraining, &training) != nil || preset == "local" && training.Adapter.Mode != "local" || preset == "connected" && training.Adapter.Mode != "hosted" {
			return contracts.Fail("invalid_request")
		}
		if !HasProcessing(d) {
			return nil
		}
	}
	if e := ValidateStage(d.Recognition, "transcription"); e != nil {
		return e
	}
	if e := ValidateStage(d.Diarization, "diarization"); e != nil {
		return e
	}
	if preset == "local" && (d.Recognition.Mode != "local" || d.Diarization.Mode != "local") || preset == "connected" && (d.Recognition.Mode != "hosted" || d.Diarization.Mode != "hosted") {
		return contracts.Fail("invalid_request")
	}
	return processing.ValidateQuality(d.Quality)
}
func HasProcessing(d Definition) bool {
	return !reflect.ValueOf(d.Recognition).IsZero() || !reflect.ValueOf(d.Diarization).IsZero()
}
func Elect(preset string, d Definition, o Override) (Definition, error) {
	if !HasProcessing(d) {
		return Definition{}, contracts.Fail("unsupported_capability")
	}
	if o.Recognition != nil {
		d.Recognition = *o.Recognition
	}
	if o.Diarization != nil {
		d.Diarization = *o.Diarization
	}
	if o.Quality != nil {
		d.Quality = *o.Quality
	}
	if e := Validate(preset, d); e != nil {
		return Definition{}, e
	}
	d.Recognition.Limits, _ = NormalizeLimits(d.Recognition.Limits)
	d.Diarization.Limits, _ = NormalizeLimits(d.Diarization.Limits)
	// JSON copy also detaches all pointers, hint/capability arrays and thresholds.
	raw, e := json.Marshal(d)
	if e != nil {
		return Definition{}, contracts.Fail("invalid_request")
	}
	var copy Definition
	if json.Unmarshal(raw, &copy) != nil {
		return Definition{}, contracts.Fail("invalid_request")
	}
	return copy, nil
}
