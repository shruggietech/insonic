// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shruggietech/insonic/internal/contracts"
)

const SchemaID = "https://cueson.io/schema/v1.2.0/cueson.schema.json"
const SchemaVersion = "1.2.0"
const SchemaSHA256 = "f2661a3d52effbab4a82a4d47197b5c7fae58496dc30a397ea3f2f668358b654"
const MaxDocumentBytes = 16 << 20

//go:embed schema/cueson.schema.json
var schemaBytes []byte

func SchemaBytes() []byte { return bytes.Clone(schemaBytes) }

var compileOnce sync.Once
var compiledSchema *jsonschema.Schema
var compileErr error

type offlineLoader struct{}

func (offlineLoader) Load(string) (any, error) { return nil, errors.New("unpackaged subtitle schema") }
func compileSchema() {
	compileOnce.Do(func() {
		hash := sha256.Sum256(schemaBytes)
		if hex.EncodeToString(hash[:]) != SchemaSHA256 {
			compileErr = errors.New("subtitle schema checksum mismatch")
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiler.UseLoader(offlineLoader{})
		resource, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
		if err != nil {
			compileErr = err
			return
		}
		if err = compiler.AddResource(SchemaID, resource); err != nil {
			compileErr = err
			return
		}
		compiledSchema, compileErr = compiler.Compile(SchemaID)
	})
}

type attribution struct {
	SpeakerID string `json:"speaker_id"`
	Start     *int64 `json:"start_milliseconds,omitempty"`
	End       *int64 `json:"end_milliseconds,omitempty"`
}
type cueTiming struct {
	Start    int64 `json:"start_milliseconds,omitempty"`
	End      int64 `json:"end_milliseconds,omitempty"`
	Duration int64 `json:"duration_milliseconds"`
}
type semanticCue struct {
	ID           string            `json:"id"`
	Ordinal      int               `json:"ordinal"`
	Timing       cueTiming         `json:"timing"`
	Attributions []attribution     `json:"speaker_attributions"`
	Tokens       []json.RawMessage `json:"tokens"`
}
type semanticTiming struct {
	Duration int64  `json:"duration_milliseconds"`
	Start    *int64 `json:"timeline_start_milliseconds,omitempty"`
}
type semanticTimestamp struct {
	ISO    string `json:"iso"`
	UnixNS int64  `json:"unix_ns"`
}
type semanticAsset struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	Data     string `json:"data_base64"`
	Size     struct {
		Bytes int64 `json:"bytes"`
	} `json:"size"`
	Hashes struct {
		SHA256 string `json:"sha256"`
	} `json:"hashes"`
	Timestamps struct {
		Created       *semanticTimestamp `json:"created"`
		Modified      *semanticTimestamp `json:"modified"`
		Accessed      *semanticTimestamp `json:"accessed"`
		CreatedSource string             `json:"created_source"`
	} `json:"timestamps"`
}
type semanticDocument struct {
	Format  string          `json:"format"`
	Schema  string          `json:"$schema"`
	Version string          `json:"schema_version"`
	Cues    []semanticCue   `json:"cues"`
	Media   *semanticTiming `json:"media_timing"`
	Source  struct {
		Primary string          `json:"primary_asset_id"`
		Assets  []semanticAsset `json:"assets"`
	} `json:"source"`
	Summary struct {
		Count int    `json:"cue_count"`
		Start *int64 `json:"media_start_milliseconds"`
		End   *int64 `json:"media_end_milliseconds"`
		Span  *int64 `json:"media_span_milliseconds"`
		Words bool   `json:"has_word_level_timing"`
	} `json:"document"`
	Stats struct {
		Count       int    `json:"cue_count"`
		Span        *int64 `json:"media_span_milliseconds"`
		Words       bool   `json:"has_word_level_timing"`
		Diagnostics int    `json:"diagnostic_count"`
		Warnings    int    `json:"warning_count"`
		Errors      int    `json:"error_count"`
	} `json:"stats"`
	Diagnostics []struct {
		Severity string  `json:"severity"`
		CueID    *string `json:"cue_id"`
	} `json:"diagnostics"`
}

