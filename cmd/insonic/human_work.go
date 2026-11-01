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
	if completed, ok := windowProgressCount(result["completed_windows"]); ok {
		if total, valid := windowProgressCount(result["total_windows"]); valid && completed <= total {
			b.line("Windows complete: %d of %d", completed, total)
		} else if state == "succeeded" {
			b.line("Windows complete: %d", completed)
		}
	}
	if recordingID, _ := result["recording_id"].(string); contracts.ValidID(recordingID) {
		switch outcome, _ := result["state"].(string); outcome {
		case "ready", "no-speech", "no-timed-subtitles":
			b.line("Outcome: %s", humanLabel(outcome))
		}
	}
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
	humanRepairSummary(b, result)
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
	for _, adjustment := range attempt.ResourceAdjustments {
		b.line("Resource adjustment: batch %d -> %d (%s)", adjustment.PreviousBatchSize, adjustment.BatchSize, adjustment.Reason)
	}
	if attempt.FallbackReason != "" {
		b.line("Fallback cause: %s: %s", attempt.FallbackReason, contracts.Fail(attempt.FallbackReason).Message)
	}
}

func windowProgressCount(value any) (int64, bool) {
	raw, err := json.Marshal(value)
	var count int64
	if err != nil || value == nil || json.Unmarshal(raw, &count) != nil || count < 0 || count > 16384 {
		return 0, false
	}
	return count, true
}
