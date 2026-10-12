// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/schemas"
)

func TestHistoricalManifestAdmittedIntoCurrentRuntimeRequest(t *testing.T) {
	file := filepath.Join(t.TempDir(), "historical.json")
	original := []byte(`{"kind":"import-manifest","schema_version":"1.0.0","defaults":{"originated_at":"2020-01-01T00:00:00.123456789Z"},"items":[{"source":"fixture.wav"}]}`)
	if e := os.WriteFile(file, original, 0600); e != nil {
		t.Fatal(e)
	}
	op, item, data, e := parseDomain([]string{"media", "import", "--manifest", file})
	if e != nil {
		t.Fatal(e)
	}
	raw, e := json.Marshal(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: op, ItemID: item, Data: data})
	if e != nil {
		t.Fatal(e)
	}
	if e = schemas.ValidateRequest(raw); e != nil {
		t.Fatal("historical file rejected at IPC admission", e)
	}
	var request library.ImportRequest
	if e = json.Unmarshal(data, &request); e != nil || request.Version != contracts.Version || request.Defaults.OriginatedAt != "2020-01-01T00:00:00.123456789Z" {
		t.Fatal("request normalization", request, e)
	}
	after, e := os.ReadFile(file)
	if e != nil || string(after) != string(original) {
		t.Fatal("historical source rewritten", e)
	}
}

func TestHistoricalModelManifestKeepsDigestThroughCurrentAdmission(t *testing.T) {
	m := models.Manifest{Kind: "base-model-manifest", Version: "1.0.0", Name: "historical", ModelVersion: "1", Revision: "immutable", License: "fixture", Capabilities: []string{"transcription"}, Files: []models.File{{Role: "weights", Size: 1, SHA256: strings.Repeat("a", 64), URL: "https://example.org/weights"}}}
	original, e := json.Marshal(m)
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(t.TempDir(), "historical-model.json")
	if e = os.WriteFile(file, original, 0600); e != nil {
		t.Fatal(e)
	}
	for _, action := range []string{"register", "acquire"} {
		op, item, data, e := parseDomain([]string{"models", action, file})
		if e != nil {
			t.Fatal(e)
		}
		raw, e := json.Marshal(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: op, ItemID: item, Data: data})
		if e != nil {
			t.Fatal(e)
		}
		if e = schemas.ValidateRequest(raw); e != nil {
			t.Fatal("historical model rejected at IPC admission", e)
		}
		var request models.Request
		if e = json.Unmarshal(data, &request); e != nil || request.Manifest.Digest() != m.Digest() {
			t.Fatal("immutable model manifest changed", e)
		}
	}
	after, e := os.ReadFile(file)
	if e != nil || string(after) != string(original) {
		t.Fatal("historical model source rewritten", e)
	}
}

func TestDomainImportFlagsAndManifestPrecedence(t *testing.T) {
	operation, id, data, e := parseDomain([]string{"media", "import", "one.wav", "two.mp4", "--reference", "--originated-on", "2026-10-07", "--timezone", "UTC"})
	if e != nil || operation != "media.import" || id != "" {
		t.Fatal(operation, id, e)
	}
	var req library.ImportRequest
	if json.Unmarshal(data, &req) != nil || len(req.Items) != 2 || req.Overrides.Copy == nil || *req.Overrides.Copy || req.Overrides.OriginatedOn != "2026-10-07" {
		t.Fatal(string(data))
	}
	if !filepath.IsAbs(req.Items[0].Source) {
		t.Fatal("relative CLI source")
	}
	dir := t.TempDir()
	manifest := filepath.Join(dir, "inputs.json")
	if e = os.WriteFile(manifest, []byte(`{"kind":"import-manifest","schema_version":"1.0.1","items":[{"source":"fixture.wav","originated_on":"2025-01-02"}]}`), 0600); e != nil {
		t.Fatal(e)
	}
	_, _, data, e = parseDomain([]string{"media", "import", "--manifest", manifest, "--originated-on", "2026-10-07", "--timezone", "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(data, &req)
	if req.Items[0].OriginatedOn != "2025-01-02" || req.Overrides.OriginatedOn != "2026-10-07" || req.Items[0].Source != filepath.Join(dir, "fixture.wav") {
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

func TestTranscriptManifestCredentialAndAdapterPrecedence(t *testing.T) {
	cliCredential := contracts.ID()
	itemCredential := contracts.ID()
	manifest := filepath.Join(t.TempDir(), "transcripts.json")
	request := library.ImportRequest{Kind: "import-manifest", Version: contracts.Version, Items: []library.Item{{Record: contracts.ID(), Transcript: "https://captions.invalid/a.vtt"}, {Record: contracts.ID(), Transcript: "https://captions.invalid/b.vtt", TranscriptCredentialID: itemCredential, TranscriptAdapter: "custom"}}}
	data, _ := json.Marshal(request)
	if err := os.WriteFile(manifest, data, 0600); err != nil {
		t.Fatal(err)
	}
	op, _, raw, err := parseDomain([]string{"transcript", "import", "--manifest", manifest, "--transcript-credential-id", cliCredential, "--transcript-adapter", "https"})
	if err != nil || op != "media.import" {
		t.Fatal(op, err)
	}
	json.Unmarshal(raw, &request)
	if request.Items[0].TranscriptCredentialID != cliCredential || request.Items[0].TranscriptAdapter != "https" || request.Items[1].TranscriptCredentialID != itemCredential || request.Items[1].TranscriptAdapter != "custom" {
		t.Fatal("CLI fallback or explicit item election lost", string(raw))
	}
}
