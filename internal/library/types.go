// SPDX-License-Identifier: Apache-2.0
package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

type PinnedFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Tool struct {
	Path         string       `json:"path"`
	SHA256       string       `json:"sha256"`
	Version      string       `json:"version"`
	SupportFiles []PinnedFile `json:"support_files,omitempty"`
	Interpreter  *PinnedFile  `json:"interpreter,omitempty"`
}
type Tools struct {
	FFmpeg   Tool   `json:"ffmpeg,omitzero"`
	Kind     string `json:"kind,omitempty"`
	Version  string `json:"schema_version,omitempty"`
	ExifTool Tool   `json:"exiftool"`
	FFprobe  Tool   `json:"ffprobe"`
}
type Options struct {
	TranscriptMaxBytes  *int64 `json:"transcript_max_bytes,omitempty"`
	TranscriptTimeoutMS *int64 `json:"transcript_timeout_ms,omitempty"`
	Attribution         string `json:"attribution,omitempty"`
	TranscriptFormat    string `json:"transcript_format,omitempty"`
	ReplaceTranscript   *bool  `json:"replace_transcript,omitempty"`
	SubtitleStreamIndex *int   `json:"subtitle_stream_index,omitempty"`
	SubtitleLanguage    string `json:"subtitle_language,omitempty"`
	DiarizationModelID  string `json:"diarization_model_id,omitempty"`

	LocalHTTP            *bool            `json:"local_http,omitempty"`
	AcquisitionMaxBytes  *int64           `json:"acquisition_max_bytes,omitempty"`
	AcquisitionTimeoutMS *int64           `json:"acquisition_timeout_ms,omitempty"`
	Copy                 *bool            `json:"copy,omitempty"`
	OriginatedAt         string           `json:"originated_at,omitempty"`
	OriginatedOn         string           `json:"originated_on,omitempty"`
	OriginatedEarliest   *catalog.Instant `json:"originated_earliest,omitempty"`
	OriginatedLatest     *catalog.Instant `json:"originated_latest,omitempty"`
	Timezone             string           `json:"timezone,omitempty"`
	DSTFold              string           `json:"dst_fold,omitempty"`
	DSTGap               string           `json:"dst_gap,omitempty"`
	Preset               string           `json:"preset,omitempty"`
	DatePrecedence       string           `json:"date_precedence,omitempty"`
	Extensions           json.RawMessage  `json:"extensions,omitempty"`
}
type Item struct {
	ExpectedRevision          *int64 `json:"expected_revision,omitempty"`
	ExpectedRecordingRevision *int64 `json:"expected_recording_revision,omitempty"`
	TargetError               string `json:"target_error,omitempty"`
	Transcript                string `json:"transcript,omitempty"`
	Record                    string `json:"record,omitempty"`
	TranscriptCredentialID    string `json:"transcript_credential_id,omitempty"`
	TranscriptAdapter         string `json:"transcript_adapter,omitempty"`
	AcceptedReceipt           string `json:"accepted_receipt,omitempty"`

	Source             string `json:"source,omitempty"`
	Subtitle           string `json:"subtitle,omitempty"`
	Title              string `json:"title,omitempty"`
	CredentialID       string `json:"credential_id,omitempty"`
	AcquisitionAdapter string `json:"acquisition_adapter,omitempty"`
	NewEntry           bool   `json:"new_entry,omitempty"`
	Options
}
type ImportRequest struct {
	RequestDigest string          `json:"request_digest,omitempty"`
	Kind          string          `json:"kind,omitempty"`
	Version       string          `json:"schema_version,omitempty"`
	Defaults      Options         `json:"defaults,omitempty"`
	Items         []Item          `json:"items"`
	Extensions    json.RawMessage `json:"extensions,omitempty"`
}
type RefreshRequest struct {
	MediaID string  `json:"media_id"`
	Options Options `json:"options,omitempty"`
}
type ItemResult struct {
	Notices           []string `json:"notices,omitempty"`
	RecordingRevision int64    `json:"recording_revision,omitempty"`
	DocumentDigest    string   `json:"document_digest,omitempty"`
	Ordinal           int      `json:"ordinal"`
	MediaID           string   `json:"media_id,omitempty"`
	Digest            string   `json:"digest,omitempty"`
	State             string   `json:"state"`
	CaptureState      string   `json:"capture_state,omitempty"`
	DateState         string   `json:"date_state,omitempty"`
	DateUnresolved    bool     `json:"date_unresolved,omitempty"`
	SubtitleState     string   `json:"subtitle_state,omitempty"`
	Error             string   `json:"error,omitempty"`
	QueuedJobIDs      []string `json:"queued_job_ids"`
}
type ImportResult struct {
	Items            []ItemResult `json:"items"`
	Partial          bool         `json:"partial"`
	CapturedTimezone string       `json:"captured_timezone"`
}

