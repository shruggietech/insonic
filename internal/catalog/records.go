// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Instant keeps Cueson nanoseconds out of floating-point and native SQL timestamps.
type Instant struct {
	ISO    string `json:"iso"`
	UnixNS int64  `json:"unix_ns"`
}

func (i Instant) valid() bool {
	t, e := time.Parse(time.RFC3339Nano, i.ISO)
	return e == nil && !t.Before(time.Unix(0, math.MinInt64)) && !t.After(time.Unix(0, math.MaxInt64)) && t.UnixNano() == i.UnixNS
}

type Artifact struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
	Kind   string `json:"kind"`
}
type ArtifactLocation struct {
	ID              string  `json:"id"`
	ArtifactID      string  `json:"artifact_id"`
	ProfileID       string  `json:"profile_id"`
	ProfileRevision int64   `json:"profile_revision"`
	Key             string  `json:"key"`
	Version         *string `json:"version"`
	State           string  `json:"state"`
}
type Profile struct {
	ExpectedBackendVersion *string         `json:"expected_backend_version"`
	ID                     string          `json:"id"`
	Revision               int64           `json:"revision"`
	Role                   string          `json:"role"`
	Adapter                string          `json:"adapter"`
	Version                string          `json:"version"`
	Configuration          json.RawMessage `json:"configuration"`
	CredentialID           *string         `json:"credential_id"`
}
type Media struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Class string `json:"class"`
}
type Asset struct {
	ID         string `json:"id"`
	ArtifactID string `json:"artifact_id"`
	Role       string `json:"role"`
	DurationUS int64  `json:"duration_us"`
}
type MediaAsset struct {
	ID      string `json:"id"`
	MediaID string `json:"media_id"`
	AssetID string `json:"asset_id"`
	Role    string `json:"role"`
}
type MetadataSnapshot struct {
	ID               string  `json:"id"`
	AssetID          string  `json:"asset_id"`
	ReportArtifactID *string `json:"report_artifact_id"`
	State            string  `json:"state"`
	Extractor        string  `json:"extractor"`
	Version          string  `json:"version"`
	Captured         Instant `json:"captured"`
}
type Observation struct {
	ID         string          `json:"id"`
	SnapshotID string          `json:"snapshot_id"`
	Family     string          `json:"family"`
	Tag        string          `json:"tag"`
	Instance   int64           `json:"instance"`
	Raw        json.RawMessage `json:"raw"`
	Value      json.RawMessage `json:"value"`
	Unit       *string         `json:"unit"`
	Precision  *string         `json:"precision"`
}
type DateObservation struct {
	ID            string   `json:"id"`
	MediaID       string   `json:"media_id"`
	ObservationID *string  `json:"observation_id"`
	Basis         string   `json:"basis"`
	Literal       string   `json:"literal"`
	Zone          *string  `json:"zone"`
	OffsetSeconds *int64   `json:"offset_seconds"`
	RulesVersion  *string  `json:"rules_version"`
	Precision     string   `json:"precision"`
	Resolved      *Instant `json:"resolved"`
}
type DateSelection struct {
	ID       string  `json:"id"`
	MediaID  string  `json:"media_id"`
	Revision int64   `json:"revision"`
	DateID   *string `json:"date_id"`
	Policy   string  `json:"policy"`
	Reason   string  `json:"reason"`
}
type Speaker struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Segment struct {
	ID             string          `json:"id"`
	Revision       int64           `json:"revision"`
	AssetID        string          `json:"asset_id"`
	SpeakerID      *string         `json:"speaker_id"`
	StartUS        int64           `json:"start_us"`
	EndUS          int64           `json:"end_us"`
	Channel        int64           `json:"channel"`
	Attribution    json.RawMessage `json:"attribution"`
	ClipArtifactID *string         `json:"clip_artifact_id"`
}
type Dataset struct {
	ID                 string          `json:"id"`
	SpeakerID          string          `json:"speaker_id"`
	ManifestArtifactID string          `json:"manifest_artifact_id"`
	Options            json.RawMessage `json:"options"`
}
type DatasetMember struct {
	ID              string `json:"id"`
	DatasetID       string `json:"dataset_id"`
	Ordinal         int64  `json:"ordinal"`
	SegmentID       string `json:"segment_id"`
	SegmentRevision int64  `json:"segment_revision"`
}
type TrainingRun struct {
	ID                    string          `json:"id"`
	DatasetID             string          `json:"dataset_id"`
	SpeakerID             string          `json:"speaker_id"`
	JobID                 string          `json:"job_id"`
	PreparationArtifactID string          `json:"preparation_artifact_id"`
	Adapter               string          `json:"adapter"`
	Options               json.RawMessage `json:"options"`
}
type Model struct {
	ID        string `json:"id"`
	SpeakerID string `json:"speaker_id"`
	Name      string `json:"name"`
}
type ModelVersion struct {
	ID                 string `json:"id"`
	ModelID            string `json:"model_id"`
	RunID              string `json:"run_id"`
	DatasetID          string `json:"dataset_id"`
	ManifestArtifactID string `json:"manifest_artifact_id"`
	Kind               string `json:"kind"`
}
type ModelArtifact struct {
	ID         string `json:"id"`
	VersionID  string `json:"version_id"`
	ArtifactID string `json:"artifact_id"`
	Role       string `json:"role"`
	Format     string `json:"format"`
}
type ModelAssociation struct {
	ID        string `json:"id"`
	ModelID   string `json:"model_id"`
	SpeakerID string `json:"speaker_id"`
	Revision  int64  `json:"revision"`
	Reason    string `json:"reason"`
}
type Records struct {
	Library    []LibraryEntry     `json:"library,omitempty"`
	BaseModels []BaseModelInstall `json:"base_models,omitempty"`
	Works      []Work             `json:"works,omitempty"`
	Cleanups   []Cleanup          `json:"cleanups,omitempty"`

	Profiles          []Profile          `json:"profiles"`
	Artifacts         []Artifact         `json:"artifacts"`
	Locations         []ArtifactLocation `json:"locations"`
	Media             []Media            `json:"media"`
	Assets            []Asset            `json:"assets"`
	MediaAssets       []MediaAsset       `json:"media_assets"`
	Metadata          []MetadataSnapshot `json:"metadata"`
	Observations      []Observation      `json:"observations"`
	Dates             []DateObservation  `json:"dates"`
	DateSelections    []DateSelection    `json:"date_selections"`
	Speakers          []Speaker          `json:"speakers"`
	Segments          []Segment          `json:"segments"`
	Datasets          []Dataset          `json:"datasets"`
	Members           []DatasetMember    `json:"members"`
	Runs              []TrainingRun      `json:"runs"`
	Models            []Model            `json:"models"`
	Versions          []ModelVersion     `json:"versions"`
	ModelArtifacts    []ModelArtifact    `json:"model_artifacts"`
	ModelAssociations []ModelAssociation `json:"model_associations"`
}
type Setting struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}
type Projection struct {
	Target   string          `json:"target"`
	Document json.RawMessage `json:"document"`
}
type Mutation struct {
	OperationID string       `json:"operation_id"`
	Expected    int64        `json:"expected"`
	Settings    []Setting    `json:"settings"`
	Records     Records      `json:"records"`
	Projections []Projection `json:"projections"`
}
type Receipt struct {
	OperationID string          `json:"operation_id"`
	Revision    int64           `json:"revision"`
	Result      json.RawMessage `json:"result"`
}

