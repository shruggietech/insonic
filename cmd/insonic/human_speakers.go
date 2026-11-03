// SPDX-License-Identifier: Apache-2.0
package main

func humanSpeakerApplication(b *humanBuffer, root map[string]any) bool {
	if _, ok := root["applied"]; !ok {
		return false
	}
	b.line("Contextual mappings: %s applied, %s preserved, %s retracted.", humanValue(root["applied"]), humanValue(root["preserved"]), humanValue(root["retracted"]))
	b.line("Unapplied voices: %s unknown, %s ambiguous, %s conflicting.", humanValue(root["unknown"]), humanValue(root["ambiguous"]), humanValue(root["conflicting"]))
	b.line("Contextual support is independent of acoustic matching. Full lineage and mapping revisions: use --json.")
	return true
}
func humanSpeakerWorkflow(b *humanBuffer, result map[string]any) {
	if stage, ok := result["stage"].(string); ok {
		b.line("Speaker workflow stage: %s.", stage)
	}
	if id, ok := result["awaiting_work_id"].(string); ok {
		b.line("Waiting for child work: %s", id)
	}
	if app, ok := result["application"].(map[string]any); ok {
		humanSpeakerApplication(b, app)
	}
	if rows, ok := result["enrollments"].([]any); ok {
		for _, raw := range rows {
			if row, ok := raw.(map[string]any); ok {
				b.line("Speaker %s: %s; dataset %s; model version %s.", humanValue(row["speaker_id"]), humanValue(row["state"]), humanValue(row["dataset_id"]), humanValue(row["version_id"]))
			}
		}
	}
	b.line("Hypotheses, contextual assignments and acoustic profiles remain distinct. Inspect child work and use --json for complete evidence.")
}
