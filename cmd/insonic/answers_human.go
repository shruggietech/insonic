// SPDX-License-Identifier: Apache-2.0
package main

func humanAnswerSummary(b *humanBuffer, fields map[string]any) bool {
	p, ok := fields["package"].(map[string]any)
	if !ok || p["kind"] != "answer-evidence-context" {
		return false
	}
	humanExactField(b, "package_id", p["id"])
	b.line("Evidence state: %s; current source freshness: %s; cache reused: %s.", humanValue(p["state"]), humanValue(fields["freshness"]), humanValue(fields["cache_hit"]))
	items, _ := p["items"].([]any)
	b.line("Evidence spans: %d; distinct observed source families: %s.", len(items), humanValue(p["independent_families"]))
	if observation, ok := p["observation"].(map[string]any); ok {
		b.line("Graph: %s.", humanValue(observation["graph_state"]))
	}
	if c, ok := p["context"].(map[string]any); ok {
		b.line("Context: %s bytes, %s conservative token units; %s evidence spans omitted.", humanValue(c["bytes"]), humanValue(c["tokens"]), humanValue(c["omitted"]))
	}
	for _, set := range []map[string]any{p, fields} {
		if messages, ok := set["diagnostics"].([]any); ok {
			for _, message := range messages {
				b.line("Limitation: %s", humanValue(message))
			}
		}
	}
	if next, ok := p["next"].(string); ok && next != "" {
		humanExactField(b, "next_cursor", next)
		b.line("Continue: copy this value into request.cursor and keep the question and scope unchanged.")
	}
	b.line("Source families are provenance groups; unknown shared upstream origins remain possible. Context is extractive and is not a synthesized answer.")
	b.line("Complete evidence, citations and context: %s", invocationJSONCommand())
	return true
}
