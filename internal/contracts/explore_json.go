// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"encoding/json"
)

func (d QueryDefinition) MarshalJSON() ([]byte, error) {
	type alias QueryDefinition
	raw, e := json.Marshal(alias(d))
	if e != nil {
		return nil, e
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if string(fields["filters"]) == "{}" {
		delete(fields, "filters")
	}
	return json.Marshal(fields)
}
