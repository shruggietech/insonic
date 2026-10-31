// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestNativeRepairPreservesUnaffectedFieldsAndSource(t *testing.T) {
	driver := nativeDriver(t)
	ctx := context.Background()
	for _, format := range []string{"srt", "vtt", "ass", "ssa"} {
		t.Run(format, func(t *testing.T) {
			original, e := driver.Ingest(ctx, []byte("1\n00:00:00,000 --> 00:00:01,000\nOld target.\n\n2\n00:00:04,000 --> 00:00:05,000\nUntouched words.\n\n"), "srt")
			if e != nil {
				t.Fatal(e)
			}
			if format != "srt" {
				native, e := driver.Export(ctx, original, format, false)
				if e != nil {
					t.Fatal(e)
				}
				original, e = driver.Ingest(ctx, native.Bytes, format)
				if e != nil {
					t.Fatal(e)
				}
			}
			candidate, e := driver.Ingest(ctx, []byte("1\n00:00:00,000 --> 00:00:01,000\nRepaired target.\n\n"), "srt")
			if e != nil {
				t.Fatal(e)
			}
			oldCues, _ := DocumentCues(original)
			newCues, _ := DocumentCues(candidate)
			merged, e := MergeRepairDocument(ctx, driver, original, candidate, []string{oldCues[0].ID}, []string{newCues[0].ID}, "repair-fixture")
			if e != nil {
				t.Fatal(e)
			}
			if e = driver.Validate(ctx, merged.Document); e != nil {
				t.Fatal(e)
			}
			var oldFields, newFields map[string]json.RawMessage
			json.Unmarshal(original, &oldFields)
			json.Unmarshal(merged.Document, &newFields)
			var oldRows, newRows []map[string]json.RawMessage
			json.Unmarshal(oldFields["cues"], &oldRows)
			json.Unmarshal(newFields["cues"], &newRows)
			delete(oldRows[1], "ordinal")
			delete(newRows[0], "ordinal")
			if !reflect.DeepEqual(oldRows[1], newRows[0]) {
				t.Fatal("unaffected semantic cue changed")
			}
			if string(oldFields["source"]) != string(newFields["source"]) {
				t.Fatal("canonical native source changed")
			}
			if len(newRows) != 2 || len(merged.CueIDs) != 1 {
				t.Fatal("wrong cue merge")
			}
		})
	}
}
