// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestPartialDatesStaySuccessfulAndQuietKeepsWarnings(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id := contracts.ID()
	operatorContext = invocation{Path: []string{"work", "wait"}, Args: []string{"work", "wait", id}, Root: "Library"}
	result := contracts.Response{Result: map[string]any{"id": id, "state": "succeeded", "result": map[string]any{"partial": true, "items": []any{map[string]any{"ordinal": 1, "media_id": contracts.ID(), "state": "admitted", "date_state": "ambiguous", "date_unresolved": true}}}}}
	var out, errOut bytes.Buffer
	if code := outputTo(result, nil, false, contracts.Response{}, &out, &errOut); code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), "1 admitted, 0 failed, 1 date warnings") || !strings.Contains(out.String(), "--originated-on") {
		t.Fatal(code, out.String(), errOut.String())
	}
	operatorContext.Quiet = true
	out.Reset()
	if code := outputTo(result, nil, false, contracts.Response{}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "Warning:") || strings.Contains(out.String(), "completed.") {
		t.Fatal(code, out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := outputTo(result, nil, true, contracts.Response{}, &out, &errOut); code != 0 || !strings.HasPrefix(out.String(), "{") || strings.Contains(out.String(), "Warning:") {
		t.Fatal(code, out.String())
	}
}
func TestWaitBeforeFirstObservationHasExactContinuationCommands(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id := contracts.ID()
	operatorContext = invocation{Path: []string{"work", "wait"}, Args: []string{"work", "wait", id}}
	var out, errOut bytes.Buffer
	if code := outputTo(nil, contracts.Condition("unavailable", "wait_timeout"), false, contracts.Response{}, &out, &errOut); code != 4 || !strings.Contains(errOut.String(), "work cancel "+id) || !strings.Contains(errOut.String(), "No durable state") {
		t.Fatal(code, errOut.String())
	}
}

func TestQuietBoundedWarningDisplayIdentifiesOmittedItems(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id := contracts.ID()
	operatorContext = invocation{Path: []string{"work", "wait"}, Args: []string{"work", "wait", id}, Quiet: true}
	items := make([]any, 65)
	for i := range items {
		items[i] = map[string]any{"ordinal": i + 1, "media_id": contracts.ID(), "state": "admitted", "date_unresolved": i == 64, "date_state": "ambiguous"}
	}
	// The undisplayed item carries the warning. Quiet must retain an explicit
	// omission and an exact retrieval route rather than imply a clean batch.
	for i := 0; i < 64; i++ {
		items[i].(map[string]any)["date_state"] = "resolved"
	}
	result := contracts.Response{Result: map[string]any{"id": id, "state": "succeeded", "result": map[string]any{"items": items}}}
	var out, errOut bytes.Buffer
	if code := outputTo(result, nil, false, contracts.Response{}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "Warning: Additional item details omitted") || !strings.Contains(out.String(), "work results "+id) {
		t.Fatal(code, out.String(), errOut.String())
	}
}
func TestHumanTerminalWorkDoesNotOfferCancellation(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id := contracts.ID()
	operatorContext = invocation{Path: []string{"work", "wait"}, Args: []string{"work", "wait", id}}
	var out, errOut bytes.Buffer
	outputTo(map[string]any{"id": id, "state": "succeeded"}, nil, false, contracts.Response{}, &out, &errOut)
	if strings.Contains(out.String(), "work cancel") {
		t.Fatal(out.String())
	}
}
func TestHumanFailureWithoutResultDoesNotPrintUnsetResult(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"models", "verify"}}
	var out, errOut bytes.Buffer
	outputTo(nil, contracts.Condition("unavailable", "input_missing"), false, contracts.Response{}, &out, &errOut)
	if strings.Contains(errOut.String(), "not set") {
		t.Fatal(errOut.String())
	}
}

func TestQuietWarningLineLimitIdentifiesTruncation(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id := contracts.ID()
	operatorContext = invocation{Path: []string{"work", "wait"}, Args: []string{"work", "wait", id}, Quiet: true}
	items := make([]any, 64)
	for i := range items {
		items[i] = map[string]any{"ordinal": i + 1, "media_id": contracts.ID(), "date_state": "ambiguous", "date_unresolved": true}
	}
	var out, errOut bytes.Buffer
	outputTo(map[string]any{"id": id, "state": "succeeded", "result": map[string]any{"items": items}}, nil, false, contracts.Response{}, &out, &errOut)
	if !strings.Contains(out.String(), "Display limit reached") || strings.Count(out.String(), "\n") > 130 {
		t.Fatal("quiet warning truncation not identified", out.String())
	}
}

func TestPublicHumanFieldsRetainContractAndQueryColumnNames(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	cases := []struct {
		path     []string
		result   any
		expected []string
	}{
		{[]string{"settings", "show"}, map[string]any{"sections": []any{map[string]any{"name": "processing", "value": map[string]any{"parallelism": 2}}}}, []string{"Parallelism: 2"}},
		{[]string{"doctor"}, map[string]any{"data": "public data directory"}, []string{"Data: public data directory"}},
		{[]string{"query", "native"}, map[string]any{"rows": []any{map[string]any{"value": "public value", "data": "public data", "owner": "public owner", "payload": "public payload", "token": "public token", "encoding": "base64"}}}, []string{"Value: public value", "Data: public data", "Owner: public owner", "Payload: public payload", "Token: public token"}},
	}
	for _, item := range cases {
		operatorContext = invocation{Path: item.path}
		var out, errOut bytes.Buffer
		outputTo(item.result, nil, false, contracts.Response{}, &out, &errOut)
		for _, expected := range item.expected {
			if !strings.Contains(out.String(), expected) {
				t.Errorf("%v omitted %q: %s", item.path, expected, out.String())
			}
		}
	}
}
func TestKnownCredentialAndWorkPayloadsStayOmitted(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	for _, item := range []struct {
		path   []string
		result any
	}{
		{[]string{"credentials", "status"}, map[string]any{"value": "synthetic private credential", "backend": "vault", "stored": true}},
		{[]string{"work", "show"}, map[string]any{"id": contracts.ID(), "kind": "media.import", "generation": 0, "state": "running", "payload": map[string]any{"data": "synthetic private credential"}}},
	} {
		operatorContext = invocation{Path: item.path}
		var out, errOut bytes.Buffer
		outputTo(item.result, nil, false, contracts.Response{}, &out, &errOut)
		if strings.Contains(out.String(), "synthetic private credential") {
			t.Fatal(out.String())
		}
	}
}
func TestMissingLockedVaultDoesNotClaimSelectedValueStored(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"credentials", "status"}}
	for _, stored := range []bool{false, true} {
		var out, errOut bytes.Buffer
		state := "missing"
		if stored {
			state = "locked"
		}
		outputTo(map[string]any{"credential_id": contracts.ID(), "backend": "vault", "state": state, "stored": stored, "locked": true}, nil, false, contracts.Response{}, &out, &errOut)
		if strings.Contains(out.String(), "remain stored") != stored || !strings.Contains(out.String(), "manager is locked") {
			t.Fatal(stored, out.String())
		}
	}
}
