// SPDX-License-Identifier: Apache-2.0
package contracts

import "unicode/utf8"

// ValidTextLength applies JSON Schema string bounds in Unicode code points.
// It preserves the original bytes and does not normalize combining sequences.
// Document, transport and protected-value byte budgets remain independent.
func ValidTextLength(value string, min, max int) bool {
	if !utf8.ValidString(value) {
		return false
	}
	count := utf8.RuneCountInString(value)
	return count >= min && (max < 0 || count <= max)
}

// MaxJobDurationMS is the public synthetic-job interval, independent of leases.
const MaxJobDurationMS = 600000
