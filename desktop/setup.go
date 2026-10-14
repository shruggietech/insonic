// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/firstsetup"
	"time"
)

// SetupDraft shares optional pre-library location and nonsecret persistence with CLI.
func (b *Bridge) SetupDraft(action, input, id string) contracts.Response {
	ctx, finish, e := b.beginOperation(id, 10*time.Second)
	if e != nil {
		return b.failed(e)
	}
	defer finish()
	response := contracts.Response{Kind: "runtime-response", Version: contracts.Version, RequestID: id}
	var value any
	switch action {
	case "location":
		var in struct {
			Path string `json:"path,omitempty"`
		}
		if len(input) > 16<<10 {
			e = contracts.Fail("input_limit")
			break
		}
		if input == "" {
			input = "{}"
		}
		d := json.NewDecoder(bytes.NewBufferString(input))
		d.DisallowUnknownFields()
		if catalog.ValidateJSON([]byte(input)) != nil || d.Decode(&in) != nil {
			e = contracts.Fail("invalid_request")
			break
		}
		value, e = firstsetup.Preview(in.Path)
	case "show", "save":
		var s firstsetup.Store
		s, e = firstsetup.ProfileStore()
		if e != nil {
			break
		}
		if action == "show" {
			if input != "" && input != "{}" {
				e = contracts.Fail("invalid_request")
				break
			}
			value, e = s.Show()
		} else {
			value, e = s.Save(ctx, id, []byte(input))
		}
	default:
		e = contracts.Fail("invalid_request")
	}
	response.Result = value
	if e != nil {
		response.Error = contracts.SafeError(e)
	}
	return response
}
