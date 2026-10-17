// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"
)

// ValidateJSON rejects ambiguous keys and damaged UTF-8 before lossless decoding.
func ValidateJSON(data []byte) error {
	if !utf8.Valid(data) {
		return Fail("invalid_request")
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
					return Fail("invalid_request")
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
	if value() != nil {
		return Fail("invalid_request")
	}
	if _, e := d.Token(); e != io.EOF {
		return Fail("invalid_request")
	}
	return nil
}
