// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"errors"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"testing"
)

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("fixture output failure") }
func TestMachineOutputPreservesEnvelopeContextAndStreams(t *testing.T) {
	context := contracts.Response{RequestID: contracts.ID(), WorkspaceID: contracts.ID()}
	for _, code := range []string{"", "invalid_request", "incompatible_version", "unavailable", "conflict"} {
		var stdout, stderr bytes.Buffer
		var err error
		want := 0
		if code != "" {
			err = contracts.Fail(code)
			want = 1
		}
		if code == "invalid_request" {
			want = 2
		}
		if code == "incompatible_version" {
			want = 3
		}
		if code == "unavailable" {
			want = 4
		}
		got := outputTo(map[string]any{"state": "failed", "count": int64(9007199254740993)}, err, true, context, &stdout, &stderr)
		if got != want {
			t.Fatalf("%s exit %d", code, got)
		}
		raw := stdout.Bytes()
		if code != "" {
			raw = stderr.Bytes()
			if stdout.Len() != 0 {
				t.Fatal("failure polluted stdout")
			}
		} else if stderr.Len() != 0 {
			t.Fatal("success polluted stderr")
		}
		if schemas.ValidateResponse(raw) != nil {
			t.Fatalf("invalid envelope %s", raw)
		}
		for _, token := range []string{context.RequestID, context.WorkspaceID, "9007199254740993", "result"} {
			if !bytes.Contains(raw, []byte(token)) {
				t.Fatalf("missing %s: %s", token, raw)
			}
		}
	}
}
func TestOutputDeliveryFailureCannotReportSuccess(t *testing.T) {
	var stderr bytes.Buffer
	if outputTo(map[string]any{"ok": true}, nil, true, contracts.Response{}, failedWriter{}, &stderr) == 0 {
		t.Fatal("write failure succeeded")
	}
	if outputTo(make(chan int), nil, true, contracts.Response{}, &bytes.Buffer{}, &stderr) == 0 {
		t.Fatal("encode failure succeeded")
	}
}
