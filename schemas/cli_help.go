// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
)

// InputHelp renders packaged contracts without opening a workspace or provider.
// The template is illustrative: UUIDs and revisions must name current records.
func InputHelp(operation string) (json.RawMessage, json.RawMessage) {
	var request map[string]any
	raw, _ := registry.ReadFile("v" + contracts.Version + "/runtime-request.schema.json")
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	_ = decoder.Decode(&request)
	var inputs []any
	for _, value := range request["allOf"].([]any) {
		branch := value.(map[string]any)
		condition, _ := branch["if"].(map[string]any)
		props, _ := condition["properties"].(map[string]any)
		op, _ := props["operation"].(map[string]any)
		matches := op["const"] == operation
		if values, ok := op["enum"].([]any); ok {
			for _, v := range values {
				if v == operation {
					matches = true
				}
			}
		}
		if !matches {
			continue
		}
		then, _ := branch["then"].(map[string]any)
		fields, _ := then["properties"].(map[string]any)
		if data, ok := fields["data"].(map[string]any); ok {
			inputs = append(inputs, data)
		}
	}
	if len(inputs) == 0 {
		return nil, nil
	}
	input := map[string]any{"allOf": inputs}
	expanded := expandHelp(input, request, 0).(map[string]any)
	template, _ := json.MarshalIndent(helpExample(expanded), "", "  ")
	schema, _ := json.MarshalIndent(expanded, "", "  ")
	return template, schema
}

func expandHelp(value any, document map[string]any, depth int) any {
	if depth > 40 {
		return value
	}
	switch n := value.(type) {
	case map[string]any:
		out := map[string]any{}
		if ref, ok := n["$ref"].(string); ok {
			parts := strings.SplitN(ref, "#", 2)
			targetDoc := document
			if parts[0] != "" {
				var raw map[string]any
				lookup := "v" + contracts.Version + "/" + parts[0]
				if index := strings.LastIndex(parts[0], "/schemas/"); index >= 0 {
					lookup = parts[0][index+len("/schemas/"):]
				} else if id, ok := document["$id"].(string); ok {
					if index := strings.LastIndex(id, "/schemas/"); index >= 0 {
						base := id[index+len("/schemas/"):]
						slash := strings.LastIndex(base, "/")
						if slash >= 0 {
							lookup = base[:slash+1] + parts[0]
						}
					}
				}
				data, e := registry.ReadFile(lookup)
				if e != nil {
					return n
				}
				if e == nil {
					d := json.NewDecoder(bytes.NewReader(data))
					d.UseNumber()
					_ = d.Decode(&raw)
					targetDoc = raw
				}
			}
			var target any = targetDoc
			if len(parts) > 1 {
				for _, part := range strings.Split(strings.TrimPrefix(parts[1], "/"), "/") {
					obj, ok := target.(map[string]any)
					if !ok {
						target = nil
						break
					}
					target = obj[part]
				}
			}
			if obj, ok := expandHelp(target, targetDoc, depth+1).(map[string]any); ok {
				for k, v := range obj {
					out[k] = v
				}
			} else {
				out["$ref"] = ref
			}
		}
		for k, v := range n {
			if k != "$ref" {
				out[k] = expandHelp(v, document, depth+1)
			}
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i, v := range n {
			out[i] = expandHelp(v, document, depth+1)
		}
		return out
	default:
		return value
	}
}

func mergeHelp(base, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		switch k {
		case "properties":
			p := map[string]any{}
			if old, ok := out[k].(map[string]any); ok {
				for x, y := range old {
					p[x] = y
				}
			}
			for x, y := range v.(map[string]any) {
				if old, ok := p[x].(map[string]any); ok {
					p[x] = mergeHelp(old, y.(map[string]any))
				} else {
					p[x] = y
				}
			}
			out[k] = p
		case "required":
			r, _ := out[k].([]any)
			out[k] = append(append([]any{}, r...), v.([]any)...)
		default:
			out[k] = v
		}
	}
	return out
}
func helpExample(original map[string]any) any {
	n := mergeHelp(nil, original)
	if branches, ok := n["allOf"].([]any); ok {
		delete(n, "allOf")
		for _, b := range branches {
			branch := b.(map[string]any)
			if _, conditional := branch["if"]; !conditional {
				n = mergeHelp(n, branch)
			}
		}
	}
	if v, ok := n["const"]; ok {
		return v
	}
	if v, ok := n["default"]; ok {
		return v
	}
	if v, ok := n["examples"].([]any); ok && len(v) > 0 {
		return v[0]
	}
	if v, ok := n["enum"].([]any); ok && len(v) > 0 {
		return v[0]
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		if v, ok := n[key].([]any); ok && len(v) > 0 {
			delete(n, key)
			return helpExample(mergeHelp(n, v[0].(map[string]any)))
		}
	}
	kind := n["type"]
	if types, ok := kind.([]any); ok && len(types) > 0 {
		kind = types[0]
	}
	switch kind {
	case "null":
		return nil
	case "integer", "number":
		if v, ok := n["minimum"]; ok {
			return v
		}
		return 1
	case "boolean":
		return false
	case "array":
		out := []any{}
		if v, ok := n["minItems"].(json.Number); ok && v != "0" {
			if item, ok := n["items"].(map[string]any); ok {
				out = append(out, helpExample(item))
			}
		}
		return out
	case "string":
		if n["format"] == "uuid" {
			return "11111111-1111-4111-8111-111111111111"
		}
		if n["format"] == "uri" {
			return "https://example.org/catalog.json"
		}
		if pattern, ok := n["pattern"].(string); ok {
			if strings.Contains(pattern, "64}") {
				return strings.Repeat("a", 64)
			}
			if strings.Contains(pattern, "8}") {
				return "11111111-1111-4111-8111-111111111111"
			}
		}
		return "example"
	}
	out := map[string]any{}
	properties, _ := n["properties"].(map[string]any)
	add := func(fields []any, props map[string]any) {
		for _, field := range fields {
			key := field.(string)
			if schema, ok := props[key].(map[string]any); ok {
				out[key] = helpExample(schema)
			}
		}
	}
	if fields, ok := n["required"].([]any); ok {
		add(fields, properties)
	}
	// Apply required fields and property overrides in the selected conditional section.
	if branches, ok := original["allOf"].([]any); ok {
		for _, b := range branches {
			branch := b.(map[string]any)
			condition, conditional := branch["if"].(map[string]any)
			if !conditional {
				continue
			}
			fields, _ := condition["properties"].(map[string]any)
			match := true
			for key, v := range fields {
				rule := v.(map[string]any)
				if c, ok := rule["const"]; ok {
					if actual, exists := out[key]; exists && actual != c {
						match = false
					}
				}
			}
			if match {
				if then, ok := branch["then"].(map[string]any); ok {
					merged := mergeHelp(n, then)
					props, _ := merged["properties"].(map[string]any)
					required, _ := merged["required"].([]any)
					add(required, props)
				}
			}
		}
	}
	return out
}
