// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

type acquisitionSecret struct{ resolved []byte }

func TestAcquisitionBoundsCancellationAndRedaction(t *testing.T) {
	for _, tc := range []struct{ mode, want string }{
		{"headers-timeout", "acquisition_timeout"}, {"body-timeout", "acquisition_timeout"},
		{"probe-timeout", "acquisition_timeout"}, {"cancel", "cancelled"},
		{"declared-limit", "input_limit"}, {"streamed-limit", "input_limit"},
		{"exact-limit", ""}, {"truncated", "operation_failed"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch tc.mode {
				case "headers-timeout":
					<-r.Context().Done()
				case "body-timeout", "probe-timeout", "cancel":
					w.WriteHeader(http.StatusOK)
					if tc.mode == "probe-timeout" {
						w.Write([]byte("12345678"))
					} else {
						w.Write([]byte("1234"))
					}
					w.(http.Flusher).Flush()
					if tc.mode == "cancel" {
						cancel()
					}
					<-r.Context().Done()
				case "declared-limit":
					w.Header().Set("Content-Length", "9")
					w.Write([]byte("123456789"))
				case "streamed-limit":
					w.(http.Flusher).Flush()
					w.Write([]byte("123456789"))
				case "truncated":
					w.Header().Set("Content-Length", "7")
					w.Write([]byte("1234"))
				default:
					w.Write([]byte("12345678"))
				}
			}))
			defer server.Close()
			max, timeout, local := int64(8), int64(100), true
			if tc.mode == "cancel" {
				timeout = 1000
			}
			destination := filepath.Join(t.TempDir(), "private-input")
			adapter := httpAcquisition{Secrets: &acquisitionSecret{}}
			_, err := adapter.Acquire(ctx, server.URL+"/private-selector", contracts.ID(), destination, Options{AcquisitionMaxBytes: &max, AcquisitionTimeoutMS: &timeout, LocalHTTP: &local})
			if tc.want == "" {
				data, readErr := os.ReadFile(destination)
				if err != nil || readErr != nil || string(data) != "12345678" {
					t.Fatal("exact cap rejected", err, readErr)
				}
				return
			}
			if code(err) != tc.want {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
			if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
				t.Fatal("failed acquisition left scratch bytes", statErr)
			}
			for _, forbidden := range []string{server.URL, "private-selector", "fixture-access", "12345678"} {
				if strings.Contains(err.Error(), forbidden) {
					t.Fatal("private transport details escaped")
				}
			}
		})
	}
}

func TestTranscriptAcquisitionFailurePersistsCorrectionAndAcceptedDocument(t *testing.T) {
	for _, tc := range []struct{ mode, want, flag string }{{"timeout", "acquisition_timeout", "--transcript-timeout-ms"}, {"declared", "input_limit", "--transcript-max-bytes"}, {"streamed", "input_limit", "--transcript-max-bytes"}} {
		t.Run(tc.mode, func(t *testing.T) {
			s, db := libraryFixture(t)
			ctx := context.Background()
			entry := seedLegacyAdmission(t, s, db, "")
			transcript, _ := transcriptFixture(t)
			if first := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: transcript}); first.State != "admitted" {
				t.Fatal(first)
			}
			before, err := db.Recording(ctx, entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.mode == "timeout" {
					<-r.Context().Done()
					return
				}
				if tc.mode == "streamed" {
					w.(http.Flusher).Flush()
				}
				w.Write([]byte("1\n00:00:00,000 --> 00:00:00,010\nprivate fixture text\n"))
			}))
			defer server.Close()
			max, timeout, replace := int64(8), int64(100), true
			work := claimLibraryWork(t, db, "media.import", ImportRequest{Items: []Item{{Record: entry.ID, Transcript: server.URL + "/selected.srt", Options: Options{ReplaceTranscript: &replace, TranscriptMaxBytes: &max, TranscriptTimeoutMS: &timeout}}}})
			result, err := s.Execute(ctx, work)
			if err != nil {
				t.Fatal(err)
			}
			item := result.(ImportResult).Items[0]
			if item.State != "failed" || item.Error != tc.want || !strings.Contains(strings.Join(item.Notices, " "), tc.flag) {
				t.Fatalf("failure contract: %+v", item)
			}
			current, err := db.Recording(ctx, entry.ID)
			if err != nil || current.Revision != before.Revision || !bytes.Equal(current.Document, before.Document) {
				t.Fatal("failed acquisition changed accepted document", err)
			}
			// Reopen through another catalog connection; corrections are persisted,
			// rather than supplied by the lifetime of this service instance.
			reopened, err := catalog.OpenWorkspace(ctx, s.Artifacts.Workspace, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			durable, err := reopened.Work(ctx, work.ID)
			if err != nil {
				t.Fatal(err)
			}
			var persisted ImportResult
			if json.Unmarshal(durable.Result, &persisted) != nil || len(persisted.Items) != 1 || persisted.Items[0].Error != tc.want || !strings.Contains(strings.Join(persisted.Items[0].Notices, " "), tc.flag) {
				t.Fatal("durable correction lost", string(durable.Result))
			}
			if bytes.Contains(durable.Result, []byte("private fixture text")) {
				t.Fatal("response body leaked into durable failure")
			}
		})
	}
}

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
