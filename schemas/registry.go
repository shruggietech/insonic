// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
	"strings"
	"sync"
)

//go:embed v*/*.json
var registry embed.FS
var once sync.Once
var compiled *jsonschema.Schema
var requestSchema *jsonschema.Schema
var responseSchema *jsonschema.Schema
var speakerModelSchema *jsonschema.Schema
var speakerDatasetSchema *jsonschema.Schema
var processingToolsSchema *jsonschema.Schema
var pipelineConfigurationSchema *jsonschema.Schema
var mediaToolsSchema *jsonschema.Schema
var graphQuerySchema *jsonschema.Schema
var compileErr error
var historicalMu sync.Mutex
var historicalSchemas = map[string]*jsonschema.Schema{}

// validateDocument selects the declared, packaged immutable release contract.
// Transport envelopes deliberately continue to use the current-only validators.
func validateDocument(current *jsonschema.Schema, filename string, data []byte) error {
	var header struct {
		Version string `json:"schema_version"`
	}
	if json.Unmarshal(data, &header) != nil {
		return errors.New("invalid document")
	}
	if header.Version == "" || header.Version == contracts.Version {
		return validate(current, data)
	}
	if !contracts.SupportedDocumentVersion(header.Version) {
		return errors.New("unsupported document version")
	}
	key := header.Version + "/" + filename
	historicalMu.Lock()
	defer historicalMu.Unlock()
	schema := historicalSchemas[key]
	if schema == nil {
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiler.UseLoader(offlineLoader{})
		upstream, err := jsonschema.UnmarshalJSON(bytes.NewReader(subtitles.SchemaBytes()))
		if err != nil {
			return err
		}
		if err := compiler.AddResource(subtitles.SchemaID, upstream); err != nil {
			return err
		}
		files, err := registry.ReadDir("v" + header.Version)
		if err != nil {
			return err
		}
		for _, file := range files {
			raw, err := registry.ReadFile("v" + header.Version + "/" + file.Name())
			if err != nil {
				return err
			}
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
			if err != nil {
				return err
			}
			if err := compiler.AddResource(doc.(map[string]any)["$id"].(string), doc); err != nil {
				return err
			}
		}
		schema, err = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + header.Version + "/schemas/v" + header.Version + "/" + filename)
		if err != nil {
			return err
		}
		historicalSchemas[key] = schema
	}
	return validate(schema, data)
}

type offlineLoader struct{}

func (offlineLoader) Load(string) (any, error) {
	return nil, errors.New("schema resource is not packaged")
}

func initialize() {
	once.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiler.UseLoader(offlineLoader{})
		upstream, err := jsonschema.UnmarshalJSON(bytes.NewReader(subtitles.SchemaBytes()))
		if err != nil {
			compileErr = err
			return
		}
		if err = compiler.AddResource(subtitles.SchemaID, upstream); err != nil {
			compileErr = err
			return
		}
		files, err := registry.ReadDir("v" + contracts.Version)
		if err != nil {
			compileErr = err
			return
		}
		for _, file := range files {
			data, err := registry.ReadFile("v" + contracts.Version + "/" + file.Name())
			if err != nil {
				compileErr = err
				return
			}
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
			if err != nil {
				compileErr = err
				return
			}
			id := doc.(map[string]any)["$id"].(string)
			if err := compiler.AddResource(id, doc); err != nil {
				compileErr = err
				return
			}
		}
		compiled, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/workspace-config.schema.json")
		if compileErr == nil {
			requestSchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/runtime-request.schema.json")
		}
		if compileErr == nil {
			responseSchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/runtime-response.schema.json")
		}
		if compileErr == nil {
			processingToolsSchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/processing-tools.schema.json")
		}
		if compileErr == nil {
			pipelineConfigurationSchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/pipeline-config.schema.json#/$defs/configuration")
		}
		if compileErr == nil {
			graphQuerySchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/graph-query.schema.json")
		}
		if compileErr == nil {
			mediaToolsSchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/media-tools.schema.json")
		}
		if compileErr == nil {
			speakerModelSchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/speaker-model.schema.json")
		}
		if compileErr == nil {
			speakerDatasetSchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v" + contracts.Version + "/schemas/v" + contracts.Version + "/speaker-dataset.schema.json")
		}
	})
}

// ValidateSpeakerDocument checks the portable speaker contracts packaged in the release master.
func ValidateSpeakerDocument(data []byte) error {
	initialize()
	var envelope struct {
		Kind string `json:"kind"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return errors.New("invalid speaker document")
	}
	switch envelope.Kind {
	case "speaker-model":
		return validateDocument(speakerModelSchema, "speaker-model.schema.json", data)
	case "speaker-dataset":
		return validateDocument(speakerDatasetSchema, "speaker-dataset.schema.json", data)
	default:
		return errors.New("unsupported speaker document")
	}
}

func ValidateGraphQuery(data []byte) error { initialize(); return validate(graphQuerySchema, data) }
func ValidateWorkspace(data []byte) error {
	initialize()
	return validateDocument(compiled, "workspace-config.schema.json", data)
}

func ValidateRequest(data []byte) error {
	initialize()
	return validate(requestSchema, data)
}

// ValidateResponse checks the public shared-runtime result envelope offline.
// Operation-specific summaries retain their distinct acquisition/missing states.
func ValidateResponse(data []byte) error {
	initialize()
	return validate(responseSchema, data)
}

func ValidateMediaTools(data []byte) error {
	initialize()
	return validateDocument(mediaToolsSchema, "media-tools.schema.json", data)
}

// ValidateDocument validates processing configuration against its master-registered
// release contract without compiling unrelated contracts or using network access.
func ValidateDocument(data []byte) error {
	initialize()
	return validateDocument(processingToolsSchema, "processing-tools.schema.json", data)
}

// ValidatePipelineConfiguration validates a saved bare worker definition offline.
// Typed pipeline validation additionally checks routing and cross-field semantics.
func ValidatePipelineConfiguration(data []byte) error {
	initialize()
	return validate(pipelineConfigurationSchema, data)
}

func validate(schema *jsonschema.Schema, data []byte) error {
	if compileErr != nil {
		return compileErr
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return schema.Validate(doc)
}

// QuerySchemaBytes expands the packaged bare QueryInput using local definitions.
// Providers receive no dangling references and no saved-query envelope fields.
func QuerySchemaBytes() []byte {
	var request, common map[string]any
	load := func(path string, target any) {
		raw, _ := registry.ReadFile(path)
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		_ = decoder.Decode(target)
	}
	load("v"+contracts.Version+"/runtime-request.schema.json", &request)
	load("v"+contracts.Version+"/common.schema.json", &common)
	var expand func(any) any
	expand = func(value any) any {
		switch node := value.(type) {
		case map[string]any:
			out := map[string]any{}
			if ref, ok := node["$ref"].(string); ok {
				doc := request
				if strings.HasPrefix(ref, "common.schema.json") {
					doc = common
				}
				var target any = doc
				for _, part := range strings.Split(strings.SplitN(ref, "#", 2)[1][1:], "/") {
					target = target.(map[string]any)[part]
				}
				out = expand(target).(map[string]any)
			}
			for key, child := range node {
				if key != "$ref" {
					out[key] = expand(child)
				}
			}
			return out
		case []any:
			out := make([]any, len(node))
			for i, child := range node {
				out[i] = expand(child)
			}
			return out
		default:
			return value
		}
	}
	raw, _ := json.Marshal(expand(request["$defs"].(map[string]any)["exploreQuery"]))
	return raw
}
