// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/base64"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestHumanOutputIsReadableAndExact(t *testing.T) {
	var out, err bytes.Buffer
	code := outputTo(map[string]any{"items": []any{map[string]any{"id": "model-a", "revision": int64(9007199254740993), "state": "registered"}}}, nil, false, contracts.Response{}, &out, &err)
	if code != 0 || strings.Contains(out.String(), "\"runtime-response\"") || !strings.Contains(out.String(), "9007199254740993") || !strings.Contains(out.String(), "registered") {
		t.Fatal(out.String(), err.String())
	}
}

func TestHumanTranscriptCorrectionPreservesLiteralFlags(t *testing.T) {
	var out, stderr bytes.Buffer
	notice := "Transcript input exceeded its elected byte limit. Increase --transcript-max-bytes (manifest transcript_max_bytes), then retry this work."
	result := map[string]any{"result": map[string]any{"items": []any{map[string]any{"state": "failed", "error": "input_limit", "notices": []any{notice}}}}}
	outputTo(result, contracts.Fail("input_limit"), false, contracts.Response{}, &out, &stderr)
	if out.Len() != 0 || !strings.Contains(stderr.String(), notice) {
		t.Fatal("actionable transcript notice changed", out.String(), stderr.String())
	}
}
func TestHumanDocumentDoesNotDumpBase64(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"recordings", "document"}}
	var out, err bytes.Buffer
	content := "Readable document fixture"
	raw := base64.StdEncoding.EncodeToString([]byte(content))
	outputTo(map[string]any{"encoding": "base64", "data": raw, "complete": true, "offset": 0, "next_offset": len(content)}, nil, false, contracts.Response{}, &out, &err)
	if strings.Contains(out.String(), raw) || !strings.Contains(out.String(), content) {
		t.Fatal(out.String())
	}
}
func TestHumanOutputBoundsAndTerminalControls(t *testing.T) {
	var out, err bytes.Buffer
	items := make([]any, 1000)
	for i := range items {
		items[i] = map[string]any{"name": "fixture\x1b[31m\r\n" + strings.Repeat("x", 10000)}
	}
	outputTo(map[string]any{"items": items}, nil, false, contracts.Response{}, &out, &err)
	if len(out.String()) > 65536 || strings.Contains(out.String(), "\x1b") || strings.Count(out.String(), "\n") > 130 {
		t.Fatal("unbounded/terminal output", out.Len())
	}
}