func marshal(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
func DerivedID(op, label string) string {
	sum := sha256.Sum256([]byte(op + "\x00" + label))
	sum[6] = sum[6]&15 | 64
	sum[8] = sum[8]&63 | 128
	s := hex.EncodeToString(sum[:16])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
func merged(base, item Options) Options {
	if item.TranscriptMaxBytes != nil {
		base.TranscriptMaxBytes = item.TranscriptMaxBytes
	}
	if item.TranscriptTimeoutMS != nil {
		base.TranscriptTimeoutMS = item.TranscriptTimeoutMS
	}
	if item.ReplaceTranscript != nil {
		base.ReplaceTranscript = item.ReplaceTranscript
	}
	if item.SubtitleStreamIndex != nil {
		base.SubtitleStreamIndex = item.SubtitleStreamIndex
	}
	if item.Attribution != "" {
		base.Attribution = item.Attribution
	}
	if item.TranscriptFormat != "" {
		base.TranscriptFormat = item.TranscriptFormat
	}
	if item.SubtitleLanguage != "" {
		base.SubtitleLanguage = item.SubtitleLanguage
	}
	if item.DiarizationModelID != "" {
		base.DiarizationModelID = item.DiarizationModelID
	}

	if item.LocalHTTP != nil {
		base.LocalHTTP = item.LocalHTTP
	}
	if item.AcquisitionMaxBytes != nil {
		base.AcquisitionMaxBytes = item.AcquisitionMaxBytes
	}
	if item.AcquisitionTimeoutMS != nil {
		base.AcquisitionTimeoutMS = item.AcquisitionTimeoutMS
	}
	if item.Copy != nil {
		base.Copy = item.Copy
	}
	if item.OriginatedAt != "" {
		base.OriginatedAt = item.OriginatedAt
		base.OriginatedOn = ""
		base.OriginatedEarliest = nil
		base.OriginatedLatest = nil
	}
	if item.OriginatedOn != "" {
		base.OriginatedOn = item.OriginatedOn
		base.OriginatedAt = ""
		base.OriginatedEarliest = nil
		base.OriginatedLatest = nil
	}
	if item.OriginatedEarliest != nil || item.OriginatedLatest != nil {
		base.OriginatedEarliest = item.OriginatedEarliest
		base.OriginatedLatest = item.OriginatedLatest
		base.OriginatedAt = ""
		base.OriginatedOn = ""
	}
	if item.Timezone != "" {
		base.Timezone = item.Timezone
	}
	if item.DSTFold != "" {
		base.DSTFold = item.DSTFold
	}
	if item.DSTGap != "" {
		base.DSTGap = item.DSTGap
	}
	if item.Preset != "" {
		base.Preset = item.Preset
	}
	if item.DatePrecedence != "" {
		base.DatePrecedence = item.DatePrecedence
	}
	if len(item.Extensions) > 0 {
		base.Extensions = item.Extensions
	}
	return base
}
func validOptions(o Options) bool {
	if o.TranscriptMaxBytes != nil && (*o.TranscriptMaxBytes < 1 || *o.TranscriptMaxBytes > 16<<20) || o.TranscriptTimeoutMS != nil && (*o.TranscriptTimeoutMS < 1 || *o.TranscriptTimeoutMS > 600000) || o.SubtitleStreamIndex != nil && *o.SubtitleStreamIndex < 0 {
		return false
	}
	if o.Attribution != "" && o.Attribution != "auto" && o.Attribution != "native" && o.Attribution != "off" && o.Attribution != "diarize" {
		return false
	}
	if o.DiarizationModelID != "" && !contracts.ValidID(o.DiarizationModelID) {
		return false
	}
	if o.TranscriptFormat != "" && o.TranscriptFormat != "cueson" && o.TranscriptFormat != "srt" && o.TranscriptFormat != "vtt" && o.TranscriptFormat != "ass" && o.TranscriptFormat != "ssa" {
		return false
	}

	if (o.AcquisitionMaxBytes != nil && *o.AcquisitionMaxBytes <= 0) || (o.AcquisitionTimeoutMS != nil && (*o.AcquisitionTimeoutMS <= 0 || *o.AcquisitionTimeoutMS > 9223372036854)) {
		return false
	}
	if o.OriginatedEarliest != nil || o.OriginatedLatest != nil {
		if o.OriginatedAt != "" || o.OriginatedOn != "" || !validBound(o.OriginatedEarliest) || !validBound(o.OriginatedLatest) || (o.OriginatedEarliest != nil && o.OriginatedLatest != nil && o.OriginatedEarliest.UnixNS > o.OriginatedLatest.UnixNS) {
			return false
		}
	}
	return !(o.OriginatedAt != "" && o.OriginatedOn != "") && (o.DSTFold == "" || o.DSTFold == "earlier" || o.DSTFold == "later") && (o.DSTGap == "" || o.DSTGap == "shift-forward") && (o.DatePrecedence == "" || o.DatePrecedence == "owner-first" || o.DatePrecedence == "embedded-first" || o.DatePrecedence == "filesystem-fallback")
}
func hasOrigin(o Options) bool {
	return o.OriginatedAt != "" || o.OriginatedOn != "" || o.OriginatedEarliest != nil || o.OriginatedLatest != nil
}
func PrepareImport(r ImportRequest) (ImportRequest, error) {
	if len(r.Items) == 0 || len(r.Items) > 10000 || !validOptions(r.Defaults) || (r.Kind != "" && r.Kind != "import-manifest") || (r.Version != "" && r.Version != contracts.Version) {
		return r, contracts.Fail("invalid_request")
	}
	if r.Defaults.Timezone == "" || r.Defaults.Timezone == "local" {
		if zone, e := LocalZone(); e == nil {
			r.Defaults.Timezone = zone
		} else {
			r.Defaults.Timezone = "unresolved-local"
		}
	}
	if r.Kind == "" {
		r.Kind = "import-manifest"
	}
	if r.Version == "" {
		r.Version = contracts.Version
	}
	for i := range r.Items {
		item := &r.Items[i]
		if item.ExpectedRevision != nil && (*item.ExpectedRevision < 1 || item.Record == "") || item.ExpectedRecordingRevision != nil && (*item.ExpectedRecordingRevision < 0 || item.Record == "") {
			return r, contracts.Fail("invalid_request")
		}
		effective := merged(r.Defaults, item.Options)
		if effective.Attribution == "diarize" && !contracts.ValidID(effective.DiarizationModelID) {
			return r, contracts.Fail("invalid_request")
		}
		if item.Record != "" && item.Source != "" && item.Source != "<accepted>" {
			if item.Transcript != "" || item.Subtitle != "" {
				return r, contracts.Fail("invalid_request")
			}
			item.Transcript = item.Source
			item.Source = ""
		}

		if (r.Items[i].Source == "" && (r.Items[i].Record == "" || r.Items[i].Transcript == "" && r.Items[i].Subtitle == "")) || !validOptions(r.Items[i].Options) || (r.Items[i].CredentialID != "" && !contracts.ValidID(r.Items[i].CredentialID)) {
			return r, contracts.Fail("invalid_request")
		}
		if hasRemoteScheme(r.Items[i].Source) {
			if _, e := contracts.SourceURL(r.Items[i].Source); e != nil {
				return r, e
			}
			if e := validateAcquisitionTransport(r.Items[i].Source, r.Items[i].CredentialID, merged(r.Defaults, r.Items[i].Options)); e != nil {
				return r, e
			}
		}
		if item.Transcript != "" && item.Subtitle != "" {
			return r, contracts.Fail("invalid_request")
		}
		if item.Transcript == "" {
			item.Transcript = item.Subtitle
			item.Subtitle = ""
		}
		if item.TranscriptCredentialID != "" && !contracts.ValidID(item.TranscriptCredentialID) {
			return r, contracts.Fail("invalid_request")
		}
		if item.AcceptedReceipt != "" && (!contracts.ValidID(item.AcceptedReceipt) || item.Source != "<accepted>") {
			return r, contracts.Fail("invalid_request")
		}
		if hasRemoteScheme(item.Transcript) {
			if _, e := contracts.SourceURL(item.Transcript); e != nil {
				return r, e
			}
			if e := validateAcquisitionTransport(item.Transcript, item.TranscriptCredentialID, transcriptOptions(merged(r.Defaults, item.Options))); e != nil {
				return r, e
			}
		}

		if r.Items[i].Timezone == "local" {
			r.Items[i].Timezone = r.Defaults.Timezone
		}
	}
	if raw := marshal(r); len(raw) > contracts.MaxWorkPayload {
		return r, contracts.Fail("output_limit")
	}
	return r, nil
}

// Recognize the scheme independently of URL parsing so malformed escapes do
// not turn a remote locator into a local path before intent validation.
func hasRemoteScheme(source string) bool {
	source = strings.TrimSpace(source)
	colon := strings.IndexByte(source, ':')
	if colon <= 1 {
		return false
	} // Preserve Windows drive paths.
	for i := 0; i < colon; i++ {
		c := source[i]
		letter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		if !letter && (i == 0 || !(c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.')) {
			return false
		}
	}
	return true
}

func transcriptOptions(options Options) Options {
	max, timeout := int64(16<<20), int64(120000)
	if options.TranscriptMaxBytes != nil {
		max = *options.TranscriptMaxBytes
	}
	if options.TranscriptTimeoutMS != nil {
		timeout = *options.TranscriptTimeoutMS
	}
	options.AcquisitionMaxBytes = &max
	options.AcquisitionTimeoutMS = &timeout
	return options
}

// MergeOptions applies the same explicit per-field precedence for every client.
func MergeOptions(base, item Options) Options { return merged(base, item) }
