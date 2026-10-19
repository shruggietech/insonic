// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

// ValidateJSON rejects ambiguous keys and damaged UTF-8 before lossless decoding.
func ValidateJSON(data []byte) error {
	if !utf8.Valid(data) {
		return InputFailure("json_encoding", InputViolation{})
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return InputFailure("json_empty", InputViolation{})
	}
	return ValidateJSONDecoder(json.NewDecoder(bytes.NewReader(data)))
}

func ValidateJSONDecoder(d *json.Decoder) error {
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
					return InputFailure("json_duplicate_key", InputViolation{Offset: d.InputOffset()})
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
			return Fail("invalid_request")
		}
		_, e = d.Token()
		return e
	}
	if err := value(); err != nil {
		if typed, ok := err.(*Error); ok {
			return typed
		}
		return InputFailure("json_syntax", InputViolation{Offset: jsonErrorOffset(d, err)})
	}
	if _, e := d.Token(); e != io.EOF {
		if e != nil {
			return InputFailure("json_syntax", InputViolation{Offset: jsonErrorOffset(d, e)})
		}
		return InputFailure("json_extra_document", InputViolation{Offset: d.InputOffset()})
	}
	return nil
}

func jsonErrorOffset(d *json.Decoder, err error) int64 {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return syntax.Offset
	}
	return d.InputOffset() + 1
}
