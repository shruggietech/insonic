// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
)

func TestDomainImportFlagsAndManifestPrecedence(t *testing.T) {
	operation, id, data, e := parseDomain([]string{"media", "import", "one.wav", "two.mp4", "--reference", "--originated-on", "2026-10-07", "--timezone", "UTC"})
	if e != nil || operation != "media.import" || id != "" {
		t.Fatal(operation, id, e)
	}
	var req library.ImportRequest
	if json.Unmarshal(data, &req) != nil || len(req.Items) != 2 || req.Defaults.Copy == nil || *req.Defaults.Copy || req.Defaults.OriginatedOn != "2026-10-07" {
		t.Fatal(string(data))
	}
	if !filepath.IsAbs(req.Items[0].Source) {
		t.Fatal("relative CLI source")
	}
	dir := t.TempDir()
	manifest := filepath.Join(dir, "inputs.json")
	if e = os.WriteFile(manifest, []byte(`{"kind":"import-manifest","schema_version":"0.0.0","items":[{"source":"fixture.wav","originated_on":"2025-01-02"}]}`), 0600); e != nil {
		t.Fatal(e)
	}
	_, _, data, e = parseDomain([]string{"media", "import", "--manifest", manifest, "--originated-on", "2026-10-07", "--timezone", "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(data, &req)
	if req.Items[0].OriginatedOn != "2025-01-02" || req.Defaults.OriginatedOn != "2026-10-07" || req.Items[0].Source != filepath.Join(dir, "fixture.wav") {
		t.Fatal(string(data))
	}
}
func TestDomainRevisionAndFlagValidation(t *testing.T) {
	id := contracts.ID()
	for _, args := range [][]string{{"media", "set-origin", id, "--originated-on", "2026-10-07"}, {"media", "relocate", id, "file.wav", "--revision", "0"}, {"media", "import", "f.wav", "--copy", "--reference"}, {"media", "import", "f.wav", "--originated-on", "2026-10-07", "--originated-at", "2026-10-07T00:00:00Z"}, {"media", "show", id, "--copy"}, {"work", "show", "not-a-uuid"}} {
		if _, _, _, e := parseDomain(args); e == nil {
			t.Fatal("invalid command admitted", args)
		}
	}
	operation, item, data, e := parseDomain([]string{"media", "set-origin", id, "--revision", "2", "--originated-on", "2026-10-07", "--timezone", "UTC"})
	if e != nil || operation != "media.set-origin" || item != id {
		t.Fatal(operation, item, e)
	}
	var payload struct {
		Revision int64           `json:"revision"`
		Options  library.Options `json:"options"`
	}
	json.Unmarshal(data, &payload)
	if payload.Revision != 2 || payload.Options.OriginatedOn != "2026-10-07" {
		t.Fatal(string(data))
	}
	for _, args := range [][]string{{"models", "base", "list"}, {"work", "list"}, {"media", "list"}} {
		if _, _, data, e := parseDomain(args); e != nil || len(data) != 0 {
			t.Fatal(args, e)
		}
	}
}
func TestDomainPaginationAndWorkResults(t *testing.T) {
	id := contracts.ID()
	for _, group := range []string{"media", "models", "work"} {
		op, item, raw, e := parseDomain([]string{group, "list", "--after", id, "--limit", "25"})
		var page struct {
			AfterID string `json:"after_id"`
			Limit   int    `json:"limit"`
		}
		if e != nil || op != group+".list" || item != "" || json.Unmarshal(raw, &page) != nil || page.AfterID != id || page.Limit != 25 {
			t.Fatal(op, item, string(raw), e)
		}
	}
	op, item, raw, e := parseDomain([]string{"work", "results", id, "--after-ordinal", "42", "--limit", "100"})
	var results struct {
		AfterOrdinal int64 `json:"after_ordinal"`
		Limit        int   `json:"limit"`
	}
	if e != nil || op != "work.results" || item != id || json.Unmarshal(raw, &results) != nil || results.AfterOrdinal != 42 || results.Limit != 100 {
		t.Fatal(op, item, string(raw), e)
	}
	for _, args := range [][]string{
		{"media", "list", "--after", "invalid"},
		{"work", "list", "--limit", "0"},
		{"models", "list", "--limit", "101"},
		{"media", "show", id, "--limit", "1"},
		{"work", "results", id, "--after-ordinal", "-1"},
		{"work", "results", id, "--after", id},
		{"media", "list", "--after-ordinal", "0"},
		{"work", "results", "invalid"},
	} {
		if _, _, _, e = parseDomain(args); e == nil {
			t.Fatal("invalid pagination admitted", args)
		}
	}
}

func TestTranscriptDomainUsesSharedImportContract(t *testing.T) {
	id := contracts.ID()
	op, item, raw, err := parseDomain([]string{"transcript", "import", "caption.ass", "--record", id, "--replace-transcript", "--attribution", "native", "--transcript-max-bytes", "4096", "--transcript-timeout-ms", "30000"})
	if err != nil || op != "media.import" || item != "" {
		t.Fatal(op, item, err)
	}
	var request library.ImportRequest
	if json.Unmarshal(raw, &request) != nil || request.Items[0].Record != id || !filepath.IsAbs(request.Items[0].Transcript) || request.Items[0].Source != "" || request.Defaults.Attribution != "native" || !*request.Defaults.ReplaceTranscript || *request.Defaults.TranscriptMaxBytes != 4096 {
		t.Fatal(string(raw))
	}
	if _, _, _, err = parseDomain([]string{"media", "import", "file.wav", "--subtitle", "one.srt", "--transcript", "two.vtt"}); err == nil {
		t.Fatal("two explicit transcript choices accepted")
	}
}
