// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"sort"
	"strconv"
	"strings"
)

type RepairCodec interface {
	Ingest(context.Context, []byte, string) (json.RawMessage, error)
	Export(context.Context, []byte, string, bool) (Export, error)
}
type RepairMerge struct {
	Document json.RawMessage
	CueIDs   map[string]string
}
type repairObject map[string]json.RawMessage

func repairJSON(v any) json.RawMessage        { raw, _ := json.Marshal(v); return raw }
func repairString(raw json.RawMessage) string { var s string; json.Unmarshal(raw, &s); return s }
func repairInt(raw json.RawMessage) int64     { var n int64; json.Unmarshal(raw, &n); return n }
func repairObjectFrom(raw json.RawMessage) repairObject {
	var v repairObject
	json.Unmarshal(raw, &v)
	return v
}
func repairObjects(raw json.RawMessage) []repairObject {
	var v []repairObject
	json.Unmarshal(raw, &v)
	return v
}
func repairSetField(owner repairObject, name, value string) {
	fields := repairObjects(owner["fields"])
	for _, f := range fields {
		if strings.EqualFold(strings.TrimSpace(repairString(f["field_name"])), name) {
			f["raw_value"] = repairJSON(value)
			delete(f, "typed_value")
		}
	}
	owner["fields"] = repairJSON(fields)
}
func repairRemap(raw json.RawMessage, prefix string, cueIDs map[string]string) json.RawMessage {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.Decode(&value)
	var visit func(any)
	visit = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, item := range x {
				switch k {
				case "id", "cue_id":
					if s, ok := item.(string); ok {
						if renamed, found := cueIDs[s]; found {
							x[k] = renamed
						}
					}
				case "event_id", "source_event_id", "style_id", "record_id", "section_id", "declaration_id":
					if s, ok := item.(string); ok {
						x[k] = prefix + "-" + s
					}
				default:
					visit(item)
				}
			}
		case []any:
			for _, item := range x {
				visit(item)
			}
		}
	}
	visit(value)
	return repairJSON(value)
}

