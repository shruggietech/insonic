// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

const s062WorkID = "11111111-1111-4111-8111-111111111111"
const s062RecordingID = "22222222-2222-4222-8222-222222222222"

func TestS062CompletedWorkGolden(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"work", "wait"}, Args: []string{"work", "wait", s062WorkID}}
	result := map[string]any{"id": s062WorkID, "kind": "recording.process", "state": "succeeded", "generation": 1, "error": "", "payload": map[string]any{"private": "not rendered"}, "result": map[string]any{"recording_id": s062RecordingID, "state": "ready", "provenance": map[string]any{"worker_sha256": "detailed-worker-identity", "model": map[string]any{"layers": 32}}}}
	response := contracts.Response{Result: result}
	before, _ := json.Marshal(response)
	text, err := renderHuman(response)
	if err != nil {
		t.Fatal(err)
	}
	want := "Work wait completed.\nWork: " + s062WorkID + "\nOperation: recording.process\nState: Succeeded\nRecording ID: " + s062RecordingID + "\nComplete JSON: insonic work show " + s062WorkID + " --json\n"
	if text != want {
		t.Fatalf("want:\n%s\ngot:\n%s", want, text)
	}
	after, _ := json.Marshal(response)
	if string(before) != string(after) {
		t.Fatal("human rendering mutated machine result")
	}
	for _, mode := range []string{"verbose", "debug"} {
		operatorContext.Verbose = mode == "verbose"
		operatorContext.Debug = mode == "debug"
		text, err = renderHuman(response)
		if err != nil || !strings.Contains(text, "detailed-worker-identity") || strings.Contains(text, "not rendered") {
			t.Fatal(mode, text, err)
		}
	}
}

func TestS062MissingWorkAndUnknownCommandGolden(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"work", "show"}, Args: []string{"work", "show", s062WorkID}}
	text, err := renderHuman(contracts.Response{Error: contracts.Fail("not_found")})
	if err != nil {
		t.Fatal(err)
	}
	want := "not_found: The item is unavailable in this workspace.\nFind work: insonic work list\n"
	if text != want {
		t.Fatalf("got %q, want %q", text, want)
	}
	operatorContext = invocation{}
	text, err = renderHuman(contracts.Response{Error: contracts.Fail("invalid_request")})
	if err != nil {
		t.Fatal(err)
	}
	want = "invalid_request: The request is invalid.\nHelp: insonic --help\n"
	if text != want {
		t.Fatalf("got %q, want %q", text, want)
	}
}

func TestS062WorkOutputRetainsWarningsAndErrors(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"work", "show"}, Args: []string{"work", "show", s062WorkID}}
	for _, state := range []string{"succeeded", "failed", "pending", "running"} {
		result := map[string]any{"id": s062WorkID, "state": state, "error": "", "result": map[string]any{"items": []any{}}}
		if state == "failed" {
			result["error"] = "engine_failed"
		}
		text, err := renderHuman(contracts.Response{Result: result})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(text, "State:") != 1 {
			t.Fatal(text)
		}
		active := state == "pending" || state == "running"
		if strings.Contains(text, "Wait:") != active || strings.Contains(text, "Cancel:") != active {
			t.Fatal(text)
		}
		if strings.Contains(text, "Error:") != (state == "failed") {
			t.Fatal(text)
		}
		if state == "failed" && !strings.Contains(text, contracts.Fail("engine_failed").Message) {
			t.Fatal(text)
		}
		if !strings.Contains(text, "0 admitted, 0 failed") {
			t.Fatal(text)
		}
	}
}

func TestS062LifecycleGolden(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"artifacts", "retire"}, Args: []string{"artifacts", "retire", s062WorkID}}
	for _, reason := range []string{"artifact_retained", "artifact_active_lease", "artifact_grace_period", "artifact_retirement_active"} {
		failure := contracts.Condition("conflict", reason)
		text, err := renderHuman(contracts.Response{Error: failure})
		if err != nil {
			t.Fatal(err)
		}
		if failure.Reason != reason || text != "conflict: "+failure.Message+"\n" || strings.Contains(text, "current revision") {
			t.Fatal(reason, text)
		}
	}
}

func TestS062WorkWarningsAndFieldFailureSurviveConciseOutput(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"work", "show"}, Args: []string{"work", "show", s062WorkID}}
	notice := "CUDA was unavailable; processing completed on CPU."
	cause := contracts.InputFailure("unsupported_language", contracts.InputViolation{Pointer: "/recognition/language", Expected: "string"})
	response := contracts.Response{Result: map[string]any{"id": s062WorkID, "state": "failed", "error": "invalid_request", "result": map[string]any{"cause": cause, "notices": []string{notice}}}}
	for _, quiet := range []bool{false, true} {
		operatorContext.Quiet = quiet
		text, err := renderHuman(response)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(text, notice) != 1 {
			t.Fatal(text)
		}
		if !quiet && (!strings.Contains(text, "/recognition/language") || !strings.Contains(text, "supported faster-whisper")) {
			t.Fatal(text)
		}
	}
}

func TestS062FailedDeviceAttemptsRenderWithoutSuccessfulDeviceClaims(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	cause := contracts.WithDeviceAttempt(contracts.Fail("engine_failed"), contracts.DeviceAttempt{RequestedDevice: "auto", AttemptedDevices: []string{"cuda", "cpu"}, FallbackReason: "cuda_resource_exhausted"})
	for _, mode := range []string{"normal", "verbose", "debug"} {
		operatorContext = invocation{Path: []string{"work", "show"}, Args: []string{"work", "show", s062WorkID}, Verbose: mode == "verbose", Debug: mode == "debug"}
		for _, topLevel := range []bool{false, true} {
			response := contracts.Response{Result: map[string]any{"id": s062WorkID, "state": "failed", "error": "engine_failed", "result": map[string]any{"cause": cause}}}
			if topLevel {
				response.Error = cause
			}
			text, err := renderHuman(response)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Requested device: auto", "Attempted devices: cuda -> cpu", "Fallback cause: cuda_resource_exhausted"} {
				if strings.Count(text, want) != 1 {
					t.Fatalf("%s top=%t missing/duplicate %s:\n%s", mode, topLevel, want, text)
				}
			}
			for _, bad := range []string{"completed on CPU", "Device: cpu", "Actual device:", "Wait:", "Cancel:"} {
				if strings.Contains(text, bad) {
					t.Fatalf("%s top=%t %s:\n%s", mode, topLevel, bad, text)
				}
			}
		}
	}
}

func TestS062InvalidDeviceAttemptNeverReachesHumanOutput(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	cause := contracts.Fail("engine_failed")
	cause.DeviceAttempt = &contracts.DeviceAttempt{RequestedDevice: "auto", AttemptedDevices: []string{"cuda", "private traceback"}, FallbackReason: "private CUDA text"}
	for _, mode := range []string{"normal", "verbose", "debug"} {
		operatorContext = invocation{Path: []string{"work", "show"}, Args: []string{"work", "show", s062WorkID}, Verbose: mode == "verbose", Debug: mode == "debug"}
		response := contracts.Response{Error: cause, Result: map[string]any{"id": s062WorkID, "state": "failed", "error": "engine_failed", "result": map[string]any{"cause": cause}}}
		text, err := renderHuman(response)
		if err != nil || strings.Contains(text, "private") || strings.Contains(text, "Attempted devices:") || strings.Contains(text, "Fallback cause:") {
			t.Fatal(mode, text, err)
		}
	}
}
