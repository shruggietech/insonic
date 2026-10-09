// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const Version = "0.0.0"

// MaxWorkPayload bounds normalized durable input independently of the smaller
// response frame budget. Import manifests and runtime requests share this limit.
const MaxWorkPayload = 8 << 20

// MaxGraphSnapshot bounds the internal immutable reference snapshot path. It is
// not a runtime request/work payload: a whole library can exceed a single job.
const MaxGraphSnapshot = 128 << 20

func ID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("secure randomness unavailable")
	}
	b[6] = b[6]&15 | 64
	b[8] = b[8]&63 | 128
	s := hex.EncodeToString(b[:])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func ValidID(s string) bool { return uuid.MatchString(s) }

// ValidLocalSpeakerID follows the Cueson recording-local token contract.
// Global person identities retain UUID validation.
func ValidLocalSpeakerID(s string) bool {
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) < 1 || utf8.RuneCountInString(s) > 256 {
		return false
	}
	runes := []rune(s)
	if unicode.IsSpace(runes[0]) || unicode.IsSpace(runes[len(runes)-1]) || runes[0] == 0xfeff || runes[len(runes)-1] == 0xfeff {
		return false
	}
	for _, r := range runes {
		if r <= 0x1f || r >= 0x7f && r <= 0x9f || r == 0x2028 || r == 0x2029 || r == 0x061c || r == 0x200e || r == 0x200f || r >= 0x202a && r <= 0x202e || r >= 0x2066 && r <= 0x2069 || r == 0xfeff {
			return false
		}
	}
	return true
}

