// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/shruggietech/insonic/internal/contracts"
)

// DeviceExecution describes observed execution, never a readiness prediction.
// Count coalesces identical embedding batches without retaining clip content.
type DeviceExecution struct {
	RequestedDevice  string   `json:"requested_device"`
	AttemptedDevices []string `json:"attempted_devices"`
	Device           string   `json:"device"`
	FallbackReason   string   `json:"fallback_reason,omitempty"`
	Count            int64    `json:"count"`
}

func ValidDevice(device string) bool {
	return device == "" || device == "cpu" || device == "cuda" || device == "auto"
}

func fallbackFailure(code string) bool {
	switch code {
	case "cuda_no_device", "cuda_library_missing", "cuda_library_incompatible", "cuda_compute_unsupported", "cuda_resource_exhausted":
		return true
	}
	return false
}

func (e DeviceExecution) Valid() bool {
	if e.Count < 1 || e.Count > 1000000 || e.RequestedDevice == "" || !ValidDevice(e.RequestedDevice) {
		return false
	}
	if len(e.AttemptedDevices) == 1 {
		return e.FallbackReason == "" && e.Device == e.AttemptedDevices[0] && ((e.RequestedDevice == "cpu" && e.Device == "cpu") || (e.RequestedDevice != "cpu" && e.Device == "cuda"))
	}
	return len(e.AttemptedDevices) == 2 && e.RequestedDevice != "cpu" && e.AttemptedDevices[0] == "cuda" && e.AttemptedDevices[1] == "cpu" && e.Device == "cpu" && fallbackFailure(e.FallbackReason)
}

func AccumulateDeviceExecutions(existing []DeviceExecution, observations []DeviceExecution) ([]DeviceExecution, error) {
	for _, next := range observations {
		if !next.Valid() {
			return nil, contracts.Fail("invalid_engine_output")
		}
		found := false
		for i := range existing {
			if existing[i].RequestedDevice == next.RequestedDevice && existing[i].Device == next.Device && existing[i].FallbackReason == next.FallbackReason && slices.Equal(existing[i].AttemptedDevices, next.AttemptedDevices) {
				existing[i].Count += next.Count
				if !existing[i].Valid() {
					return nil, contracts.Fail("invalid_engine_output")
				}
				found = true
				break
			}
		}
		if !found {
			next.AttemptedDevices = slices.Clone(next.AttemptedDevices)
			existing = append(existing, next)
		}
		if len(existing) > 16 {
			return nil, contracts.Fail("invalid_engine_output")
		}
	}
	return existing, nil
}

// runDevicePolicy shares the caller's original deadline across fresh children.
// Legacy omitted values remain CPU. Admission freezes new defaults as auto.
func runDevicePolicy(ctx context.Context, requested string, attempt func(context.Context, string) ([]byte, error)) (result []byte, execution DeviceExecution, returned error) {
	defer func() { returned = withDeviceFailure(returned, execution) }()
	if !ValidDevice(requested) {
		return nil, DeviceExecution{}, contracts.Fail("unsupported_device")
	}
	if requested == "" {
		requested = "cpu"
	}
	device := "cuda"
	if requested == "cpu" {
		device = "cpu"
	}
	execution = DeviceExecution{RequestedDevice: requested, Count: 1}
	for {
		if ctx.Err() != nil {
			return nil, execution, contracts.Fail("cancelled")
		}
		execution.AttemptedDevices = append(execution.AttemptedDevices, device)
		raw, err := attempt(ctx, device)
		if ctx.Err() != nil {
			return nil, execution, contracts.Fail("cancelled")
		}
		if err == nil {
			execution.Device = device
			return raw, execution, nil
		}
		code := contracts.SafeError(err).Code
		if device != "cuda" || !fallbackFailure(code) {
			return nil, execution, err
		}
		execution.FallbackReason = code
		device = "cpu"
	}
}

func workerFailure(stderr []byte, cause error) error {
	var failure struct {
		Error string `json:"error"`
	}
	lines := bytes.Split(bytes.TrimSpace(stderr), []byte("\n"))
	if len(lines) > 0 && strict(lines[len(lines)-1], &failure) == nil {
		switch failure.Error {
		case "engine_ci_forbidden", "engine_unavailable", "model_unavailable", "incompatible_version", "checksum_mismatch", "unsupported_device", "invalid_embedding", "invalid_timing", "audio_limit", "unsupported_language", "engine_failed", "cuda_no_device", "cuda_library_missing", "cuda_library_incompatible", "cuda_compute_unsupported", "cuda_resource_exhausted":
			return contracts.Fail(failure.Error)
		}
	}
	return cause
}

// The launcher owns policy observations. A worker can report only the actual
// concrete device it was asked to execute; it cannot forge a fallback receipt.
func annotateDeviceResult(raw []byte, execution DeviceExecution) (resultBytes []byte, returned error) {
	defer func() { returned = withDeviceFailure(returned, execution) }()
	var result map[string]json.RawMessage
	if !execution.Valid() || strict(raw, &result) != nil {
		return nil, contracts.Fail("invalid_engine_output")
	}
	var provenance map[string]any
	if strict(result["provenance"], &provenance) != nil || provenance == nil || provenance["device"] != execution.Device {
		return nil, contracts.Fail("invalid_engine_output")
	}
	for _, key := range []string{"requested_device", "attempted_devices", "fallback_reason"} {
		if _, exists := provenance[key]; exists {
			return nil, contracts.Fail("invalid_engine_output")
		}
	}
	provenance["requested_device"] = execution.RequestedDevice
	provenance["attempted_devices"] = execution.AttemptedDevices
	if execution.FallbackReason != "" {
		provenance["fallback_reason"] = execution.FallbackReason
	}
	result["provenance"], _ = json.Marshal(provenance)
	return json.Marshal(result)
}

func deviceExecution(provenance map[string]any) (DeviceExecution, bool) {
	raw, err := json.Marshal(map[string]any{"requested_device": provenance["requested_device"], "attempted_devices": provenance["attempted_devices"], "device": provenance["device"], "fallback_reason": provenance["fallback_reason"], "count": 1})
	var execution DeviceExecution
	return executionFromJSON(raw, err, &execution)
}

func executionFromJSON(raw []byte, err error, execution *DeviceExecution) (DeviceExecution, bool) {
	if err != nil || json.Unmarshal(raw, execution) != nil || !execution.Valid() {
		return DeviceExecution{}, false
	}
	return *execution, true
}

func DeviceNotices(executions []DeviceExecution) []string {
	out := []string{}
	for _, execution := range executions {
		if execution.Valid() && execution.FallbackReason != "" {
			message := fmt.Sprintf("CUDA could not execute (%s); completed on CPU.", execution.FallbackReason)
			if !slices.Contains(out, message) {
				out = append(out, message)
			}
		}
	}
	return out
}

func DeviceProvenanceNotices(provenance map[string]any) []string {
	if execution, valid := deviceExecution(provenance); valid {
		return DeviceNotices([]DeviceExecution{execution})
	}
	return nil
}

func withDeviceFailure(err error, execution DeviceExecution) error {
	if err == nil || len(execution.AttemptedDevices) == 0 {
		return err
	}
	return contracts.WithDeviceAttempt(err, contracts.DeviceAttempt{RequestedDevice: execution.RequestedDevice, AttemptedDevices: execution.AttemptedDevices, FallbackReason: execution.FallbackReason})
}

func withDeviceProvenanceFailure(err error, provenance map[string]any) error {
	if execution, valid := deviceExecution(provenance); valid {
		return withDeviceFailure(err, execution)
	}
	return err
}
