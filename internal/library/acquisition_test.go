// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

type acquisitionSecret struct{ resolved []byte }

func (s *acquisitionSecret) Resolve(context.Context, string) ([]byte, error) {
	s.resolved = []byte("fixture-access")
	return s.resolved, nil
}
func (*acquisitionSecret) Status(context.Context, string) (string, error) { return "available", nil }

func TestConfiguredAcquisitionCapturesBytesAndSuppliedSubtitle(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	fixture := wavFixture(t)
	data, e := os.ReadFile(fixture)
	if e != nil {
		t.Fatal(e)
	}
	authorized := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorized = r.Header.Get("Authorization") == "Bearer fixture-access"
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("ETag", "fixture-revision")
		w.Write(data)
	}))
	defer server.Close()
	secret := &acquisitionSecret{}
	s.AcquisitionAdapters["http"] = httpAcquisition{Secrets: secret}
	subtitle := filepath.Join(t.TempDir(), "supplied.srt")
	if e = os.WriteFile(subtitle, []byte("1\n00:00:00,000 --> 00:00:00,010\nFixture\n"), 0600); e != nil {
		t.Fatal(e)
	}
	electedLocalHTTP := true
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", LocalHTTP: &electedLocalHTTP}, Items: []Item{{Source: server.URL + "/original.wav?id=fixture&page=1", Subtitle: subtitle, CredentialID: contracts.ID()}}})
	value, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	item := value.(ImportResult).Items[0]
	if item.State != "admitted" || item.SubtitleState != "available" || !authorized {
		t.Fatal("configured acquisition/subtitle did not complete")
	}
	if !bytes.Equal(secret.resolved, make([]byte, len(secret.resolved))) {
		t.Fatal("private returned bytes not zeroed")
	}
	entry, e := db.Library(ctx, item.MediaID)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(marshal(entry), []byte("fixture-access")) || bytes.Contains(work.Payload, []byte("fixture-access")) {
		t.Fatal("secret escaped private boundary")
	}
	var facts Facts
	if strict(entry.Facts, &facts) != nil || facts.Acquisition == nil || facts.Acquisition.DeclaredMIME != "audio/wav" || facts.FilesystemModified != "" {
		t.Fatal("acquisition facts lost or scratch mtime misrepresented")
	}
	if entry.OriginalPublicationID == nil || entry.SubtitlePublicationID == nil {
		t.Fatal("supplied bytes not published")
	}
	for _, id := range []string{*entry.OriginalPublicationID, *entry.SubtitlePublicationID} {
		if e = s.Artifacts.Verify(ctx, id); e != nil {
			t.Fatal(e)
		}
	}
}

func TestPrepareImportRejectsCredentialURLBeforeWorkPersistence(t *testing.T) {
	for _, source := range []string{"https://example.test/media?token=private", "https://example.test/media?X-Amz-Signature=private", "https://user:private@example.test/media", "https://example.test/%zz?token=private", " https://example.test/media?jwt=private", "\thttps://example.test/media?session_token=private"} {
		if _, e := PrepareImport(ImportRequest{Items: []Item{{Source: source}}}); e == nil {
			t.Fatal("credential URL admitted to durable intent")
		}
	}
	if _, e := PrepareImport(ImportRequest{Items: []Item{{Source: "https://example.test/media?id=42&download=1"}}}); e != nil {
		t.Fatal("ordinary query selector rejected")
	}
	for _, source := range []string{"ordinary%file.wav", `C:\Recordings\100%.wav`} {
		if _, e := PrepareImport(ImportRequest{Items: []Item{{Source: source}}}); e != nil {
			t.Fatal("local percent/drive path rejected")
		}
	}
	if _, err := PrepareImport(ImportRequest{Items: []Item{{Source: "local.wav", Subtitle: "https://example.test/subtitle.srt"}}}); err != nil {
		t.Fatal("valid remote transcript rejected", err)
	}
	if _, err := PrepareImport(ImportRequest{Items: []Item{{Source: "local.wav", Subtitle: "https://example.test/%zz?token=private"}}}); err == nil {
		t.Fatal("credential URL accepted")
	}

}

func TestMissingSuppliedSubtitleKeepsOriginalAdmission(t *testing.T) {
	s, db := libraryFixture(t)
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: wavFixture(t), Subtitle: filepath.Join(t.TempDir(), "missing.srt")}}})
	value, e := s.Execute(context.Background(), work)
	if e != nil {
		t.Fatal(e)
	}
	result := value.(ImportResult)
	if !result.Partial || result.Items[0].State != "admitted" || result.Items[0].SubtitleState != "missing" {
		t.Fatal("missing subtitle discarded usable original")
	}
}