func strict(data []byte, v any) error {
	if ValidateJSON(data) != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	d.UseNumber()
	if e := d.Decode(v); e != nil {
		return contracts.Fail("invalid_request")
	}
	if d.Decode(new(any)) != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}

// ValidateJSON rejects ambiguous keys and damaged UTF-8 before lossless decoding.
func ValidateJSON(data []byte) error {
	if !utf8.Valid(data) {
		return contracts.Fail("invalid_request")
	}
	return validateJSONDecoder(json.NewDecoder(bytes.NewReader(data)))
}

func validateJSONDecoder(d *json.Decoder) error {
	d.UseNumber()
	var value func() error
	value = func() error {
		token, e := d.Token()
		if e != nil {
			return e
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, e := d.Token()
				if e != nil {
					return e
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return contracts.Fail("invalid_request")
				}
				seen[name] = true
				if e = value(); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = value(); e != nil {
					return e
				}
			}
		default:
			return contracts.Fail("invalid_request")
		}
		_, e = d.Token()
		return e
	}
	if value() != nil {
		return contracts.Fail("invalid_request")
	}
	if _, e := d.Token(); e != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func canonical(data []byte) ([]byte, error) {
	var v any
	if e := strict(data, &v); e != nil {
		return nil, e
	}
	return json.Marshal(v)
}
func positive(n int64) bool { return n > 0 }

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validJSON(data json.RawMessage) bool { _, e := canonical(data); return e == nil }
func nonsecret(data json.RawMessage) bool {
	var v any
	if strict(data, &v) != nil {
		return false
	}
	var visit func(any) bool
	visit = func(x any) bool {
		switch y := x.(type) {
		case map[string]any:
			for k, v := range y {
				if credentialKey(k) {
					return false
				}
				if !visit(v) {
					return false
				}
			}
		case []any:
			for _, v := range y {
				if !visit(v) {
					return false
				}
			}
		}
		return true
	}
	return visit(v)
}

// Normalize case and separators without treating ordinary tokenizer/token-limit
// options or explicit credential references as resolved credential material.
func credentialKey(key string) bool {
	key = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, key)
	switch key {
	case "secret", "token", "authorization", "proxyauthorization", "pwd", "secretkey", "accesskey", "accesskeyid", "awsaccesskeyid", "clientsecret", "secretvalue", "secrettoken":
		return true
	}
	for _, suffix := range []string{"password", "passwd", "passphrase", "apikey", "apitoken", "accesstoken", "refreshtoken", "idtoken", "authtoken", "bearertoken", "clientsecret", "privatekey", "secretaccesskey"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}