// ValidateDocument checks the packaged current schema and source/consumer
// invariants without executing a tool, loading a model, or retrieving a URI.
// Local UUID scope is insonic's contract; the upstream permits broader IDs.
func ValidateDocument(data []byte) error { return validateDocument(data, false) }
func validateDocument(data []byte, allowEmpty bool) error {
	invalid := func() error { return contracts.Fail("invalid_request") }
	if len(data) == 0 || len(data) > MaxDocumentBytes || !utf8.Valid(data) || !validJSONUnicode(data) {
		return invalid()
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := uniqueJSON(decoder, 0); err != nil {
		return invalid()
	}
	if _, err := decoder.Token(); err != io.EOF {
		return invalid()
	}
	compileSchema()
	if compileErr != nil {
		return contracts.Fail("unavailable")
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return invalid()
	}
	if err = compiledSchema.Validate(instance); err != nil {
		return invalid()
	}
	var doc semanticDocument
	if err = json.Unmarshal(data, &doc); err != nil {
		return invalid()
	}
	if doc.Schema != SchemaID || doc.Version != SchemaVersion || len(doc.Cues) == 0 && !allowEmpty || len(doc.Cues) > 65536 || len(doc.Diagnostics) > 8192 {
		return invalid()
	}
	mediaStart, mediaEnd := int64(0), int64(0)
	if doc.Media != nil {
		if doc.Media.Start != nil {
			mediaStart = *doc.Media.Start
		}
		if doc.Media.Duration < 0 || mediaStart > math.MaxInt64-doc.Media.Duration {
			return invalid()
		}
		mediaEnd = mediaStart + doc.Media.Duration
	}
	ids := map[string]bool{}
	minimum, maximum := int64(0), int64(0)
	if len(doc.Cues) > 0 {
		minimum, maximum = doc.Cues[0].Timing.Start, doc.Cues[0].Timing.End
	}
	words := false
	for i, cue := range doc.Cues {
		if ids[cue.ID] || cue.Ordinal != i || cue.Timing.Start < 0 || cue.Timing.End < cue.Timing.Start || doc.Format == "webvtt" && cue.Timing.End == cue.Timing.Start || cue.Timing.Duration != cue.Timing.End-cue.Timing.Start || len(cue.Attributions) > 1024 {
			return invalid()
		}
		ids[cue.ID] = true
		if cue.Timing.Start < minimum {
			minimum = cue.Timing.Start
		}
		if cue.Timing.End > maximum {
			maximum = cue.Timing.End
		}
		words = words || len(cue.Tokens) > 0
		for _, assignment := range cue.Attributions {
			if !contracts.ValidID(assignment.SpeakerID) || (assignment.Start == nil) != (assignment.End == nil) {
				return invalid()
			}
			if assignment.Start == nil {
				continue
			}
			start, end := *assignment.Start, *assignment.End
			if start < 0 || end <= start || start < cue.Timing.Start || end > cue.Timing.End {
				return invalid()
			}
			if doc.Media != nil && (start < mediaStart || end > mediaEnd) {
				return invalid()
			}
		}
	}
	span := maximum - minimum
	if len(doc.Cues) == 0 {
		if doc.Summary.Count != 0 || doc.Stats.Count != 0 || doc.Summary.Start != nil || doc.Summary.End != nil || doc.Summary.Span != nil || doc.Stats.Span != nil || doc.Summary.Words || doc.Stats.Words {
			return invalid()
		}
	} else if doc.Summary.Count != len(doc.Cues) || doc.Stats.Count != len(doc.Cues) || doc.Summary.Start == nil || *doc.Summary.Start != minimum || doc.Summary.End == nil || *doc.Summary.End != maximum || doc.Summary.Span == nil || *doc.Summary.Span != span || doc.Stats.Span == nil || *doc.Stats.Span != span || doc.Summary.Words != words || doc.Stats.Words != words {
		return invalid()
	}
	warnings, failures := 0, 0
	for _, diagnostic := range doc.Diagnostics {
		if diagnostic.CueID != nil && !ids[*diagnostic.CueID] {
			return invalid()
		}
		if diagnostic.Severity == "warning" {
			warnings++
		}
		if diagnostic.Severity == "error" {
			failures++
		}
	}
	if doc.Stats.Diagnostics != len(doc.Diagnostics) || doc.Stats.Warnings != warnings || doc.Stats.Errors != failures {
		return invalid()
	}
	assets := map[string]bool{}
	for _, asset := range doc.Source.Assets {
		if assets[asset.ID] || asset.FileName == "" || strings.ContainsAny(asset.FileName, "/\\:") || asset.FileName == "." || asset.FileName == ".." {
			return invalid()
		}
		assets[asset.ID] = true
		decoded, err := base64.StdEncoding.Strict().DecodeString(asset.Data)
		if err != nil || base64.StdEncoding.EncodeToString(decoded) != asset.Data || int64(len(decoded)) != asset.Size.Bytes {
			return invalid()
		}
		digest := sha256.Sum256(decoded)
		if hex.EncodeToString(digest[:]) != asset.Hashes.SHA256 {
			return invalid()
		}
		for _, stamp := range []*semanticTimestamp{asset.Timestamps.Created, asset.Timestamps.Modified, asset.Timestamps.Accessed} {
			if stamp == nil {
				continue
			}
			if point := strings.IndexByte(stamp.ISO, '.'); point >= 0 {
				digits := 0
				for position := point + 1; position < len(stamp.ISO) && stamp.ISO[position] >= '0' && stamp.ISO[position] <= '9'; position++ {
					digits++
				}
				if digits > 9 {
					return invalid()
				}
			}
			instant, err := time.Parse(time.RFC3339Nano, stamp.ISO)
			if err != nil || !instant.Equal(time.Unix(0, stamp.UnixNS)) {
				return invalid()
			}
		}
		if (asset.Timestamps.CreatedSource == "unavailable") != (asset.Timestamps.Created == nil) {
			return invalid()
		}
	}
	if !assets[doc.Source.Primary] {
		return invalid()
	}
	return nil
}
func uniqueJSON(decoder *json.Decoder, depth int) error {
	if depth > 128 {
		return errors.New("JSON depth")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return errors.New("duplicate JSON member")
			}
			seen[name] = true
			if err = uniqueJSON(decoder, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("JSON object framing")
		}
	case '[':
		for decoder.More() {
			if err := uniqueJSON(decoder, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("JSON array framing")
		}
	default:
		return fmt.Errorf("JSON delimiter")
	}
	return nil
}

// encoding/json replaces unpaired surrogate escapes; reject them before any
// decoder can change a source/native observation into a replacement rune.
func validJSONUnicode(data []byte) bool {
	for index := 0; index < len(data); index++ {
		if data[index] != '\\' {
			continue
		}
		if index+1 >= len(data) {
			return false
		}
		if data[index+1] != 'u' {
			index++
			continue
		}
		if index+5 >= len(data) {
			return false
		}
		value, err := strconv.ParseUint(string(data[index+2:index+6]), 16, 16)
		if err != nil {
			return false
		}
		if value >= 0xdc00 && value <= 0xdfff {
			return false
		}
		if value >= 0xd800 && value <= 0xdbff {
			if index+11 >= len(data) || data[index+6] != '\\' || data[index+7] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(data[index+8:index+12]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			index += 11
		} else {
			index += 5
		}
	}
	return true
}
