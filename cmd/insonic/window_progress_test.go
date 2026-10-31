// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestWindowWorkSummaryShowsBoundedProgress(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"work", "show"}}
	for _, state := range []string{"running", "succeeded"} {
		var out, stderr bytes.Buffer
		result := map[string]any{"id": contracts.ID(), "state": state, "kind": "recordings.process", "result": map[string]any{"completed_windows": json.Number("3"), "total_windows": json.Number("8"), "private_transcript": "Do not render this."}}
		if outputTo(result, nil, false, contracts.Response{}, &out, &stderr) != 0 || !strings.Contains(out.String(), "Windows complete: 3 of 8") || strings.Contains(out.String(), "Do not render") {
			t.Fatal(out.String(), stderr.String())
		}
	}
	for _, v := range []any{nil, -1, 1.5, 16385, "4"} {
		if _, ok := windowProgressCount(v); ok {
			t.Fatalf("invalid progress accepted: %v", v)
		}
	}
}