// MergeRepairDocument converts only candidate bytes to the accepted native
// format, preserves untouched cue/native observations, and validates the final
// native document before it can cross the current-recording acceptance boundary.
func MergeRepairDocument(ctx context.Context, codec RepairCodec, previous, candidate []byte, removedIDs, selectedIDs []string, prefix string) (RepairMerge, error) {
	out := RepairMerge{CueIDs: map[string]string{}}
	if ValidateDocument(previous) != nil || ValidateDocument(candidate) != nil || !contracts.ValidTextLength(prefix, 1, 96) || strings.IndexFunc(prefix, func(r rune) bool { return r <= 32 || r == 47 || r == 92 }) >= 0 || len(removedIDs) > 65536 || len(selectedIDs) > 65536 {
		return out, contracts.Fail("invalid_request")
	}
	old, newFields := repairObjectFrom(previous), repairObjectFrom(candidate)
	format := repairString(old["format"])
	originalNewCues := repairObjects(newFields["cues"])
	if repairString(newFields["format"]) != format {
		exported, e := codec.Export(ctx, candidate, format, false)
		if e != nil {
			return out, e
		}
		converted, e := codec.Ingest(ctx, exported.Bytes, format)
		if e != nil {
			return out, e
		}
		newFields = repairObjectFrom(converted)
	}
	convertedCues := repairObjects(newFields["cues"])
	if len(convertedCues) != len(originalNewCues) {
		return out, contracts.Fail("invalid_timing")
	}
	selected, removed := map[string]bool{}, map[string]bool{}
	for _, id := range selectedIDs {
		if selected[id] {
			return out, contracts.Fail("invalid_request")
		}
		selected[id] = true
	}
	for _, id := range removedIDs {
		if removed[id] {
			return out, contracts.Fail("invalid_request")
		}
		removed[id] = true
	}
	added := []repairObject{}
	nativeCueIDs := map[string]string{}
	for i, c := range convertedCues {
		original := originalNewCues[i]
		id := repairString(original["id"])
		if !selected[id] {
			continue
		}
		newID := prefix + "-cue-" + strconv.Itoa(i)
		nativeCueIDs[repairString(c["id"])] = newID
		out.CueIDs[id] = newID
		c["timing"] = original["timing"]
		if assignments, ok := original["speaker_attributions"]; ok {
			c["speaker_attributions"] = assignments
		} else {
			delete(c, "speaker_attributions")
		}
		added = append(added, c)
	}
	if len(added) != len(selected) {
		return out, contracts.Fail("invalid_request")
	}
	oldCues := repairObjects(old["cues"])
	kept := []repairObject{}
	foundRemoved := 0
	nextOrder := int64(0)
	for _, c := range oldCues {
		nextOrder = max(nextOrder, repairInt(c["source_order"])+1)
		if removed[repairString(c["id"])] {
			foundRemoved++
			continue
		}
		kept = append(kept, c)
	}
	if foundRemoved != len(removed) {
		return out, contracts.Fail("invalid_request")
	}
	if format == "ass" || format == "ssa" {
		if e := mergeScriptedRepair(old, newFields, oldCues, added, removed, prefix, nativeCueIDs, format); e != nil {
			return out, e
		}
	} else {
		if format == "webvtt" {
			fd := repairObjectFrom(old["format_data"])
			native := repairObjectFrom(fd["webvtt"])
			blocks := repairObjects(native["blocks"])
			for _, b := range blocks {
				nextOrder = max(nextOrder, repairInt(b["source_order"])+1)
			}
			for _, c := range oldCues {
				if removed[repairString(c["id"])] {
					blocks = append(blocks, repairObject{"type": repairJSON("note"), "source_order": c["source_order"], "raw": repairJSON("NOTE repaired cue"), "raw_lines": repairJSON([]string{"NOTE repaired cue"})})
				}
			}
			sort.Slice(blocks, func(i, j int) bool {
				return repairInt(blocks[i]["source_order"]) < repairInt(blocks[j]["source_order"])
			})
			native["blocks"] = repairJSON(blocks)
			fd["webvtt"] = repairJSON(native)
			old["format_data"] = repairJSON(fd)
		}
		for _, c := range added {
			c["source_order"] = repairJSON(nextOrder)
			nextOrder++
		}
	}
	for _, c := range added {
		c = repairObjectFrom(repairRemap(repairJSON(c), prefix, nativeCueIDs))
		kept = append(kept, c)
	}
	if len(kept) == 0 || len(kept) > 65536 {
		return out, contracts.Fail("output_limit")
	}
	sort.SliceStable(kept, func(i, j int) bool { return repairInt(kept[i]["source_order"]) < repairInt(kept[j]["source_order"]) })
	minimum, maximum := int64(0), int64(0)
	words := false
	for i, c := range kept {
		c["ordinal"] = repairJSON(i)
		timing := repairObjectFrom(c["timing"])
		start, end := repairInt(timing["start_milliseconds"]), repairInt(timing["end_milliseconds"])
		if i == 0 {
			minimum, maximum = start, end
		} else {
			minimum = min(minimum, start)
			maximum = max(maximum, end)
		}
		var tokens []json.RawMessage
		json.Unmarshal(c["tokens"], &tokens)
		words = words || len(tokens) > 0
	}
	old["cues"] = repairJSON(kept)
	diagnostics := []repairObject{}
	warnings, failures := 0, 0
	for _, d := range repairObjects(old["diagnostics"]) {
		if removed[repairString(d["cue_id"])] {
			continue
		}
		diagnostics = append(diagnostics, d)
		switch repairString(d["severity"]) {
		case "warning":
			warnings++
		case "error":
			failures++
		}
	}
	old["diagnostics"] = repairJSON(diagnostics)
	for _, key := range []string{"document", "stats"} {
		summary := repairObjectFrom(old[key])
		summary["cue_count"] = repairJSON(len(kept))
		summary["media_span_milliseconds"] = repairJSON(maximum - minimum)
		summary["has_word_level_timing"] = repairJSON(words)
		if key == "document" {
			summary["media_start_milliseconds"] = repairJSON(minimum)
			summary["media_end_milliseconds"] = repairJSON(maximum)
		} else {
			summary["diagnostic_count"] = repairJSON(len(diagnostics))
			summary["warning_count"] = repairJSON(warnings)
			summary["error_count"] = repairJSON(failures)
		}
		old[key] = repairJSON(summary)
	}
	out.Document = repairJSON(old)
	if e := ValidateDocument(out.Document); e != nil {
		return RepairMerge{}, e
	}
	if _, e := codec.Export(ctx, out.Document, "cueson", false); e != nil {
		return RepairMerge{}, e
	}
	return out, nil
}

