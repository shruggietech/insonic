// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workers"
)

var humanIdentityKeys = []string{"id", "work_id", "job_id", "media_id", "recording_id", "model_id", "pipeline_id", "dataset_id", "speaker_id", "publication_id", "name", "model_name", "model_version", "version", "upstream_revision", "revision", "pipeline_revision", "mapping_revision", "manifest_digest", "document_digest", "digest", "sha256", "state", "registry_state", "readiness", "verification"}

func quotedModelIdentity(key string) bool {
	op := strings.Join(operatorContext.Path, " ")
	if !strings.HasPrefix(op, "models ") && !strings.HasPrefix(op, "work ") && !strings.HasPrefix(op, "pipelines ") {
		return false
	}
	switch key {
	case "name", "model_name", "model_version", "version", "upstream_revision":
		return true
	}
	return false
}

func humanFieldValue(key string, value any) string {
	if text, ok := value.(string); ok && quotedModelIdentity(key) {
		return strconv.Quote(text)
	}
	return humanValue(value)
}

// Chunk long exact identities into bounded display lines instead of silently
// changing the identity. Quoted whitespace/control characters remain literal.
func humanExactField(b *humanBuffer, key string, value any) {
	text := humanFieldValue(key, value)
	if raw, ok := value.(string); ok {
		text = raw
		if quotedModelIdentity(key) {
			text = strconv.Quote(raw)
		}
	}
	if len(text) <= 440 {
		b.line("%s: %s", humanLabel(key), text)
		return
	}
	b.line("%s: concatenate the following display lines.", humanLabel(key))
	for len(text) > 0 {
		n := min(480, len(text))
		for n > 0 && !utf8.ValidString(text[:n]) {
			n--
		}
		b.line("  %s", text[:n])
		text = text[n:]
	}
}

func humanSummary(b *humanBuffer, value any, depth int) {
	if depth > 3 {
		return
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return
	}
	humanWorkerProbe(b, fields)
	if environments, ok := fields["environments"].(map[string]any); ok {
		for _, role := range []string{"recognition", "diarization"} {
			if inspection, ok := environments[role].(map[string]any); ok {
				humanWorkerProbe(b, inspection)
			}
		}
	}
	for _, key := range humanIdentityKeys {
		if v, exists := fields[key]; exists && v != nil {
			switch v.(type) {
			case map[string]any, []any:
				continue
			}
			humanExactField(b, key, v)
		}
	}
	// Manifest identities are essential even when materialized file inventories
	// would exhaust the detail budget before reaching the manifest object.
	if manifest, ok := fields["manifest"].(map[string]any); ok && manifest["kind"] == "base-model-manifest" && quotedModelIdentity("name") {
		for _, key := range []string{"name", "model_version", "upstream_revision"} {
			text, exists := manifest[key].(string)
			if !exists {
				continue
			}
			rootKey := key
			if key == "model_version" {
				rootKey = "version"
			}
			if current, exists := fields[rootKey].(string); !exists || current != text {
				humanExactField(b, key, text)
			}
		}
	}
	if tasks, ok := fields["tasks"].(map[string]any); ok {
		names := make([]string, 0, len(tasks))
		for name := range tasks {
			names = append(names, name)
		}
		sort.Strings(names)
		// Every task conclusion precedes prerequisite inventories and corrections.
		for _, name := range names {
			if task, ok := tasks[name].(map[string]any); ok {
				b.line("Task %s: %s; usable: %s.", name, humanValue(task["state"]), humanValue(task["usable"]))
			}
		}
		for _, name := range names {
			task, ok := tasks[name].(map[string]any)
			if !ok {
				continue
			}
			seen := map[string]bool{}
			correction := func(message any) {
				text, ok := message.(string)
				if ok && text != "" && !seen[text] {
					seen[text] = true
					b.line("Correction (%s): %s", name, text)
				}
			}
			if actions, ok := task["actions"].([]any); ok {
				for _, message := range actions {
					correction(message)
				}
			}
			if prerequisites, ok := task["prerequisites"].([]any); ok {
				for _, raw := range prerequisites {
					if prerequisite, ok := raw.(map[string]any); ok {
						correction(prerequisite["correction"])
					}
				}
			}
		}
	}
	for _, key := range []string{"recording", "dataset", "result", "readiness", "processing_readiness", "worker_probe"} {
		if nested, ok := fields[key].(map[string]any); ok {
			humanSummary(b, nested, depth+1)
		}
	}
}

