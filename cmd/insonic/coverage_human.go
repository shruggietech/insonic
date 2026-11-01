// SPDX-License-Identifier: Apache-2.0
package main

func humanCoverageSummary(b *humanBuffer, fields map[string]any) bool {
	if fields["kind"] != "recording-coverage" {
		return false
	}
	b.line("Recording: %s; document revision: %s.", humanValue(fields["recording_id"]), humanValue(fields["document_revision"]))
	tracks, _ := fields["tracks"].([]any)
	for _, v := range tracks {
		track, ok := v.(map[string]any)
		if !ok {
			continue
		}
		m, _ := track["metrics"].(map[string]any)
		b.line("Track %s: selected audio %s us; cue coverage %s us.", humanValue(track["track_ordinal"]), humanValue(m["selected_audio_us"]), humanValue(m["cue_coverage_us"]))
		b.line("  observed speech %s us; uncovered observed speech %s us.", humanValue(m["observed_speech_us"]), humanValue(m["uncovered_observed_speech_us"]))
		b.line("  observed no-speech %s us; unknown activity %s us.", humanValue(m["observed_no_speech_us"]), humanValue(m["unknown_activity_us"]))
	}
	regions, _ := fields["regions"].([]any)
	b.line("Suspicious regions: %s total; %d on this page.", humanValue(fields["total_regions"]), len(regions))
	for _, v := range regions {
		r, ok := v.(map[string]any)
		if !ok {
			continue
		}
		b.line("  Track %s, %s..%s us: %s; uncertain=%s.", humanValue(r["track_ordinal"]), humanValue(r["start_us"]), humanValue(r["end_us"]), humanValue(r["reason"]), humanValue(r["uncertain"]))
	}
	if n := fields["next"]; n != nil {
		b.line("Next region offset: %s.", humanValue(n))
	}
	limits, _ := fields["limitations"].([]any)
	for _, v := range limits {
		b.line("Evidence limit: %s.", humanValue(v))
	}
	return true
}

// Durable repair reports distinguish a successful unchanged decision from an accepted replacement.
func humanRepairSummary(b *humanBuffer, fields map[string]any) bool {
	if fields["kind"] != "recording-repair" {
		return false
	}
	b.line("Repair: %s; reason: %s.", humanValue(fields["status"]), humanValue(fields["reason"]))
	selection, _ := fields["selection"].(map[string]any)
	windows, _ := selection["windows"].([]any)
	scopes, _ := selection["scopes"].([]any)
	attempts, _ := fields["attempts"].(map[string]any)
	total := int64(0)
	for _, value := range attempts {
		if n, ok := windowProgressCount(value); ok {
			total += n
		}
	}
	b.line("Repair scope: %d regions; %d windows; %d attempts.", len(scopes), len(windows), total)
	for _, phase := range []string{"before", "after"} {
		tracks, _ := fields[phase].([]any)
		for _, value := range tracks {
			track, ok := value.(map[string]any)
			if !ok {
				continue
			}
			metrics, _ := track["metrics"].(map[string]any)
			b.line("  %s track %s: uncovered observed speech %s us; duplicate overlap %s us; boundary inconsistencies %s.", phase, humanValue(track["track_ordinal"]), humanValue(metrics["uncovered_observed_speech_us"]), humanValue(metrics["duplicate_overlap_us"]), humanValue(metrics["boundary_inconsistencies"]))
		}
	}
	return true
}
