// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

var operatorContext invocation

type humanBuffer struct {
	bytes.Buffer
	lines     int
	truncated bool
}

func safeText(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) || r == 0x2028 || r == 0x2029 {
			return ' '
		}
		return r
	}, s)
	if len(s) > 512 {
		s = s[:509]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
		s += "..."
	}
	return s
}
func (b *humanBuffer) line(format string, values ...any) {
	if b.lines >= 128 || b.Len() >= 65536-1024 {
		b.truncated = true
		return
	}
	raw := fmt.Sprintf(format, values...)
	s := safeText(raw)
	if len(s) < len(raw) {
		b.truncated = true
	}
	b.WriteString(s)
	b.WriteByte('\n')
	b.lines++
}
func humanLabel(s string) string {
	s = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(s)
	if s == "" {
		return "Result"
	}
	first, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(first)) + s[size:]
}
func humanValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "not set"
	case bool:
		if x {
			return "yes"
		}
		return "no"
	case string:
		return safeText(x)
	case json.Number:
		return x.String()
	default:
		return safeText(fmt.Sprint(v))
	}
}
func humanJSON(v any) (any, error) {
	raw, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var out any
	e = d.Decode(&out)
	return out, e
}
func humanFields(b *humanBuffer, v any, depth int) {
	if depth > 6 {
		return
	}
	switch x := v.(type) {
	case map[string]any:
		keys := humanOrderedKeys(x)
		for _, k := range keys {
			if hiddenHumanField(x, k) {
				b.line("%s%s: omitted from human output.", strings.Repeat("  ", depth), humanLabel(k))
				continue
			}
			value := x[k]
			switch item := value.(type) {
			case map[string]any:
				if embeddedRecordingDocument(x, k) {
					b.line("%sDocument: embedded transcript; retrieve complete JSON for cues and provenance.", strings.Repeat("  ", depth))
					continue
				}
				b.line("%s%s:", strings.Repeat("  ", depth), humanLabel(k))
				humanFields(b, item, depth+1)
			case []any:
				b.line("%s%s: %d", strings.Repeat("  ", depth), humanLabel(k), len(item))
				humanFields(b, item, depth+1)
			default:
				b.line("%s%s: %s", strings.Repeat("  ", depth), humanLabel(k), humanFieldValue(k, value))
			}
		}
	case []any:
		if len(x) == 0 {
			b.line("%sNo items.", strings.Repeat("  ", depth))
		}
		for i, item := range x {
			if i >= 64 {
				b.line("Additional items omitted; use the complete JSON command above for this result.")
				break
			}
			if _, ok := item.(map[string]any); ok {
				b.line("%sItem %d", strings.Repeat("  ", depth), i+1)
			}
			humanFields(b, item, depth+1)
		}
	default:
		b.line("%s%s", strings.Repeat("  ", depth), humanValue(x))
	}
}

// Summarize only an application's recording envelope. A read-only query may
// return any public column name, including a map named document.
func embeddedRecordingDocument(fields map[string]any, key string) bool {
	if key != "document" {
		return false
	}
	op := strings.Join(operatorContext.Path, " ")
	if !strings.HasPrefix(op, "recordings ") && !strings.HasPrefix(op, "media ") && !strings.HasPrefix(op, "work ") {
		return false
	}
	id, _ := fields["id"].(string)
	if id == "" {
		id, _ = fields["recording_id"].(string)
	}
	digest, _ := fields["document_digest"].(string)
	return contracts.ValidID(id) && len(digest) == 64 && fields["state"] != nil
}

