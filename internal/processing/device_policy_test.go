// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestDevicePolicyFiniteFallback(t *testing.T) {
	for _, requested := range []string{"", "cpu", "cuda", "auto"} {
		t.Run(requested, func(t *testing.T) {
			calls := []string{}
			raw, execution, err := runDevicePolicy(context.Background(), requested, func(_ context.Context, device string) ([]byte, error) {
				calls = append(calls, device)
				if device == "cuda" {
					return nil, contracts.Fail("cuda_resource_exhausted")
				}
				return []byte("ok"), nil
			})
			want := []string{"cpu"}
			if requested == "auto" || requested == "cuda" {
				want = []string{"cuda", "cpu"}
			}
			if err != nil || string(raw) != "ok" || !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(execution.AttemptedDevices, want) || execution.Device != "cpu" {
				t.Fatalf("calls=%v execution=%+v err=%v", calls, execution, err)
			}
			if len(want) == 2 && execution.FallbackReason != "cuda_resource_exhausted" {
				t.Fatal(execution)
			}
		})
	}
}

func TestDevicePolicyNeverRetriesUnknownInputOrCancellation(t *testing.T) {
	for _, code := range []string{"engine_failed", "invalid_embedding", "invalid_timing", "model_unavailable", "unsupported_language", "cancelled"} {
		calls := 0
		_, _, err := runDevicePolicy(context.Background(), "auto", func(context.Context, string) ([]byte, error) { calls++; return nil, contracts.Fail(code) })
		if err == nil || calls != 1 {
			t.Fatalf("%s calls=%d err=%v", code, calls, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, _, err := runDevicePolicy(ctx, "cuda", func(context.Context, string) ([]byte, error) {
		calls++
		cancel()
		return nil, contracts.Fail("cuda_resource_exhausted")
	})
	if err == nil || calls != 1 {
		t.Fatalf("cancel calls=%d err=%v", calls, err)
	}
}

func TestDeviceObservationRejectsContradictoryOrInjectedClaims(t *testing.T) {
	execution := DeviceExecution{RequestedDevice: "auto", AttemptedDevices: []string{"cuda", "cpu"}, Device: "cpu", FallbackReason: "cuda_no_device", Count: 1}
	raw, err := annotateDeviceResult([]byte(`{"provenance":{"device":"cpu"},"srt":""}`), execution)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Provenance map[string]any `json:"provenance"`
	}
	json.Unmarshal(raw, &result)
	if !validProvenance(result.Provenance) {
		t.Fatal(string(raw))
	}
	for _, bad := range []string{`{"provenance":{"device":"cuda"}}`, `{"provenance":{"device":"cpu","requested_device":"cpu"}}`, `{"provenance":{}}`} {
		if _, err := annotateDeviceResult([]byte(bad), execution); err == nil {
			t.Fatal(bad)
		}
	}
	result.Provenance["attempted_devices"] = []any{"cpu", "cuda"}
	if validProvenance(result.Provenance) {
		t.Fatal("accepted reversed attempts")
	}
}

func TestDevicePolicyRetriesOnlyFiveGeneratedCUDAReasonsWithinOriginalContext(t *testing.T) {
	for _, reason := range []string{"cuda_no_device", "cuda_library_missing", "cuda_library_incompatible", "cuda_compute_unsupported", "cuda_resource_exhausted"} {
		for _, requested := range []string{"cpu", "auto", "cuda"} {
			ctx, cancel := context.WithCancel(context.Background())
			calls := 0
			_, execution, err := runDevicePolicy(ctx, requested, func(attemptContext context.Context, device string) ([]byte, error) {
				if attemptContext != ctx {
					t.Fatal("attempt replaced original context")
				}
				calls++
				return nil, workerFailure([]byte("{\"error\":\""+reason+"\"}"), contracts.Fail("operation_failed"))
			})
			cancel()
			want := 2
			if requested == "cpu" {
				want = 1
			}
			if err == nil || calls != want || len(execution.AttemptedDevices) != want || execution.Device != "" {
				t.Fatalf("%s %s calls=%d execution=%+v err=%v", requested, reason, calls, execution, err)
			}
		}
	}
}

func TestLocalLanguageMatchesOnePinnedLanguage(t *testing.T) {
	for _, language := range []string{"", "auto", "en", "yue", "haw"} {
		if err := ValidateLocalLanguage(language); err != nil {
			t.Fatal(language, err)
		}
	}
	for _, language := range []string{"xx", "en es", "zh ", "EN", "english"} {
		if err := ValidateLocalLanguage(language); err == nil {
			t.Fatal("accepted unsupported local language", language)
		}
	}
}

func TestFailedDeviceAttemptsRemainStructuredWithoutClaimingSuccess(t *testing.T) {
	_, _, err := runDevicePolicy(context.Background(), "auto", func(_ context.Context, device string) ([]byte, error) {
		if device == "cuda" {
			return nil, contracts.Fail("cuda_resource_exhausted")
		}
		return nil, contracts.Fail("engine_failed")
	})
	safe := contracts.SafeError(err)
	raw, _ := json.Marshal(safe)
	var envelope map[string]json.RawMessage
	json.Unmarshal(raw, &envelope)
	var attempt map[string]any
	if json.Unmarshal(envelope["device_attempt"], &attempt) != nil || safe.Code != "engine_failed" || attempt["requested_device"] != "auto" || attempt["fallback_reason"] != "cuda_resource_exhausted" {
		t.Fatalf("lost failed attempts: %s", raw)
	}
	if _, exists := attempt["device"]; exists {
		t.Fatal("failed execution claimed actual device")
	}
	execution := DeviceExecution{RequestedDevice: "auto", AttemptedDevices: []string{"cuda", "cpu"}, Device: "cpu", FallbackReason: "cuda_library_missing", Count: 1}
	_, err = annotateDeviceResult([]byte(`{"provenance":{"device":"cuda"}}`), execution)
	raw, _ = json.Marshal(contracts.SafeError(err))
	if !bytes.Contains(raw, []byte("device_attempt")) || !bytes.Contains(raw, []byte("invalid_engine_output")) {
		t.Fatalf("output validation lost attempts: %s", raw)
	}
}

func TestNativeChecksumFailureRetainsCauseAndNeverFallsBack(t *testing.T) {
	for _, policy := range []string{"cpu", "auto", "cuda"} {
		calls := 0
		_, execution, err := runDevicePolicy(context.Background(), policy, func(context.Context, string) ([]byte, error) {
			calls++
			return nil, workerFailure([]byte("untrusted loader detail\n{\"error\":\"checksum_mismatch\"}\n"), contracts.Fail("subprocess_failed"))
		})
		safe := contracts.SafeError(err)
		device := "cuda"
		if policy == "cpu" {
			device = "cpu"
		}
		if calls != 1 || safe.Code != "checksum_mismatch" || safe.DeviceAttempt == nil || safe.DeviceAttempt.RequestedDevice != policy || !reflect.DeepEqual(safe.DeviceAttempt.AttemptedDevices, []string{device}) || safe.DeviceAttempt.FallbackReason != "" || execution.Device != "" {
			t.Fatalf("integrity failure changed policy: calls=%d execution=%+v error=%+v", calls, execution, safe)
		}
		raw, _ := json.Marshal(safe)
		if bytes.Contains(raw, []byte("untrusted loader detail")) {
			t.Fatal("unbounded loader detail retained")
		}
	}
}
