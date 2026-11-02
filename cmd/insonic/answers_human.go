// SPDX-License-Identifier: Apache-2.0
package main

func humanAnswerSummary(b *humanBuffer, fields map[string]any) bool {
	if p, ok := fields["answer"].(map[string]any); ok && p["kind"] == "grounded-answer" {
		return humanGroundedSummary(b, fields, p)
	}
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

func humanGroundedSummary(b *humanBuffer, fields, p map[string]any) bool {
	humanExactField(b, "answer_id", p["id"])
	humanExactField(b, "package_id", p["package_id"])
	answer, _ := p["answer"].(map[string]any)
	provider, _ := p["provider"].(map[string]any)
	b.line("Answer state: %s; accepted: %s; current source freshness: %s.", humanValue(answer["state"]), humanValue(fields["accepted"]), humanValue(fields["freshness"]))
	b.line("Route: %s; model: %s; evidence spans: %s; distinct source families: %s.", humanValue(p["route"]), humanValue(provider["model"]), humanValue(p["evidence_count"]), humanValue(p["independent_families"]))
	claims, _ := answer["claims"].([]any)
	for i, raw := range claims {
		if i >= 8 {
			b.line("Additional claims are available in complete JSON.")
			break
		}
		claim, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		b.line("%s (%s, %s): %s", humanValue(claim["id"]), humanValue(claim["kind"]), humanValue(claim["support"]), humanValue(claim["text"]))
		if reason, ok := claim["uncertainty"].(string); ok && reason != "" {
			b.line("Uncertainty: %s", reason)
		}
		citations, _ := claim["citations"].([]any)
		for j, raw := range citations {
			if j >= 4 {
				break
			}
			cite, ok := raw.(map[string]any)
			if ok {
				b.line("Source: %s", humanValue(cite["evidence_id"]))
			}
		}
	}
	for _, source := range []map[string]any{fields, p, answer} {
		for _, key := range []string{"diagnostics", "limitations"} {
			if list, ok := source[key].([]any); ok {
				for i, message := range list {
					if i >= 12 {
						break
					}
					b.line("Limitation: %s", humanValue(message))
				}
			}
		}
	}
	b.line("Citation validation establishes source authority; semantic support requires separate evaluation.")
	b.line("Complete claims, citations and provenance: %s", invocationJSONCommand())
	return true
}
func humanAnswerWork(b *humanBuffer, root, result map[string]any) {
	election, _ := root["election"].(map[string]any)
	if election == nil {
		election = result
	}
	provider, _ := election["provider"].(map[string]any)
	b.line("Route: %s; model: %s; provider budget: %s ms; repair cap: %s.", humanValue(election["route"]), humanValue(provider["model"]), humanValue(election["provider_timeout_ms"]), humanValue(election["max_repairs"]))
	b.line("Evidence spans: %s; distinct source families: %s.", humanValue(result["evidence_count"]), humanValue(result["independent_families"]))
	if budget, ok := election["budget"].(map[string]any); ok {
		b.line("Retrieval budget: %s ms; %s items; context: %s bytes.", humanValue(budget["deadline_ms"]), humanValue(budget["items"]), humanValue(budget["context_bytes"]))
	}
	if id, ok := result["answer_id"].(string); ok && id != "" {
		humanExactField(b, "answer_id", id)
		b.line("Inspect: %s", commandFor("answers show "+id))
	}
	if id, ok := root["retry_of"].(string); ok && id != "" {
		humanExactField(b, "retry_of", id)
	}
	if list, ok := result["limitations"].([]any); ok {
		for i, v := range list {
			if i >= 12 {
				break
			}
			b.line("Limitation: %s", humanValue(v))
		}
	}
}
