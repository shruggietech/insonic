// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"bytes"
	"embed"
	"errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"sync"
)

//go:embed v0.0.0/*.json
var registry embed.FS
var once sync.Once
var compiled *jsonschema.Schema
var requestSchema *jsonschema.Schema
var compileErr error

type offlineLoader struct{}

func (offlineLoader) Load(string) (any, error) {
	return nil, errors.New("schema resource is not packaged")
}

func initialize() {
	once.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiler.UseLoader(offlineLoader{})
		files, err := registry.ReadDir("v0.0.0")
		if err != nil {
			compileErr = err
			return
		}
		for _, file := range files {
			data, err := registry.ReadFile("v0.0.0/" + file.Name())
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
		compiled, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v0.0.0/schemas/v0.0.0/workspace-config.schema.json")
		if compileErr == nil {
			requestSchema, compileErr = compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v0.0.0/schemas/v0.0.0/runtime-request.schema.json")
		}
	})
}

func ValidateWorkspace(data []byte) error {
	initialize()
	return validate(compiled, data)
}

func ValidateRequest(data []byte) error {
	initialize()
	return validate(requestSchema, data)
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