func hiddenHumanField(fields map[string]any, key string) bool {
	op := strings.Join(operatorContext.Path, " ")
	// Query fields are user-selected public result columns, not credential input.
	if strings.HasPrefix(op, "query ") {
		return false
	}
	if (op == "recordings document" || op == "media capture") && fields["encoding"] == "base64" && key == "data" {
		return true
	}
	// Durable work is a known runtime structure; omit its internal input/owner
	// details without treating an unrelated public field name as a secret.
	if strings.HasPrefix(op, "work ") && fields["generation"] != nil && fields["kind"] != nil && fields["payload"] != nil {
		switch key {
		case "payload", "owner", "owner_token", "journal_receipt_id":
			return true
		}
	}
	if strings.HasPrefix(op, "credentials ") {
		switch key {
		case "value", "password", "passphrase", "token", "secret", "secrets", "input", "environment":
			return true
		}
	}
	return false
}
func commandFor(path string) string {
	prefix := "insonic"
	if operatorContext.Root != "" {
		prefix += " --workspace " + quoteCLI(operatorContext.Root)
	}
	return prefix + " " + path
}
func quoteCLI(s string) string {
	if runtime.GOOS == "windows" {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
func humanActions(b *humanBuffer, root map[string]any) {
	op := strings.Join(operatorContext.Path, " ")
	id := humanValue(root["id"])
	if id == "not set" {
		id = humanValue(root["work_id"])
	}
	if root["kind"] != nil && strings.HasSuffix(humanValue(root["kind"]), "import") || strings.HasPrefix(op, "work ") || op == "workers setup" {
		if contracts.ValidID(id) {
			b.line("Next: %s", commandFor("work show "+id))
			b.line("Wait: %s", commandFor("work wait "+id))
			if root["state"] == nil || root["state"] == "pending" || root["state"] == "running" {
				b.line("Cancel: %s", commandFor("work cancel "+id))
			}
			if root["state"] == "failed" || root["state"] == "interrupted" || root["state"] == "cancelled" {
				b.line("Retry: %s", commandFor("work retry "+id))
			}
			if root["result_inline"] == false {
				b.line("Results: %s", commandFor("work results "+id))
			}
		}
	}
	if strings.HasPrefix(op, "setup ") || op == "settings setup" {
		b.line("Setup is optional; accepted settings and running work remain available.")
		b.line("Reopen: %s", commandFor("setup show"))
		b.line("Task prerequisites: %s", commandFor("setup readiness"))
		if operatorContext.Root != "" {
			b.line("First import: %s", commandFor("media import "+quoteCLI("MEDIA_FILE")))
		}
	}
	if op == "workspace init" || op == "workspace show" || op == "doctor" {
		b.line("Client version: %s", contracts.Version)
		if operatorContext.Root != "" {
			b.line("Workspace: %s", operatorContext.Root)
		}
		b.line("Next import: %s", commandFor("media import "+quoteCLI("MEDIA_FILE")))
		b.line("Readiness: configured resources are not authenticated or inference-tested by startup.")
	}
	if strings.HasPrefix(op, "credentials ") {
		if root["backend"] == "session" {
			b.line("Session values last only for the current runtime owner.")
		}
		if root["locked"] == true {
			b.line("The vault manager is locked. Use credentials unlock with protected stdin.")
			if root["stored"] == true {
				b.line("The selected credential values remain stored in the vault.")
			}
		}
		b.line("Credential registration does not prove provider authentication.")
	}
	if strings.HasPrefix(op, "workers ") || op == "processing readiness" {
		b.line("Inspect task prerequisites: %s", commandFor("processing readiness"))
		b.line("Prepare or verify environments: %s", commandFor("workers plan --input SETUP_INPUT.json"))
		b.line("Model choices: %s", commandFor("models defaults"))
		b.line("Prerequisite verification is separate from measured model execution. Device/provider corrections apply only to new elections.")
	}
	if strings.HasPrefix(op, "models ") {
		if model, ok := root["model_id"].(string); ok && contracts.ValidID(model) {
			b.line("Inspect: %s", commandFor("models show "+model))
			b.line("Verify: %s", commandFor("models verify "+model))
			if root["readiness"] == "not_acquired" || root["readiness"] == "damaged" {
				b.line("Acquire exact bundle: %s (use the original manifest file, not the model UUID).", commandFor("models acquire MANIFEST_FILE"))
			}
		}
		if acquisition, ok := root["acquisition_work_id"].(string); ok && contracts.ValidID(acquisition) {
			b.line("Acquisition: %s", commandFor("work show "+acquisition))
			b.line("Wait: %s", commandFor("work wait "+acquisition))
			if root["acquisition_state"] == "failed" || root["acquisition_state"] == "interrupted" {
				b.line("Retry: %s", commandFor("work retry "+acquisition))
			}
		}
	}
	if strings.Contains(op, "dataset") {
		switch root["readiness"] {
		case "empty":
			b.line("Next: inspect independent current speaker mappings and recipe filters, then create another dataset.")
		case "ready_for_preparation":
			b.line("Preparation is admitted; usable training clips and acoustic quality are not yet proved.")
		case "invalidated":
			b.line("Next: create a dataset from current evidence and the elected recipe.")
		}
	}
}
func humanWarnings(b *humanBuffer, v any, depth int) {
	if depth > 5 {
		return
	}
	switch x := v.(type) {
	case map[string]any:
		if x["date_unresolved"] == true || x["date_state"] == "ambiguous" || x["date_state"] == "nonexistent" || x["date_state"] == "invalid" {
			b.line("Warning: item %s, media %s, originated date remains %s.", humanValue(x["ordinal"]), humanValue(x["media_id"]), humanValue(x["date_state"]))
			if id, ok := x["media_id"].(string); ok && contracts.ValidID(id) {
				b.line("Inspect: %s", commandFor("media show "+id))
				b.line("Repair: media set-origin %s --revision CURRENT_REVISION --originated-on YYYY-MM-DD (see --help for precision and timezone options).", id)
			}
		}
		if notices, ok := x["notices"].([]any); ok {
			for i, n := range notices {
				if i >= 16 {
					break
				}
				notice := humanValue(n)
				if !strings.Contains(notice, " ") {
					notice = humanLabel(notice)
				}
				b.line("Warning: %s", notice)
			}
		}
		if x["partial"] == true {
			b.line("Warning: accepted output is partially useful; inspect item/date warnings. Unresolved dates alone do not fail accepted media.")
		}
		for k, item := range x {
			if k == "items" || k == "result" {
				humanWarnings(b, item, depth+1)
			}
		}
	case []any:
		if len(x) > 64 {
			route := "use --json for complete item/date warnings"
			if strings.HasPrefix(strings.Join(operatorContext.Path, " "), "work ") && contracts.ValidID(selectedItem()) {
				route = commandFor("work results " + selectedItem())
			}
			b.line("Warning: Additional item details omitted; %s.", route)
		}
		for i, item := range x {
			if i >= 64 {
				break
			}
			humanWarnings(b, item, depth+1)
		}
	}
}
func selectedItem() string {
	if len(operatorContext.Args) < len(operatorContext.Path) {
		return "RECORDING_UUID"
	}
	for _, arg := range operatorContext.Args[len(operatorContext.Path):] {
		if contracts.ValidID(arg) {
			return arg
		}
	}
	return "RECORDING_UUID"
}

// Human text previews exact machine byte pages, whose boundaries may bisect UTF-8.
func humanTextPage(raw []byte) (string, int, int, bool) {
	start := 0
	for start < len(raw) && raw[start]&0xc0 == 0x80 && start < 3 {
		start++
	}
	data := raw[start:]
	end := len(data)
	if !utf8.Valid(data) && end > 0 {
		boundary := end - 1
		for boundary > 0 && data[boundary]&0xc0 == 0x80 {
			boundary--
		}
		if !utf8.FullRune(data[boundary:]) {
			end = boundary
		}
	}
	if !utf8.Valid(data[:end]) {
		return "", 0, 0, false
	}
	return string(data[:end]), start, len(data) - end, true
}

func humanDocuments(b *humanBuffer, v any) {
	root, ok := v.(map[string]any)
	if !ok {
		return
	}
	op := strings.Join(operatorContext.Path, " ")
	if (op == "recordings document" || op == "media capture") && root["encoding"] == "base64" {
		data, ok := root["data"].(string)
		if !ok {
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(data)
		label := "Document"
		if op == "media capture" {
			label = "Capture " + humanValue(root["section"])
		}
		b.line("%s page (bytes %s to %s):", label, humanValue(root["offset"]), humanValue(root["next_offset"]))
		if root["complete"] != true {
			humanPageInput(b, op, root, "page.json")
		}
		if err == nil {
			text, leading, trailing, valid := humanTextPage(decoded)
			if valid {
				if leading > 0 || trailing > 0 {
					b.line("UTF-8 boundary: %d leading and %d trailing bytes belong to split characters; concatenate exact JSON page bytes to reconstruct them.", leading, trailing)
				}
				for i, line := range strings.Split(text, "\n") {
					if i >= 64 {
						b.line("Additional page text omitted; exact page bytes: %s", invocationJSONCommand())
						break
					}
					b.line("%s", line)
				}
			} else {
				b.line("Page contains non-UTF-8 bytes. Exact base64 page: %s", invocationJSONCommand())
			}
		} else {
			b.line("Page could not be decoded. Exact base64 page: %s", invocationJSONCommand())
		}
	}
	if root["document_inline"] == false {
		b.line("Exact document: %s (see --help for digest and page fields).", commandFor("recordings document "+selectedItem()+" --input page.json"))
	}
}
func humanMaterializations(b *humanBuffer, v any, depth int) {
	if depth > 5 {
		return
	}
	switch x := v.(type) {
	case map[string]any:
		if pub, ok := x["publication_id"].(string); ok {
			lease, ok := x["lease_id"].(string)
			if !ok {
				if value, isMap := x["lease"].(map[string]any); isMap {
					lease, ok = value["id"].(string)
				}
			}
			if ok && contracts.ValidID(pub) && contracts.ValidID(lease) {
				b.line("Release: %s", commandFor("artifacts lease-release "+pub+" "+lease))
				b.line("Renew: %s", commandFor("artifacts lease-renew "+pub+" "+lease))
			}
		}
		for _, item := range x {
			switch item.(type) {
			case map[string]any, []any:
				humanMaterializations(b, item, depth+1)
			}
		}
	case []any:
		for i, item := range x {
			if i >= 64 {
				break
			}
			humanMaterializations(b, item, depth+1)
		}
	}
}

// Query metadata must precede details that can exhaust the human display bound.
func humanQueryPaging(b *humanBuffer, v any) {
	if root, ok := v.(map[string]any); ok {
		humanOrdinaryPaging(b, root)
	}
	op := strings.Join(operatorContext.Path, " ")
	if op != "search" && op != "query run" && op != "timeline calendar" && op != "timeline recording" {
		return
	}
	root, ok := v.(map[string]any)
	if !ok {
		return
	}
	if root["total"] == nil {
		return
	}
	b.line("Matching rows: %s; rows outside this page: %s.", humanValue(root["total"]), humanValue(root["omitted"]))
	if cur, ok := root["next_cursor"].(string); ok && cur != "" {
		humanExactField(b, "next_cursor", cur)
		if op == "search" {
			b.line("Continue: repeat this search text and filters with --after NEXT_CURSOR (the complete value above).")
		} else {
			b.line("Continue: set pagination.cursor in the input JSON to the complete cursor above, keeping the query/filter unchanged.")
		}
	}
}
func renderHuman(response contracts.Response) (string, error) {
	v, e := humanJSON(response.Result)
	if e != nil {
		return "", e
	}
	b := &humanBuffer{}
	op := strings.Join(operatorContext.Path, " ")
	if op == "" {
		op = "Command"
	}
	if response.Error != nil {
		b.line("%s: %s", response.Error.Code, response.Error.Message)
		for _, d := range response.Error.Diagnostics {
			b.line("%s: %s", d.Code, d.Message)
		}
		if response.Error.Reason == "wait_timeout" || response.Error.Reason == "wait_interrupted" {
			b.line("Stopping the wait does not cancel durable work.")
		}
		if response.Error.Code == "invalid_request" {
			b.line("Help: %s", commandFor(op+" --help"))
		}
	} else if !operatorContext.Quiet {
		b.line("%s completed.", humanLabel(op))
	}
	if operatorContext.Verbose || operatorContext.Debug {
		if response.RequestID != "" {
			b.line("Request: %s", response.RequestID)
		}
		if response.WorkspaceID != "" {
			b.line("Workspace ID: %s", response.WorkspaceID)
		}
	}
	if !operatorContext.Quiet || response.Error != nil {
		humanSummary(b, v, 0)
		if root, ok := v.(map[string]any); ok {
			humanActions(b, root)
		}
		humanQueryPaging(b, v)
		if root, ok := v.(map[string]any); ok {
			humanRetrieval(b, root)
		}
	}
	if root, ok := v.(map[string]any); ok && (!operatorContext.Quiet || response.Error != nil) {
		if state, ok := root["state"].(string); ok {
			b.line("State: %s", humanLabel(state))
		}
		if result, ok := root["result"].(map[string]any); ok {
			if items, ok := result["items"].([]any); ok {
				admitted, failed, warnings := 0, 0, 0
				for _, raw := range items {
					if item, ok := raw.(map[string]any); ok {
						if item["state"] == "failed" {
							failed++
						} else {
							admitted++
						}
						if item["date_unresolved"] == true {
							warnings++
						}
					}
				}
				b.line("Items: %d admitted, %d failed, %d date warnings.", admitted, failed, warnings)
			}
		}
	}
	humanWarnings(b, v, 0)
	if v == nil && response.Error != nil && strings.HasPrefix(op, "work ") {
		humanActions(b, map[string]any{"id": selectedItem()})
		b.line("No durable state was observed by this invocation.")
	}

	if !operatorContext.Quiet || response.Error != nil {
		humanDocuments(b, v)
		humanMaterializations(b, v, 0)
		if v != nil {
			humanFields(b, v, 0)
		}
	}
	if b.lines >= 128 || b.truncated {
		b.WriteString("Warning: Display limit reached. Use the complete JSON command above for full detail.\n")
	}
	return b.String(), nil
}
