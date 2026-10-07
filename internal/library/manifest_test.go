// SPDX-License-Identifier: Apache-2.0
package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestsResolveRelativeAndOverrideDateKind(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "batch.json")
	data := `{"kind":"import-manifest","schema_version":"0.0.0","defaults":{"originated_at":"2026-10-07T12:00:00","timezone":"UTC"},"items":[{"source":"a.wav","originated_on":"2026-10-06","subtitle":"a.srt"}]}`
	os.WriteFile(file, []byte(data), 0600)
	r, e := ReadManifest(file)
	if e != nil {
		t.Fatal(e)
	}
	item := r.Items[0]
	if item.Source != filepath.Join(dir, "a.wav") || item.Subtitle != filepath.Join(dir, "a.srt") {
		t.Fatalf("paths %+v", item)
	}
	options := merged(r.Defaults, item.Options)
	if options.OriginatedAt != "" || options.OriginatedOn != "2026-10-06" {
		t.Fatalf("override %+v", options)
	}
	csv := filepath.Join(dir, "batch.csv")
	os.WriteFile(csv, []byte("source,copy,originated_on,timezone\na.wav,false,2026-10-07,UTC\n"), 0600)
	r, e = ReadManifest(csv)
	if e != nil || r.Items[0].Copy == nil || *r.Items[0].Copy {
		t.Fatalf("CSV %+v %v", r, e)
	}
	os.WriteFile(file, []byte(`{"items":[{"source":"a.wav","secret":"bad"}]}`), 0600)
	if _, e = ReadManifest(file); e == nil {
		t.Fatal("credential accepted")
	}
	os.WriteFile(csv, []byte("source,source\na.wav,b.wav\n"), 0600)
	if _, e = ReadManifest(csv); e == nil {
		t.Fatal("ambiguous duplicate CSV accepted")
	}
}
