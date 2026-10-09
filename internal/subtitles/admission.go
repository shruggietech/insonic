// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"sync"
	"unicode/utf8"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shruggietech/insonic/internal/contracts"
)

//go:embed schema/v1.0.0/cueson.schema.json
var historical100 []byte

//go:embed schema/v1.1.0/cueson.schema.json
var historical110 []byte
var historyOnce sync.Once
var historySchemas map[string]*jsonschema.Schema
var historyErr error

func compileHistory() {
	historyOnce.Do(func() {
		historySchemas = map[string]*jsonschema.Schema{}
		for version, resource := range map[string]struct {
			data   []byte
			digest string
		}{
			"1.0.0": {historical100, "1aad14567033d7e14d9beb78985e18007aefb5345095370b11b6b887df7ec541"},
			"1.1.0": {historical110, "223b61cbcf6337167039268576b2c739564585fff10e6cc6076e47a03526a0f7"},
		} {
			sum := sha256.Sum256(resource.data)
			if hex.EncodeToString(sum[:]) != resource.digest {
				historyErr = contracts.Fail("unavailable")
				return
			}
			compiler := jsonschema.NewCompiler()
			compiler.AssertFormat()
			compiler.UseLoader(offlineLoader{})
			value, err := jsonschema.UnmarshalJSON(bytes.NewReader(resource.data))
			if err != nil {
				historyErr = err
				return
			}
			id := "https://cueson.io/schema/v" + version + "/cueson.schema.json"
			if err = compiler.AddResource(id, value); err != nil {
				historyErr = err
				return
			}
			historySchemas[version], err = compiler.Compile(id)
			if err != nil {
				historyErr = err
				return
			}
		}
	})
}

type Admission struct {
	Document         json.RawMessage `json:"document"`
	OriginalVersion  string          `json:"original_version"`
	Translated       bool            `json:"translated"`
	AttributionBasis []Observation   `json:"attribution_basis"`
}
type Observation struct {
	SpeakerID string `json:"speaker_id"`
	Label     string `json:"label"`
	Origin    string `json:"origin"`
}

// Preserve validates the exact source schema before an explicit historical
// identity translation. RawMessage retains integer precision and native data.
func Preserve(data []byte) (Admission, error) {
	result := Admission{AttributionBasis: []Observation{}}
	invalid := contracts.Fail("invalid_request")
	if len(data) == 0 || len(data) > MaxDocumentBytes || !utf8.Valid(data) || !validJSONUnicode(data) {
		return result, invalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if uniqueJSON(decoder, 0) != nil {
		return result, invalid
	}
	if _, err := decoder.Token(); err != io.EOF {
		return result, invalid
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return result, invalid
	}
	var version, id string
	if json.Unmarshal(fields["schema_version"], &version) != nil || json.Unmarshal(fields["$schema"], &id) != nil {
		return result, invalid
	}
	result.OriginalVersion = version
	if version == SchemaVersion {
		if err := ValidateDocument(data); err != nil {
			return result, err
		}
		result.Document = bytes.Clone(data)
		return result, nil
	}
	if version != "1.0.0" && version != "1.1.0" || id != "https://cueson.io/schema/v"+version+"/cueson.schema.json" {
		return result, contracts.Fail("incompatible_version")
	}
	compileHistory()
	if historyErr != nil {
		return result, contracts.Fail("unavailable")
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil || historySchemas[version].Validate(value) != nil {
		return result, invalid
	}
	fields["$schema"], _ = json.Marshal(SchemaID)
	fields["schema_version"], _ = json.Marshal(SchemaVersion)
	translated, err := json.Marshal(fields)
	if err != nil {
		return result, invalid
	}
	// Existing fields have unchanged current semantics; validate hashes, exact
	// instants, ordinals and summary/timing relationships after identity upgrade.
	if err = ValidateDocument(translated); err != nil {
		return result, err
	}
	result.Document = translated
	result.Translated = true
	return result, nil
}

// Attribute derives untimed participation only from Cueson's structured
// observations. Any existing assignment makes the whole document authoritative.
func Attribute(input Admission, recording, mode string) (Admission, error) {
	if !contracts.ValidID(recording) || mode != "auto" && mode != "native" && mode != "off" {
		return input, contracts.Fail("invalid_request")
	}
	if err := ValidateDocument(input.Document); err != nil {
		return input, err
	}
	if mode == "off" {
		return input, nil
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(input.Document, &fields)
	var cues []map[string]json.RawMessage
	json.Unmarshal(fields["cues"], &cues)
	for _, cue := range cues {
		var assignments []attribution
		json.Unmarshal(cue["speaker_attributions"], &assignments)
		if len(assignments) > 0 {
			return input, nil
		}
	}
	observations := map[string]bool{}
	changed := false
	count := 0
	for _, cue := range cues {
		var labels []struct {
			Name   string `json:"name"`
			Origin string `json:"origin"`
		}
		json.Unmarshal(cue["speakers"], &labels)
		seen := map[string]bool{}
		assignments := []attribution{}
		for _, label := range labels {
			if label.Name == "" || label.Origin != "native" && label.Origin != "heuristic" || mode == "native" && label.Origin != "native" {
				continue
			}
			sum := sha256.Sum256([]byte(recording + "\x00" + label.Origin + "\x00" + label.Name))
			id := "native-" + hex.EncodeToString(sum[:])
			if !seen[id] {
				assignments = append(assignments, attribution{SpeakerID: id})
				seen[id] = true
				count++
			}
			if !observations[id] {
				input.AttributionBasis = append(input.AttributionBasis, Observation{id, label.Name, label.Origin})
				observations[id] = true
			}
		}
		if len(assignments) > 1024 || count > MaxAssignments {
			return input, contracts.Fail("output_limit")
		}
		if len(assignments) > 0 {
			cue["speaker_attributions"], _ = json.Marshal(assignments)
			changed = true
		}
	}
	if changed {
		fields["cues"], _ = json.Marshal(cues)
		input.Document, _ = json.Marshal(fields)
		if err := ValidateDocument(input.Document); err != nil {
			return input, err
		}
	}
	return input, nil
}