type Error struct {
	Code        string       `json:"code"`
	Message     string       `json:"message"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func Fail(code string) *Error {
	message := map[string]string{
		"timing_unavailable":     "The selected stream has no usable source-clock timing.",
		"invalid_audio":          "The selected audio is invalid or cannot be decoded.",
		"audio_limit":            "The audio exceeds the configured resource limit.",
		"unsupported_audio":      "The selected audio layout is unsupported.",
		"model_unavailable":      "The elected managed model is unavailable.",
		"unsupported_capability": "The selected adapter does not support this operation.",
		"input_limit":            "The processing input exceeds the configured limit.",
		"invalid_engine_output":  "The local engine returned an invalid result.",
		"invalid_timing":         "The local engine returned invalid source-clock timing.",
		"engine_ci_forbidden":    "Model execution is disabled in CI.",
		"engine_unavailable":     "The configured local engine is unavailable.",
		"unsupported_device":     "The selected processing device is unsupported.",
		"invalid_embedding":      "The local engine returned invalid speaker embeddings.",
		"invalid_request":        "The request is invalid.", "incompatible_version": "The contract version is unsupported.",
		"workspace_mismatch": "The request belongs to another workspace.", "not_found": "The item is unavailable in this workspace.",
		"conflict": "Another owner or attempt already controls this operation.", "unavailable": "The selected dependency is unavailable.",
		"cancelled": "The operation was cancelled.", "output_limit": "The child output limit was exceeded.", "operation_failed": "The operation failed.",
	}[code]
	if message == "" {
		code = "operation_failed"
		message = "The operation failed."
	}
	return &Error{Code: code, Message: message}
}

// Diagnostic is a bounded generated condition, not a child-output envelope.
// Count aggregates all represented occurrences; Pointer identifies the first.
type Diagnostic struct {
	Code    string `json:"code"`
	Pointer string `json:"pointer,omitempty"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

var diagnosticPointer = regexp.MustCompile(`^/(media_timing|cues/(0|[1-9][0-9]*)/speaker_attributions/(0|[1-9][0-9]*))$`)
var subtitleDiagnosticCodes = strings.Fields(`consumer_speaker_attribution_omitted consumer_media_timing_omitted consumer_cue_media_conflict export_loss_limit strict_export_refused subtitle_diagnostic conversion_metadata_omitted conversion_payload_line_degraded conversion_ocr_observation_omitted conversion_speaker_observation_omitted conversion_token_timing_omitted conversion_placement_omitted conversion_nul_degraded conversion_subrip_coordinates_omitted conversion_subrip_font_degraded conversion_subrip_unrecognized_block_omitted conversion_subrip_payload_ambiguous conversion_webvtt_description_omitted conversion_webvtt_metadata_omitted conversion_webvtt_block_omitted conversion_webvtt_identifier_omitted conversion_webvtt_setting_omitted conversion_webvtt_setting_occurrence_omitted conversion_webvtt_voice_degraded conversion_webvtt_markup_degraded conversion_webvtt_inline_timing_omitted conversion_webvtt_entity_ambiguous conversion_scripted_metadata_omitted conversion_scripted_style_field_omitted conversion_scripted_event_field_omitted conversion_scripted_record_omitted conversion_scripted_section_omitted conversion_scripted_attachment_omitted conversion_scripted_override_omitted conversion_scripted_override_degraded conversion_scripted_override_comment_omitted conversion_scripted_drawing_omitted conversion_scripted_centisecond_quantized conversion_scripted_variant_field_omitted conversion_scripted_variant_field_degraded conversion_source_identifier_omitted conversion_speaker_attribution_omitted conversion_media_timing_omitted`)

func safeDiagnosticMessage(code string) string {
	switch code {
	case "consumer_speaker_attribution_omitted", "conversion_speaker_attribution_omitted":
		return "Native subtitle output omits consumer speaker assignments."
	case "consumer_media_timing_omitted", "conversion_media_timing_omitted":
		return "Native subtitle output omits declared media timing."
	case "consumer_cue_media_conflict":
		return "Some source cues fall outside declared media bounds."
	case "export_loss_limit":
		return "Complete native-export diagnostic accounting exceeds the configured limit; no output is published."
	case "strict_export_refused":
		return "Strict native subtitle export refuses a lossy result before publication."
	case "subtitle_diagnostic":
		return "The subtitle operation reported an additional condition."
	}
	if strings.HasSuffix(code, "_omitted") {
		return "Native subtitle conversion omits this source field."
	}
	if strings.HasSuffix(code, "_ambiguous") {
		return "Native subtitle conversion cannot represent this field unambiguously."
	}
	if strings.HasSuffix(code, "_quantized") {
		return "Native subtitle conversion reduces timing precision."
	}
	return "Native subtitle conversion changes this source field."
}

// AggregateDiagnostics preserves occurrence accounting in a bounded response.
// Codes use the current stable vocabulary, pointers are generated cue/media
// locations, and messages are fixed locally so raw child output cannot escape.
func AggregateDiagnostics(input []Diagnostic) []Diagnostic {
	if len(input) > 8192 {
		return []Diagnostic{{Code: "export_loss_limit", Message: safeDiagnosticMessage("export_loss_limit"), Count: 1}}
	}
	result := []Diagnostic{}
	indices := map[string]int{}
	for _, item := range input {
		code := "subtitle_diagnostic"
		for _, allowed := range subtitleDiagnosticCodes {
			if item.Code == allowed {
				code = allowed
				break
			}
		}
		count := item.Count
		if count < 1 || count > 65536 {
			count = 1
		}
		if index, ok := indices[code]; ok {
			if result[index].Count > 65536-count {
				return []Diagnostic{{Code: "export_loss_limit", Message: safeDiagnosticMessage("export_loss_limit"), Count: 1}}
			} else {
				result[index].Count += count
			}
			continue
		}
		pointer := ""
		if len(item.Pointer) <= 256 && diagnosticPointer.MatchString(item.Pointer) {
			pointer = item.Pointer
		}
		indices[code] = len(result)
		result = append(result, Diagnostic{Code: code, Pointer: pointer, Message: safeDiagnosticMessage(code), Count: count})
	}
	return result
}
func WithDiagnostics(err error, input []Diagnostic) *Error {
	if err == nil {
		return nil
	}
	code := "operation_failed"
	var original *Error
	if errors.As(err, &original) {
		code = original.Code
	}
	result := Fail(code)
	result.Diagnostics = AggregateDiagnostics(input)
	return result
}

type Request struct {
	Kind            string          `json:"kind"`
	Version         string          `json:"schema_version"`
	WorkspaceID     string          `json:"workspace_id"`
	RequestID       string          `json:"request_id"`
	Operation       string          `json:"operation"`
	JobID           string          `json:"job_id,omitempty"`
	DurationMS      int             `json:"duration_ms,omitempty"`
	AfterGeneration int64           `json:"after_generation,omitempty"`
	PublicationID   string          `json:"publication_id,omitempty"`
	SourcePath      string          `json:"source_path,omitempty"`
	ArtifactKind    string          `json:"artifact_kind,omitempty"`
	LeaseID         string          `json:"lease_id,omitempty"`
	ReferenceID     string          `json:"reference_id,omitempty"`
	MaxBytes        int64           `json:"max_bytes,omitempty"`
	ItemID          string          `json:"item_id,omitempty"`
	Data            json.RawMessage `json:"data,omitempty"`
}
type Response struct {
	Kind        string `json:"kind"`
	Version     string `json:"schema_version"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	SessionID   string `json:"runtime_session_id,omitempty"`
	Result      any    `json:"result,omitempty"`
	Error       *Error `json:"error,omitempty"`
}

// Implemented streaming storage and typed catalog interfaces live in their
// domain packages; secret values remain behind this separate private boundary.
type SecretProvider interface {
	Resolve(context.Context, string) ([]byte, error)
	Status(context.Context, string) (string, error)
}
type Capability struct {
	AdapterID       string   `json:"adapter_id"`
	ContractVersion string   `json:"contract_version"`
	State           string   `json:"state"`
	Operations      []string `json:"operations"`
}
