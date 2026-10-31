// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestS062WorkerFailuresSurviveSafeError(t *testing.T) {
	for _, code := range []string{"cuda_no_device", "cuda_library_missing", "cuda_library_incompatible", "cuda_compute_unsupported", "cuda_resource_exhausted", "unsupported_language", "engine_failed"} {
		e := SafeError(fmt.Errorf("private raw worker text: %w", &Error{Code: code, Message: "private raw worker text"}))
		if e.Code != code || strings.Contains(e.Message, "private") || e.Message == "The operation failed." {
			t.Fatal(code, e)
		}
	}
}
func TestS062RetirementReasonsSurviveSafeError(t *testing.T) {
	for _, reason := range []string{"artifact_retained", "artifact_active_lease", "artifact_grace_period", "artifact_retirement_active"} {
		e := SafeError(Condition("conflict", reason))
		if e.Code != "conflict" || e.Reason != reason || strings.Contains(e.Message, "revision") {
			t.Fatal(e)
		}
	}
}

func TestS062LanguageFieldSurvivesSafeError(t *testing.T) {
	e := SafeError(InputFailure("unsupported_language", InputViolation{Pointer: "/recognition/language", Expected: "string"}))
	if e.Code != "invalid_request" || e.Reason != "unsupported_language" || e.Validation == nil || e.Validation.Pointer != "/recognition/language" || !strings.Contains(e.Message, "auto") {
		t.Fatal(e)
	}
}

func TestS062DeviceAttemptSurvivesSanitizedPersistence(t *testing.T) {
	input := DeviceAttempt{RequestedDevice: "auto", AttemptedDevices: []string{"cuda", "cpu"}, FallbackReason: "cuda_resource_exhausted"}
	original := InputFailure("unsupported_language", InputViolation{Pointer: "/recognition/language", Expected: "string"})
	original.Message = "private raw engine detail"
	got := WithDeviceAttempt(fmt.Errorf("private wrapper: %w", original), input)
	input.AttemptedDevices[0] = "private mutated path"
	raw, err := json.Marshal(got)
	if err != nil || strings.Contains(string(raw), "private") || strings.Contains(string(raw), "\"device\"") {
		t.Fatal(string(raw), err)
	}
	var durable Error
	if err = json.Unmarshal(raw, &durable); err != nil {
		t.Fatal(err)
	}
	safe := SafeError(&durable)
	if safe.Code != "invalid_request" || safe.Reason != "unsupported_language" || safe.Validation == nil || safe.Validation.Pointer != "/recognition/language" || safe.DeviceAttempt == nil || !reflect.DeepEqual(safe.DeviceAttempt.AttemptedDevices, []string{"cuda", "cpu"}) || safe.DeviceAttempt.FallbackReason != "cuda_resource_exhausted" {
		t.Fatalf("%+v", safe)
	}
	durable.DeviceAttempt.AttemptedDevices[1] = "private mutated device"
	if safe.DeviceAttempt.AttemptedDevices[1] != "cpu" {
		t.Fatal("safe receipt aliases original")
	}
	if WithDeviceAttempt(nil, input) != nil {
		t.Fatal("nil cause became failure")
	}
}
func TestS062DeviceAttemptRejectsUnboundedOrContradictoryClaims(t *testing.T) {
	invalid := []DeviceAttempt{
		{RequestedDevice: "auto"},
		{RequestedDevice: "private", AttemptedDevices: []string{"cuda"}},
		{RequestedDevice: "auto", AttemptedDevices: []string{"cpu"}},
		{RequestedDevice: "cpu", AttemptedDevices: []string{"cuda"}},
		{RequestedDevice: "cpu", AttemptedDevices: []string{"cpu"}, FallbackReason: "cuda_no_device"},
		{RequestedDevice: "auto", AttemptedDevices: []string{"cuda", "cpu"}},
		{RequestedDevice: "auto", AttemptedDevices: []string{"cpu", "cuda"}, FallbackReason: "cuda_no_device"},
		{RequestedDevice: "auto", AttemptedDevices: []string{"cuda", "cpu", "cpu"}, FallbackReason: "cuda_no_device"},
		{RequestedDevice: "cuda", AttemptedDevices: []string{"cuda"}, FallbackReason: "private traceback"},
	}
	for _, attempt := range invalid {
		e := Fail("engine_failed")
		e.DeviceAttempt = &attempt
		got := SafeError(e)
		if got.DeviceAttempt != nil || got.Code != "engine_failed" {
			t.Fatalf("invalid=%+v got=%+v", attempt, got)
		}
	}
	for _, attempt := range []DeviceAttempt{
		{RequestedDevice: "cpu", AttemptedDevices: []string{"cpu"}},
		{RequestedDevice: "auto", AttemptedDevices: []string{"cuda"}},
		{RequestedDevice: "cuda", AttemptedDevices: []string{"cuda"}, FallbackReason: "cuda_no_device"},
	} {
		got := WithDeviceAttempt(Fail("cancelled"), attempt)
		if got.DeviceAttempt == nil || got.Code != "cancelled" {
			t.Fatalf("%+v", got)
		}
	}
}
