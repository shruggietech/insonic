// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"encoding/json"
	"math"
	"regexp"
)

var diagnosticCode = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)

func validDiagnostics(diagnostics []Diagnostic) bool {
	for _, item := range diagnostics {
		if !diagnosticCode.MatchString(item.Code) || item.Count < 0 || item.Value != nil && (math.IsNaN(*item.Value) || math.IsInf(*item.Value, 0)) {
			return false
		}
	}
	return true
}

// Worker provenance accepts only aggregate, noncontent values. Transcript text,
// turn arrays and nested result documents never enter the durable record here.
func validProvenance(value map[string]any) bool {
	allowed := map[string]bool{"segment_count": true, "native_timing_projection": true, "language": true, "beam_size": true, "vad_filter": true, "condition_on_previous_text": true, "compute_type": true, "speaker_count": true, "turn_count": true, "quality_diagnostics_enabled": true, "packages": true, "device": true, "threads": true, "wall_seconds": true, "cpu_seconds": true, "peak_rss_bytes": true, "audio_sha256": true, "sample_count": true, "sample_rate": true, "telemetry_enabled": true, "offline": true}
	allowed["embedding_minimum_samples"] = true
	allowed["model_boundary_policy"] = true
	for key, item := range value {
		if !allowed[key] {
			return false
		}
		if key == "packages" {
			packages, ok := item.(map[string]any)
			if !ok || len(packages) > 32 {
				return false
			}
			for name, version := range packages {
				text, ok := version.(string)
				if !ok || len(name) > 128 || len(text) > 128 {
					return false
				}
			}
			continue
		}
		switch typed := item.(type) {
		case string:
			if len(typed) > 256 {
				return false
			}
		case bool, nil:
		case float64:
			if math.IsNaN(typed) || math.IsInf(typed, 0) {
				return false
			}
		case json.Number:
			if _, err := typed.Float64(); err != nil {
				return false
			}
		default:
			return false
		}
	}
	return true
}
