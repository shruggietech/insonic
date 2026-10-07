// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
)

func wavFixture(t *testing.T) string {
	t.Helper()
	data := make([]byte, 44+960)
	copy(data[:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	copy(data[8:16], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:20], 16)
	binary.LittleEndian.PutUint16(data[20:22], 1)
	binary.LittleEndian.PutUint16(data[22:24], 1)
	binary.LittleEndian.PutUint32(data[24:28], 48000)
	binary.LittleEndian.PutUint32(data[28:32], 96000)
	binary.LittleEndian.PutUint16(data[32:34], 2)
	binary.LittleEndian.PutUint16(data[34:36], 16)
	copy(data[36:40], "data")
	binary.LittleEndian.PutUint32(data[40:44], 960)
	path := filepath.Join(t.TempDir(), "original.wav")
	if e := os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	return path
}
func nativeTools(t *testing.T) Tools {
	t.Helper()
	path := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE")
	if path == "" {
		t.Skip("exact native media tools fixture not selected")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var tools Tools
	if strict(data, &tools) != nil {
		t.Fatal("invalid tools fixture")
	}
	return tools
}
func TestNativeMediaExtractionAndCurrentReplacement(t *testing.T) {
	tools := nativeTools(t)
	s, db := libraryFixture(t)
	s.Tools = tools
	source := wavFixture(t)
	ctx := context.Background()
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: source}}})
	value, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	result := value.(ImportResult)
	if result.Items[0].State != "admitted" || result.Items[0].CaptureState != "captured" {
		if result.Items[0].MediaID != "" {
			metadata, _ := s.Metadata(ctx, result.Items[0].MediaID)
			for _, report := range metadata.Reports {
				t.Logf("%s state=%s exit=%d stdout=%s stderr=%s", report.Tool, report.State, report.ExitCode, report.Stdout, report.Stderr)
			}
		}
		t.Fatalf("native capture %+v", result)
	}
	entry, e := db.Library(ctx, result.Items[0].MediaID)
	if e != nil {
		t.Fatal(e)
	}
	if entry.DurationUS == nil || *entry.DurationUS != 10000 {
		t.Fatalf("duration %+v", entry.DurationUS)
	}
	var facts Facts
	if json.Unmarshal(entry.Facts, &facts) != nil || len(facts.Streams) != 1 || facts.Streams[0].TimeBase != "1/48000" || facts.Streams[0].Channels == nil || *facts.Streams[0].Channels != 1 {
		t.Fatalf("stream %+v", facts)
	}
	var metadata Metadata
	json.Unmarshal(entry.Metadata, &metadata)
	if len(metadata.Reports) != 2 || len(metadata.Reports[0].Stdout) == 0 || len(metadata.Reports[1].Stdout) == 0 {
		t.Fatal("raw reports missing")
	}
	oldIDs, _ := catalog.PublicationIDs(entry.ReportPublicationIDs)
	work = claimLibraryWork(t, db, "media.refresh", RefreshRequest{MediaID: entry.ID})
	if _, e = s.Execute(ctx, work); e != nil {
		t.Fatal(e)
	}
	for _, id := range oldIDs {
		p, e := db.Publication(ctx, id)
		if e != nil || p.State != "retired" {
			t.Fatal("old report bytes retained")
		}
	}
	if e = os.Remove(source); e != nil {
		t.Fatal(e)
	}
	view, e := s.Show(ctx, entry.ID)
	if e != nil || view.Availability != "available" {
		t.Fatal("managed copy lost with source")
	}
}
