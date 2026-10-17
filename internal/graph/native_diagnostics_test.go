// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestNativeBoundedReasonsSurvivePublicSanitization(t *testing.T) {
	for _, tc := range []struct{ text, reason string }{{"RETURN range(1,501)", "native_function"}, {"RETURN 1; RETURN 2", "native_statement"}, {"MATCH (n) DELETE n", "native_read_only"}, {"RETURN 'private marker", "native_syntax"}} {
		e := ValidateNative("ladybug-cypher", tc.text, nil)
		if e == nil {
			t.Fatal("expected rejection")
		}
		safe := contracts.SafeError(e)
		if safe.Reason != tc.reason || strings.Contains(safe.Message, "private marker") {
			t.Fatal(safe)
		}
	}
}