func mergeScriptedRepair(old, candidate repairObject, oldCues, added []repairObject, removed map[string]bool, prefix string, cueIDs map[string]string, format string) error {
	oldFD, newFD := repairObjectFrom(old["format_data"]), repairObjectFrom(candidate["format_data"])
	native, newNative := repairObjectFrom(oldFD[format]), repairObjectFrom(newFD[format])
	if native == nil || newNative == nil {
		return contracts.Fail("invalid_request")
	}
	sections, records, styles, events := repairObjects(native["sections"]), repairObjects(native["records"]), repairObjects(native["styles"]), repairObjects(native["events"])
	for _, event := range events {
		if removed[repairString(event["cue_id"])] {
			event["event_type"] = repairJSON("comment")
			delete(event, "cue_id")
			event["text"] = repairJSON("")
			repairSetField(event, "Text", "")
			for _, field := range []string{"spans", "tags", "karaoke"} {
				event[field] = json.RawMessage("[]")
			}
		}
	}
	selectedEvents, selectedStyles := map[string]bool{}, map[string]bool{}
	wrap := json.RawMessage("0")
	if len(oldCues) > 0 {
		fd := repairObjectFrom(oldCues[0]["format_data"])
		data := repairObjectFrom(fd[format])
		projection := repairObjectFrom(data["projection"])
		if v, ok := projection["wrap_style"]; ok {
			wrap = v
		}
	}
	for _, c := range added {
		fd := repairObjectFrom(c["format_data"])
		data := repairObjectFrom(fd[format])
		selectedEvents[repairString(data["event_id"])] = true
		selectedStyles[repairString(data["style_id"])] = true
		delete(data, "start_timestamp_raw")
		delete(data, "end_timestamp_raw")
		projection := repairObjectFrom(data["projection"])
		projection["wrap_style"] = wrap
		data["projection"] = repairJSON(projection)
		fd[format] = repairJSON(data)
		c["format_data"] = repairJSON(fd)
	}
	requiredRecords := map[string]bool{}
	newEvents, newStyles := []repairObject{}, []repairObject{}
	styleNames := map[string]string{}
	for _, s := range repairObjects(newNative["styles"]) {
		if !selectedStyles[repairString(s["style_id"])] {
			continue
		}
		name := repairString(s["name"])
		replacement := prefix + "-" + name
		styleNames[name] = replacement
		s["name"] = repairJSON(replacement)
		repairSetField(s, "Name", replacement)
		requiredRecords[repairString(s["record_id"])] = true
		if id := repairString(s["declaration_id"]); id != "" {
			requiredRecords[id] = true
		}
		newStyles = append(newStyles, s)
	}
	for _, event := range repairObjects(newNative["events"]) {
		if !selectedEvents[repairString(event["event_id"])] {
			continue
		}
		for _, f := range repairObjects(event["fields"]) {
			if strings.EqualFold(strings.TrimSpace(repairString(f["field_name"])), "Style") {
				name := repairString(f["raw_value"])
				replacement, ok := styleNames[name]
				if !ok {
					return contracts.Fail("invalid_request")
				}
				repairSetField(event, "Style", replacement)
			}
		}
		requiredRecords[repairString(event["record_id"])] = true
		if id := repairString(event["declaration_id"]); id != "" {
			requiredRecords[id] = true
		}
		newEvents = append(newEvents, event)
	}
	requiredSections := map[string]bool{}
	newRecords := []repairObject{}
	for _, record := range repairObjects(newNative["records"]) {
		if requiredRecords[repairString(record["record_id"])] {
			requiredSections[repairString(record["section_id"])] = true
			delete(record, "raw_line")
			newRecords = append(newRecords, record)
		}
	}
	newSections := []repairObject{}
	for _, section := range repairObjects(newNative["sections"]) {
		if requiredSections[repairString(section["section_id"])] {
			delete(section, "raw_header")
			newSections = append(newSections, section)
		}
	}
	combined := append(append([]repairObject{}, newSections...), newRecords...)
	sort.Slice(combined, func(i, j int) bool {
		return repairInt(combined[i]["source_order"]) < repairInt(combined[j]["source_order"])
	})
	next := len(sections) + len(records)
	if next+len(combined) > 65536 {
		return contracts.Fail("output_limit")
	}
	recordOrders := map[string]int{}
	for _, item := range combined {
		item["source_order"] = repairJSON(next)
		if id := repairString(item["record_id"]); id != "" {
			recordOrders[id] = next
		}
		next++
	}
	eventRecords := map[string]string{}
	for _, event := range newEvents {
		eventRecords[repairString(event["event_id"])] = repairString(event["record_id"])
	}
	for _, cue := range added {
		fd := repairObjectFrom(cue["format_data"])
		data := repairObjectFrom(fd[format])
		recordID, ok := eventRecords[repairString(data["event_id"])]
		if !ok {
			return contracts.Fail("invalid_request")
		}
		order, ok := recordOrders[recordID]
		if !ok {
			return contracts.Fail("invalid_request")
		}
		cue["source_order"] = repairJSON(order)
	}
	appendRemapped := func(existing, add []repairObject) []repairObject {
		for _, item := range add {
			existing = append(existing, repairObjectFrom(repairRemap(repairJSON(item), prefix, cueIDs)))
		}
		return existing
	}
	native["sections"] = repairJSON(appendRemapped(sections, newSections))
	native["records"] = repairJSON(appendRemapped(records, newRecords))
	native["styles"] = repairJSON(appendRemapped(styles, newStyles))
	native["events"] = repairJSON(appendRemapped(events, newEvents))
	oldFD[format] = repairJSON(native)
	old["format_data"] = repairJSON(oldFD)
	return nil
}
