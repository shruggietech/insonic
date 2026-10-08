// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
)

func decodeChange(raw []byte, out *Change) error {
	if len(raw) > contracts.MaxGraphSnapshot {
		return contracts.Fail("input_limit")
	}
	if catalog.ValidateJSON(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}