func humanJSONCommand(path string) string { return commandFor(path) + " --json" }

func invocationJSONCommand() string {
	args := operatorContext.Args
	if len(args) == 0 {
		args = operatorContext.Path
	}
	values := make([]string, len(args))
	for i, arg := range args {
		values[i] = quoteCLI(arg)
		if i < len(operatorContext.Path) {
			values[i] = arg
		}
	}
	return commandFor("--json " + strings.Join(values, " "))
}

func humanRetrieval(b *humanBuffer, root map[string]any) {
	op := strings.Join(operatorContext.Path, " ")
	// Existing read commands retain every operand/filter/input path when replayed.
	read := op == "doctor" || op == "search" || strings.HasPrefix(op, "timeline ")
	switch op {
	case "query run", "query native", "query explain", "query validate", "query services", "query assistance-show", "graph capabilities", "media metadata", "media raw", "recordings mappings", "recordings cues", "speakers select", "speakers diagnostics":
		read = true
	}
	for _, suffix := range []string{" show", " list", " inspect", " readiness", " results", " history", " capture", " document", " defaults", " catalog", " services", " location"} {
		read = read || strings.HasSuffix(op, suffix)
	}
	if read && len(operatorContext.Args) > 0 {
		humanExactField(b, "Complete JSON", invocationJSONCommand())
		return
	}
	route := humanMutationReadback(op, root)
	if strings.HasPrefix(op, "work ") {
		if id, ok := root["id"].(string); ok && contracts.ValidID(id) {
			route = "work show " + id
		}
	}
	for _, key := range []string{"work_id", "acquisition_work_id"} {
		if id, ok := root[key].(string); route == "" && ok && contracts.ValidID(id) {
			route = "work show " + id
			break
		}
	}
	if route == "" {
		for _, pair := range [][2]string{{"model_id", "models show"}, {"media_id", "media show"}, {"recording_id", "recordings show"}, {"pipeline_id", "pipelines show"}, {"publication_id", "artifacts show"}} {
			if id, ok := root[pair[0]].(string); ok && contracts.ValidID(id) {
				route = pair[1] + " " + id
				break
			}
		}
	}
	if route != "" {
		humanExactField(b, "Complete JSON", humanJSONCommand(route))
	} else if (op == "query assist" || op == "query assistance-test") && len(operatorContext.Args) > 0 {
		// These ephemeral provider operations have no durable readback resource.
		humanExactField(b, "Repeat as JSON (new provider request)", invocationJSONCommand())
	}
}

func humanMutationReadback(op string, root map[string]any) string {
	id, _ := root["id"].(string)
	if !contracts.ValidID(id) {
		id = selectedItem()
	}
	switch op {
	case "query assistance-set":
		return "query assistance-show"
	case "query save":
		if contracts.ValidID(id) {
			return "query show " + id
		}
	case "views save":
		if contracts.ValidID(id) {
			return "views show " + id
		}
	case "settings set", "media tools", "processing tools":
		return "settings show"
	case "setup save", "setup provision":
		return "setup show"
	case "setup draft save":
		return "setup draft show"
	case "pipelines set":
		if contracts.ValidID(id) {
			return "pipelines show " + id
		}
	case "terms set":
		if contracts.ValidID(id) {
			return "terms show " + id
		}
	case "speakers set", "speakers aliases":
		if contracts.ValidID(id) {
			return "speakers show " + id
		}
	case "models alias set", "models alias remove":
		if contracts.ValidID(id) {
			return "models alias show " + id
		}
	case "models source set", "models source remove":
		if contracts.ValidID(id) {
			return "models source show " + id
		}
	case "jobs start", "jobs cancel", "jobs retry":
		if job, ok := root["job_id"].(string); ok && contracts.ValidID(job) {
			return "jobs show " + job
		}
	case "graph publish", "graph rebuild":
		return "graph status"
	}
	return ""
}

func humanOrderedKeys(fields map[string]any) []string {
	rank := map[string]int{}
	for i, key := range humanIdentityKeys {
		rank[key] = i + 1
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, c := rank[keys[i]], rank[keys[j]]
		if a == 0 {
			a = 1000
		}
		if c == 0 {
			c = 1000
		}
		if a != c {
			return a < c
		}
		return keys[i] < keys[j]
	})
	return keys
}

