// SPDX-License-Identifier: Apache-2.0
package contracts

import "testing"

func TestObservedResultPreservesEstablishedPartialAndFalseZero(t *testing.T) {
	failure := Fail("unavailable")
	type receipt struct {
		ID        string
		Count     int
		Available bool
	}
	for _, result := range []any{receipt{ID: "observed"}, map[string]any{"count": 0, "available": false}, false, 0, "", receipt{}} {
		got := ObservedResult(result, failure)
		if value, ok := result.(receipt); ok && value.ID == "" {
			if got != nil {
				t.Fatal("zero receipt retained")
			}
		} else if got == nil {
			t.Fatalf("meaningful result discarded: %#v", result)
		}
	}
	if ObservedResult(receipt{}, nil) == nil {
		t.Fatal("successful zero result discarded")
	}
}
