// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
)

// Durable work has a known envelope. Show its outcome without recursively
// dumping model/configuration internals; read-only query columns remain intact.
func humanWorkSummary(b *humanBuffer, root map[string]any, errorShown bool) bool {
	if !strings.HasPrefix(strings.Join(operatorContext.Path, " "), "work ") {
		return false
	}
	id, _ := root["id"].(string)
	state, _ := root["state"].(string)
	if !contracts.ValidID(id) || state == "" {
		return false
	}
	b.line("Work: %s", id)
	if kind, ok := root["kind"].(string); ok && kind != "" {
		b.line("Operation: %s", kind)
	}
	b.line("State: %s", humanLabel(state))
	if phase, ok := root["phase"].(string); ok && phase != "" && (state == "pending" || state == "running") {
		b.line("Phase: %s", phase)
	}
	result, _ := root["result"].(map[string]any)
	for _, field := range [][2]string{{"recording_id", "Recording ID"}, {"media_id", "Media ID"}, {"model_id", "Model ID"}, {"version_id", "Model version ID"}, {"dataset_id", "Dataset ID"}, {"speaker_id", "Speaker ID"}} {
		if value, ok := result[field[0]].(string); ok && contracts.ValidID(value) {
			b.line("%s: %s", field[1], value)
		}
	}
	if !errorShown {
		var failure *contracts.Error
		if cause, ok := result["cause"].(map[string]any); ok {
			raw, err := json.Marshal(cause)
			var typed contracts.Error
			if err == nil && json.Unmarshal(raw, &typed) == nil {
				failure = contracts.SafeError(&typed)
			}
		}
		if failure == nil {
			if code, ok := root["error"].(string); ok && code != "" {
				failure = contracts.Fail(code)
			}
		}
		if failure != nil {
			b.line("Error: %s: %s", failure.Code, failure.Message)
			humanDeviceAttempt(b, failure)
		}
	}
	humanWorkerProbe(b, result)
	return true
}

// Attempt observations are separate from successful execution provenance.
func humanDeviceAttempt(b *humanBuffer, failure *contracts.Error) {
	attempt := contracts.SafeError(failure).DeviceAttempt
	if attempt == nil {
		return
	}
	b.line("Requested device: %s", attempt.RequestedDevice)
	b.line("Attempted devices: %s", strings.Join(attempt.AttemptedDevices, " -> "))
	if attempt.FallbackReason != "" {
		b.line("Fallback cause: %s: %s", attempt.FallbackReason, contracts.Fail(attempt.FallbackReason).Message)
	}
}