func humanPageInput(b *humanBuffer, operation string, fields map[string]any, key string) {
	next := map[string]any{"offset": fields["next_offset"], "limit": 65536}
	if operation == "media capture" {
		for _, name := range []string{"section", "revision", "sha256"} {
			next[name] = fields[name]
		}
	} else {
		next["document_digest"] = fields["document_digest"]
	}
	raw, _ := json.Marshal(next)
	b.line("Next page input (%s): %s", key, string(raw))
	humanExactField(b, "Next page", commandFor(operation+" "+selectedItem()+" --input "+quoteCLI(key)))
}

// Continue flag-based lists using the actual accepted operands and filters.
func humanContinuationCommand(option, value string) string {
	args := operatorContext.Args
	if len(args) == 0 {
		args = operatorContext.Path
	}
	next := []string{}
	for i := 0; i < len(args); i++ {
		if i >= len(operatorContext.Path) && args[i] == option {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		next = append(next, args[i])
	}
	next = append(next, option, value)
	words := make([]string, len(next))
	for i, arg := range next {
		_, known := optionDefinitions[arg]
		if i < len(operatorContext.Path) || known {
			words[i] = arg
		} else {
			words[i] = quoteCLI(arg)
		}
	}
	return commandFor(strings.Join(words, " "))
}

func humanInputContinuation(b *humanBuffer, op, key string, value any, authority map[string]any) {
	fields := map[string]any{key: value}
	for name, item := range authority {
		if item != nil {
			fields[name] = item
		}
	}
	raw, _ := json.Marshal(fields)
	humanExactField(b, "Next page fields (page.json)", string(raw))
	b.line("Preserve all other current input fields, including filters and limit; set the fields above in page.json.")
	command := op
	switch op {
	case "speakers select", "speakers diagnostics", "recordings cues":
		command += " " + selectedItem()
	}
	humanExactField(b, "Next page", commandFor(command+" --input "+quoteCLI("page.json")))
}

// Paging authority belongs before optional rows for each real paging contract.
// No cursor field is interpreted as an option on commands that lack that route.
func humanOrdinaryPaging(b *humanBuffer, root map[string]any) {
	op := strings.Join(operatorContext.Path, " ")
	if next, ok := root["next_id"].(string); ok && contracts.ValidID(next) {
		switch op {
		case "media list", "models list", "models base list", "work list":
			humanExactField(b, "Next ID", next)
			humanExactField(b, "Next page", humanContinuationCommand("--after", next))
		case "pipelines list", "speakers list", "terms list", "models alias list", "models source list", "models dataset list", "models speaker list":
			humanInputContinuation(b, op, "after_id", next, nil)
		}
	}
	if next := root["next_generation"]; next != nil && op == "jobs history" {
		humanExactField(b, "Next generation", next)
		humanExactField(b, "Next page", commandFor("jobs history "+selectedItem()+" "+humanValue(next)))
	}
	if next := root["next_ordinal"]; next != nil {
		switch op {
		case "work results":
			humanExactField(b, "Next ordinal", next)
			humanExactField(b, "Next page", humanContinuationCommand("--after-ordinal", humanValue(next)))
		case "recordings cues":
			humanInputContinuation(b, op, "after_ordinal", next, map[string]any{"revision": root["revision"], "document_digest": root["document_digest"]})
		}
	}
	if next, ok := root["next_cursor"].(string); ok && next != "" && (op == "speakers select" || op == "speakers diagnostics") {
		humanInputContinuation(b, op, "cursor", next, nil)
	}
}

func humanWorkerProbe(b *humanBuffer, fields map[string]any) {
	raw, ok := fields["probe"]
	if !ok {
		return
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return
	}
	var p workers.Probe
	if json.Unmarshal(encoded, &p) != nil || !p.Valid() {
		return
	}
	role, _ := fields["role"].(string)
	if role != "recognition" && role != "diarization" {
		role = "environment"
	}
	b.line("Worker %s probe: %s at %s; native %d ms of %d ms.", role, p.Reason, p.Stage, p.ElapsedMS, p.BudgetMS)
	if p.Child != nil {
		b.line("Python probe: %d ms; hashed %d/%d files (%d bytes).", p.Child.ElapsedMS, p.Child.HashedFiles, p.Child.Files, p.Child.HashedBytes)
	}
}
