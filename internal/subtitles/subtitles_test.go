// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaExactIdentity(t *testing.T) {
	data := SchemaBytes()
	hash := sha256.Sum256(data)
	if len(data) != 191170 || hex.EncodeToString(hash[:]) != SchemaSHA256 {
		t.Fatal("wrong packaged schema")
	}
	data[0] = 'x'
	if !bytes.HasPrefix(SchemaBytes(), []byte("{")) {
		t.Fatal("mutable packaged resource")
	}
}
func TestAssemblyInwardOverlapAndUntimed(t *testing.T) {
	source := testDocument(t)
	duration := int64(1000000000)
	result, err := AssembleDocument(source, &duration, []Turn{{SpeakerID: "11111111-1111-4111-8111-111111111111", StartNS: 100000001, EndNS: 400999999}, {SpeakerID: "22222222-2222-4222-8222-222222222222", StartNS: 200000000, EndNS: 600000000}}, []Participation{{CueID: "cue-000000", SpeakerID: "11111111-1111-4111-8111-111111111111"}})
	if err != nil {
		t.Fatal(err)
	}
	var doc semanticDocument
	json.Unmarshal(result.Document, &doc)
	a := doc.Cues[0].Attributions
	if len(a) != 3 || *a[0].Start != 101 || *a[0].End != 400 || a[2].Start != nil {
		t.Fatalf("incorrect projection %+v", a)
	}
	var before, after map[string]json.RawMessage
	json.Unmarshal(source, &before)
	json.Unmarshal(result.Document, &after)
	if !bytes.Equal(before["source"], after["source"]) {
		t.Fatal("source envelope changed")
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "timing_rounded" {
		t.Fatalf("missing rounding diagnostics %+v", result.Diagnostics)
	}
	second, err := AssembleDocument(result.Document, nil, []Turn{{SpeakerID: "22222222-2222-4222-8222-222222222222", StartNS: 0, EndNS: 1000000000}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(second.Document, &doc)
	if len(doc.Cues[0].Attributions) != 1 {
		t.Fatal("retained previous assignments")
	}
	if bytes.Contains(second.Document, []byte(`"media_timing"`)) {
		t.Fatal("unknown duration retained previous declaration")
	}
	cues, err := DocumentCues(second.Document)
	if err != nil || len(cues) != 1 || cues[0].ID != "cue-000000" || cues[0].StartMS != 0 || cues[0].EndMS != 1000 {
		t.Fatalf("cue IDs not retained %+v %v", cues, err)
	}
}
func TestAssemblyCollapsedAndInvalidIntervals(t *testing.T) {
	source := testDocument(t)
	duration := int64(1000000000)
	result, err := AssembleDocument(source, &duration, []Turn{{SpeakerID: "11111111-1111-4111-8111-111111111111", StartNS: 500000001, EndNS: 500999999}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var doc semanticDocument
	json.Unmarshal(result.Document, &doc)
	if len(doc.Cues[0].Attributions) != 0 {
		t.Fatal("fabricated collapsed assignment")
	}
	if len(result.Diagnostics) == 0 || result.Diagnostics[0].Code != "timing_collapsed" {
		t.Fatal("missing collapse diagnosis")
	}
	for _, turn := range []Turn{{SpeakerID: "", StartNS: 0, EndNS: 1}, {SpeakerID: "11111111-1111-4111-8111-111111111111", StartNS: -1, EndNS: 1}, {SpeakerID: "11111111-1111-4111-8111-111111111111", StartNS: 2, EndNS: 1}, {SpeakerID: "11111111-1111-4111-8111-111111111111", StartNS: 0, EndNS: 1000000001}} {
		if _, err := AssembleDocument(source, &duration, []Turn{turn}, nil); err == nil {
			t.Fatal("accepted invalid turn")
		}
	}
	if _, err := AssembleDocument(source, nil, nil, []Participation{{CueID: "missing", SpeakerID: "11111111-1111-4111-8111-111111111111"}}); err == nil {
		t.Fatal("accepted missing cue evidence")
	}
}
func TestAssemblyUncoveredAndClippedCoverage(t *testing.T) {
	result, err := AssembleDocument(testDocument(t), nil, []Turn{{SpeakerID: "11111111-1111-4111-8111-111111111111", StartNS: 500000000, EndNS: 1500000000}, {SpeakerID: "22222222-2222-4222-8222-222222222222", StartNS: 2000000000, EndNS: 2500000000}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var doc semanticDocument
	json.Unmarshal(result.Document, &doc)
	if len(doc.Cues[0].Attributions) != 1 || *doc.Cues[0].Attributions[0].End != 1000 {
		t.Fatal("incorrect cue intersection")
	}
	codes := map[string]bool{}
	for _, d := range result.Diagnostics {
		codes[d.Code] = true
	}
	if !codes["timing_clipped"] || !codes["timing_uncovered"] {
		t.Fatalf("missing diagnostics %+v", result.Diagnostics)
	}
}
func nativeDriver(t *testing.T) *Driver {
	t.Helper()
	path := os.Getenv("CUESON_EXECUTABLE")
	if path == "" {
		t.Skip("native pinned Cueson qualification is a separate native check")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	driver, err := New(Tool{Executable: path, ExecutableSHA256: hex.EncodeToString(hash[:])})
	if err != nil {
		t.Fatal(err)
	}
	return driver
}
func TestNativeIngestAssemblyExports(t *testing.T) {
	driver := nativeDriver(t)
	ctx := context.Background()
	source := []byte("1\n00:00:00,000 --> 00:00:01,000\nActual source.\n\n")
	doc, err := driver.Ingest(ctx, source, "srt")
	if err != nil {
		t.Fatal(err)
	}
	duration := int64(1000000000)
	assembled, err := driver.Assemble(ctx, doc, &duration, []Turn{{SpeakerID: "11111111-1111-4111-8111-111111111111", StartNS: 100000000, EndNS: 900000000}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := driver.Export(ctx, assembled.Document, "cueson", false)
	if err != nil || !bytes.Equal(exported.Bytes, assembled.Document) {
		t.Fatalf("Cue JSON export changed document %v", err)
	}
	for _, format := range []string{"srt", "vtt"} {
		exported, err = driver.Export(ctx, assembled.Document, format, false)
		if err != nil || !bytes.Contains(exported.Bytes, []byte("Actual source.")) || len(exported.Diagnostics) < 2 {
			t.Fatalf("native export missing omissions %s %v %+v", format, err, exported.Diagnostics)
		}
		strict, err := driver.Export(ctx, assembled.Document, format, true)
		if err == nil || len(strict.Bytes) != 0 || len(strict.Diagnostics) < 2 {
			t.Fatal("strict published lossy bytes")
		}
	}
	if _, err := driver.Ingest(ctx, nil, "srt"); err == nil {
		t.Fatal("fabricated no-speech document")
	}
	if err := driver.Validate(ctx, assembled.Document); err != nil {
		t.Fatal(err)
	}
	if report, err := driver.Inspect(ctx, assembled.Document); err != nil || !bytes.Contains(report, []byte(`"consumer_annotations"`)) {
		t.Fatal("missing native consumer inspection")
	}
}
func TestNativeRejectsWrongDigestAndChangedExecutable(t *testing.T) {
	driver := nativeDriver(t)
	if _, err := New(Tool{Executable: driver.tool.Executable, ExecutableSHA256: strings.Repeat("0", 64)}); err == nil {
		t.Fatal("wrong executable accepted")
	}
}

func TestNativeLicensedFixtureSubtitles(t *testing.T) {
	driver := nativeDriver(t)
	root := filepath.Join("..", "..", "tests", "fixtures", "media")
	for _, fixture := range []struct {
		name       string
		durationNS int64
	}{{"speech", 10800000000}, {"sintel", 30000000000}} {
		for _, format := range []string{"srt", "vtt"} {
			t.Run(fixture.name+"-"+format, func(t *testing.T) {
				source, err := os.ReadFile(filepath.Join(root, fixture.name+"."+format))
				if err != nil {
					t.Fatal(err)
				}
				doc, err := driver.Ingest(context.Background(), source, format)
				if err != nil {
					t.Fatal(err)
				}
				cues, err := DocumentCues(doc)
				if err != nil {
					t.Fatal(err)
				}
				localID := contracts.ID()
				turns := []Turn{}
				for _, cue := range cues {
					turns = append(turns, Turn{SpeakerID: localID, StartNS: cue.StartMS * nanosecondsPerMillisecond, EndNS: cue.EndMS * nanosecondsPerMillisecond})
				}
				assembled, err := driver.Assemble(context.Background(), doc, &fixture.durationNS, turns, nil)
				if err != nil {
					t.Fatal(err)
				}
				output, err := driver.Export(context.Background(), assembled.Document, format, false)
				if err != nil || len(output.Diagnostics) != len(cues)+1 {
					t.Fatalf("fixture omissions %v %+v", err, output.Diagnostics)
				}
				if err = driver.Validate(context.Background(), assembled.Document); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestAssemblyRejectsBoundedLossAndPreservesRepeatedWarnings(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	data := mutateDocument(t, func(fields map[string]json.RawMessage) {
		var prototype []map[string]json.RawMessage
		json.Unmarshal(fields["cues"], &prototype)
		cues := []map[string]json.RawMessage{}
		assignments := make([]attribution, 1024)
		for i := range assignments {
			assignments[i] = attribution{SpeakerID: id}
		}
		for i := 0; i < 9; i++ {
			var cue map[string]json.RawMessage
			encoded, _ := json.Marshal(prototype[0])
			json.Unmarshal(encoded, &cue)
			cue["id"], _ = json.Marshal("cue-" + integer(i))
			cue["ordinal"], _ = json.Marshal(i)
			cue["source_order"], _ = json.Marshal(i)
			cue["speaker_attributions"], _ = json.Marshal(assignments)
			cues = append(cues, cue)
		}
		fields["cues"], _ = json.Marshal(cues)
		for _, key := range []string{"document", "stats"} {
			var summary map[string]json.RawMessage
			json.Unmarshal(fields[key], &summary)
			summary["cue_count"] = json.RawMessage("9")
			fields[key], _ = json.Marshal(summary)
		}
	})
	if err := ValidateDocument(data); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := omissionDiagnostics(data, false)
	if err == nil || len(diagnostics) != 1 || diagnostics[0].Code != "export_loss_limit" {
		t.Fatal("incomplete omission accounting accepted")
	}
	repeated, err := addUpstreamWarnings(nil, []byte("warning: conversion_font_degraded: First occurrence.\nwarning: conversion_font_degraded: Second occurrence.\n"))
	if err != nil || len(repeated) != 2 {
		t.Fatal("repeated conversion losses collapsed")
	}
}
func TestNativeRejectsChangedBinaryAndCancellation(t *testing.T) {
	original := nativeDriver(t)
	content, err := os.ReadFile(original.tool.Executable)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), filepath.Base(original.tool.Executable))
	if err = os.WriteFile(destination, content, 0700); err != nil {
		t.Fatal(err)
	}
	driver, err := New(Tool{Executable: destination, ExecutableSHA256: original.tool.ExecutableSHA256})
	if err != nil {
		t.Fatal(err)
	}
	content[0] ^= 1
	if err = os.WriteFile(destination, content, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = driver.Ingest(context.Background(), []byte("1\n00:00:00,000 --> 00:00:01,000\nSource.\n\n"), "srt"); err == nil {
		t.Fatal("changed executable ran")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = original.Ingest(ctx, []byte("1\n00:00:00,000 --> 00:00:01,000\nSource.\n\n"), "srt"); err == nil {
		t.Fatal("cancelled ingest ran")
	}
}
func TestNativeEmptyCueSetReturnsNoDocument(t *testing.T) {
	driver := nativeDriver(t)
	source := []byte("[Script Info]\nScriptType: v4.00+\n[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\nStyle: Default,Arial,20,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,0,2,10,10,10,1\n[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\nComment: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,Comment only\n")
	doc, err := driver.Ingest(context.Background(), source, "ass")
	if err != nil || len(doc) != 0 {
		t.Fatalf("empty cue sequence fabricated document: %v %s", err, doc)
	}
}
