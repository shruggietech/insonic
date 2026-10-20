// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"errors"
	"math/big"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/shruggietech/insonic/internal/contracts"
)

// Keep the structured source available to existing operation-specific field
// projections without returning its potentially private text to callers.
type inputValidationError struct {
	safe   *contracts.Error
	source *jsonschema.ValidationError
}

func (e *inputValidationError) Error() string   { return e.safe.Error() }
func (e *inputValidationError) Unwrap() []error { return []error{e.safe, e.source} }

// inputError projects structured validator metadata. Neither the validator's
// text nor its submitted Got/unknown-property fields become output.
func inputError(err error) error {
	if err == nil {
		return nil
	}
	var original *contracts.Error
	if errors.As(err, &original) {
		return original
	}
	var root *jsonschema.ValidationError
	if !errors.As(err, &root) {
		return contracts.Condition("invalid_request", "input_invalid")
	}
	var selected *contracts.Error
	best := -1
	var visit func(*jsonschema.ValidationError)
	visit = func(e *jsonschema.ValidationError) {
		causes := e.Causes
		switch e.ErrorKind.(type) {
		case *kind.AnyOf, *kind.OneOf:
			var matching []*jsonschema.ValidationError
			for _, cause := range causes {
				if !discriminatorMismatch(cause) {
					matching = append(matching, cause)
				}
			}
			if len(matching) > 0 {
				causes = matching
			}
		}
		for _, cause := range causes {
			visit(cause)
		}
		detail := contracts.InputViolation{Pointer: contracts.InputPointer(e.InstanceLocation)}
		reason := ""
		bound := func(v *big.Rat) int64 {
			if v != nil && v.IsInt() && v.Num().IsInt64() {
				return v.Num().Int64()
			}
			return 0
		}
		switch k := e.ErrorKind.(type) {
		case *kind.Type:
			reason = "input_type"
			if len(k.Want) == 1 {
				detail.Expected = k.Want[0]
			}
			if len(e.InstanceLocation) == 0 || detail.Pointer == "/data" {
				reason = "input_shape"
			}
		case *kind.Enum, *kind.Const:
			reason = "input_enum"
		case *kind.Required:
			reason = "input_required"
			if len(k.Missing) > 0 {
				detail.Pointer = contracts.InputPointer(append(append([]string{}, e.InstanceLocation...), k.Missing[0]))
			}
		case *kind.AdditionalProperties:
			reason = "input_unknown_field"
		case *kind.Minimum:
			reason = "input_range"
			detail.Constraint = "minimum"
			detail.Bound = bound(k.Want)
		case *kind.Maximum:
			reason = "input_range"
			detail.Constraint = "maximum"
			detail.Bound = bound(k.Want)
		case *kind.MinLength:
			reason = "input_length"
			detail.Constraint = "minLength"
			detail.Bound = int64(k.Want)
		case *kind.MaxLength:
			reason = "input_length"
			detail.Constraint = "maxLength"
			detail.Bound = int64(k.Want)
		case *kind.MinItems:
			reason = "input_length"
			detail.Constraint = "minItems"
			detail.Bound = int64(k.Want)
		case *kind.MaxItems:
			reason = "input_length"
			detail.Constraint = "maxItems"
			detail.Bound = int64(k.Want)
		case *kind.Format, *kind.Pattern:
			reason = "input_constraint"
		}
		if reason == "" {
			return
		}
		if strings.HasSuffix(detail.Pointer, "/schema_version") {
			reason = "input_version"
		}
		score := len(e.InstanceLocation) * 10
		if detail.Constraint != "" {
			score += 3
		}
		if reason == "input_version" {
			score += 1000
		}
		if score > best {
			best = score
			selected = contracts.InputFailure(reason, detail)
		}
	}
	visit(root)
	if selected == nil {
		selected = contracts.Condition("invalid_request", "input_invalid")
	}
	return &inputValidationError{safe: selected, source: root}
}

// A union branch that contradicts an explicit discriminator cannot describe the
// submitted mode. Nested unions select their own branches when visited.
func discriminatorMismatch(e *jsonschema.ValidationError) bool {
	switch k := e.ErrorKind.(type) {
	case *kind.AnyOf, *kind.OneOf:
		return false
	case *kind.Const:
		if len(e.InstanceLocation) > 0 {
			switch e.InstanceLocation[len(e.InstanceLocation)-1] {
			case "mode", "kind", "operation", "section", "type", "adapter_id":
				got, gotOK := k.Got.(string)
				want, wantOK := k.Want.(string)
				return gotOK && wantOK && got != want
			}
		}
	}
	for _, cause := range e.Causes {
		if discriminatorMismatch(cause) {
			return true
		}
	}
	return false
}
