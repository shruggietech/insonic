// SPDX-License-Identifier: Apache-2.0
package main

import "github.com/shruggietech/insonic/internal/contracts"

func checkedInput(raw []byte, limit int, readErr error) error {
	if readErr != nil {
		return contracts.Condition("unavailable", "input_missing")
	}
	if len(raw) > limit {
		return contracts.InputFailure("input_bytes", contracts.InputViolation{Constraint: "maxBytes", Bound: int64(limit)})
	}
	return contracts.ValidateJSON(raw)
}
