// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/subtitles"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func seedLegacyAdmission(t *testing.T, s *Service, db catalog.Catalog, sidecar string) catalog.LibraryEntry {
	t.Helper()
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: wavFixture(t), Subtitle: sidecar}}})
	result, err := s.Execute(context.Background(), work)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := db.Library(context.Background(), result.(ImportResult).Items[0].MediaID)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}
func transcriptFixture(t *testing.T) (string, []byte) {
	t.Helper()
	data, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transcript.cueson.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path, data
}
func executeTranscript(t *testing.T, s *Service, db catalog.Catalog, item Item) ItemResult {
	t.Helper()
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "off"}, Items: []Item{item}})
	result, err := s.Execute(context.Background(), work)
	if err != nil {
		t.Fatal(err)
	}
	return result.(ImportResult).Items[0]
}
func TestStandaloneAdmissionCollisionReplacementAndReplay(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	entry := seedLegacyAdmission(t, s, db, "")
	path, data := transcriptFixture(t)
	first := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: path})
	if first.State != "admitted" {
		t.Fatalf("first %+v", first)
	}
	old, err := db.Recording(ctx, entry.ID)
	if err != nil || !bytes.Equal(old.Document, bytes.TrimSpace(data)) {
		t.Fatal("document not preserved", err)
	}
	skip := executeTranscript(t, s, db, Item{Record: entry.Title, Transcript: filepath.Join(t.TempDir(), "missing.json")})
	if skip.State != "skipped" || skip.DocumentDigest != old.DocumentDigest {
		t.Fatalf("collision %+v", skip)
	}
	current, _ := db.Recording(ctx, entry.ID)
	if current.Revision != old.Revision {
		t.Fatal("skip changed current")
	}
	replace := true
	bad := filepath.Join(t.TempDir(), "error.json")
	os.WriteFile(bad, []byte("<html>not a transcript</html>"), 0600)
	failed := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: bad, Options: Options{ReplaceTranscript: &replace}})
	if failed.State != "failed" {
		t.Fatalf("invalid replacement %+v", failed)
	}
	current, _ = db.Recording(ctx, entry.ID)
	if current.Revision != old.Revision {
		t.Fatal("invalid input changed current")
	}
	replaced := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: path, Options: Options{ReplaceTranscript: &replace}})
	if replaced.State != "replaced" || replaced.RecordingRevision <= old.Revision {
		t.Fatalf("replacement %+v", replaced)
	}
	snapshot, err := db.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(snapshot)
	json.Unmarshal(raw, &snapshot)
	restored, err := catalog.OpenSQLite(ctx, filepath.Join(t.TempDir(), "fresh.sqlite"), snapshot.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if err = restored.Restore(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
}
func TestManagedSidecarConversionRetiresOnlyOwnedBytes(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	path, _ := transcriptFixture(t)
	entry := seedLegacyAdmission(t, s, db, path)
	if entry.SubtitlePublicationID == nil {
		t.Fatal("legacy sidecar missing")
	}
	sidecar := *entry.SubtitlePublicationID
	result := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: "<managed>"})
	if result.State != "admitted" {
		t.Fatalf("conversion %+v", result)
	}
	current, _ := db.Library(ctx, entry.ID)
	if current.SubtitlePublicationID != nil || *current.OriginalPublicationID != *entry.OriginalPublicationID {
		t.Fatal("incorrect retirement")
	}
	publication, err := db.Publication(ctx, sidecar)
	if err != nil || publication.State != "retired" {
		t.Fatal("sidecar retained", err)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal("owner input removed")
	}
	if _, err = db.Export(ctx); err != nil {
		t.Fatal("retirement proof", err)
	}
}

type transcriptSecret struct{ seen []string }

func (s *transcriptSecret) Resolve(_ context.Context, id string) ([]byte, error) {
	s.seen = append(s.seen, id)
	return []byte("transcript-only"), nil
}
func (*transcriptSecret) Status(context.Context, string) (string, error) { return "available", nil }
func TestIndependentRemoteTranscriptValidationAndBounds(t *testing.T) {
	s, db := libraryFixture(t)
	entry := seedLegacyAdmission(t, s, db, "")
	_, data := transcriptFixture(t)
	credential := contracts.ID()
	mediaCredential := contracts.ID()
	secret := &transcriptSecret{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer transcript-only" {
			t.Error("independent credential missing")
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}))
	defer server.Close()
	s.AcquisitionAdapters["http"] = httpAcquisition{Secrets: secret}
	local := true
	result := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: server.URL + "/wrong.srt", CredentialID: mediaCredential, TranscriptCredentialID: credential, Options: Options{LocalHTTP: &local}})
	if result.State != "admitted" || len(secret.seen) != 1 || secret.seen[0] != credential {
		t.Fatalf("remote %+v credentials %v", result, secret.seen)
	}
	recording, err := db.Recording(context.Background(), entry.ID)
	if err != nil || !bytes.Contains(recording.Provenance, []byte("text/html")) {
		t.Fatal("MIME provenance missing", err)
	}
	replace := true
	limit := int64(1)
	failed := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: server.URL + "/bounded", TranscriptCredentialID: credential, Options: Options{LocalHTTP: &local, ReplaceTranscript: &replace, TranscriptMaxBytes: &limit}})
	if failed.State != "failed" {
		t.Fatal("oversize transcript admitted")
	}
	same, _ := db.Recording(context.Background(), entry.ID)
	if same.Revision != recording.Revision {
		t.Fatal("limit failure changed current")
	}
}
func TestTranscriptBOMDetectionAndEffectiveOptions(t *testing.T) {
	format, err := transcriptFormat([]byte("\xef\xbb\xbfWEBVTT\n\n"), "", "download")
	if err != nil || format != "vtt" {
		t.Fatal("BOM sniff", format, err)
	}
	max := int64(100)
	timeout := int64(200)
	replace := true
	request, err := PrepareImport(ImportRequest{Defaults: Options{Attribution: "auto", TranscriptMaxBytes: &max}, Items: []Item{{Record: contracts.ID(), Source: "fixture.json", Options: Options{Attribution: "off", ReplaceTranscript: &replace, TranscriptTimeoutMS: &timeout}}}})
	if err != nil {
		t.Fatal(err)
	}
	item := request.Items[0]
	effective := merged(request.Defaults, item.Options)
	if item.Source != "" || item.Transcript != "fixture.json" || effective.Attribution != "off" || *effective.TranscriptMaxBytes != 100 || *effective.TranscriptTimeoutMS != 200 || !*effective.ReplaceTranscript {
		t.Fatal("effective precedence")
	}
}
func TestNativeAdmissionEmbeddedAndNativeFormats(t *testing.T) {
	tools := nativeTools(t)
	executable := os.Getenv("CUESON_EXECUTABLE")
	if executable == "" {
		t.Skip("pinned Cueson not selected")
	}
	digest, _, err := identity(context.Background(), executable)
	if err != nil {
		t.Fatal(err)
	}
	driver, err := subtitles.New(subtitles.Tool{Executable: executable, ExecutableSHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	s, db := libraryFixture(t)
	s.Tools = tools
	s.legacyFixture = false
	s.NativeIngest = driver.Ingest
	ctx := context.Background()
	dir := t.TempDir()
	wav := wavFixture(t)
	native := filepath.Join(dir, "voice.vtt")
	os.WriteFile(native, []byte("WEBVTT\n\n00:00:02.000 --> 00:00:02.010\n<v Alice>Text from selected track\n\n"), 0600)

	container := filepath.Join(dir, "embedded.mkv")
	mediaFixtureCommand(t, tools.FFmpeg, "-i", wav, "-i", native, "-map", "0:a:0", "-map", "1:s:0", "-c:a", "flac", "-c:s", "webvtt", "-metadata:s:s:0", "language=eng", container)
	result, err := s.Execute(ctx, claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", SubtitleLanguage: "eng"}, Items: []Item{{Source: container}}}))
	if err != nil {
		t.Fatal(err)
	}
	item := result.(ImportResult).Items[0]
	if item.State != "admitted" {
		t.Fatalf("embedded %+v", item)
	}
	recording, err := db.Recording(ctx, item.MediaID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(recording.Document, []byte("Text from selected track")) || !bytes.Contains(recording.Document, []byte("2000")) {
		t.Fatal("embedded text/time lost")
	}
	current, _ := transcriptFixture(t)
	result, err = s.Execute(ctx, claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "off"}, Items: []Item{{Source: container, Transcript: current, NewEntry: true}}}))
	if err != nil {
		t.Fatal(err)
	}
	item = result.(ImportResult).Items[0]
	if item.State != "admitted" || len(item.Notices) != 1 {
		t.Fatalf("explicit precedence %+v", item)
	}
	// Qualify each original native format against the actual packaged executable.
	for _, versioned := range []string{"historical-v1.1.0-ass.json", "historical-v1.1.0-ssa.json", "historical-v1.1.0-subrip.json", "historical-v1.1.0-webvtt.json"} {
		raw, err := os.ReadFile("../subtitles/testdata/" + versioned)
		if err != nil {
			t.Fatal(err)
		}
		var document struct {
			Format string `json:"format"`
			Source struct {
				Assets []struct {
					Data string `json:"data_base64"`
				} `json:"assets"`
			} `json:"source"`
		}
		json.Unmarshal(raw, &document)
		if len(document.Source.Assets) == 0 {
			t.Fatal("source assets missing")
		}
		encoded := document.Source.Assets[0].Data
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(data) == 0 {
			t.Fatal("native envelope fixture", versioned)
		}
		format := document.Format
		if format == "subrip" {
			format = "srt"
		}
		if format == "webvtt" {
			format = "vtt"
		}
		if _, err = s.document(ctx, data, format, contracts.ID(), "auto"); err != nil {
			t.Fatal(versioned, err)
		}
	}
}

type admissionFault struct {
	catalog.Catalog
	unknown bool
	failed  bool
}

func (f *admissionFault) CommitAdmission(ctx context.Context, w catalog.Work, ordinal int, expected int64, e catalog.LibraryEntry, r *catalog.Recording, result ...json.RawMessage) (catalog.LibraryEntry, *catalog.Recording, error) {
	if !f.failed {
		f.failed = true
		if !f.unknown {
			return e, r, contracts.Fail("unavailable")
		}
		current, document, err := f.Catalog.CommitAdmission(ctx, w, ordinal, expected, e, r, result...)
		if err == nil {
			return current, document, contracts.Fail("unavailable")
		}
		return current, document, err
	}
	return f.Catalog.CommitAdmission(ctx, w, ordinal, expected, e, r, result...)
}
func TestNativeAdmissionUnknownCommitAndCandidateRetry(t *testing.T) {
	tools := nativeTools(t)
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "rejected", true: "accepted-response-lost"}[unknown], func(t *testing.T) {
			s, db := libraryFixture(t)
			s.Tools = tools
			s.legacyFixture = false
			s.Catalog = &admissionFault{Catalog: db, unknown: unknown}
			path, _ := transcriptFixture(t)
			source := wavFixture(t)
			work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "off"}, Items: []Item{{Source: source, Transcript: path}}})
			result, err := s.Execute(context.Background(), work)
			if err != nil || result.(ImportResult).Items[0].State != "failed" {
				t.Fatal("fault not exercised", err)
			}
			if unknown {
				os.Remove(source)
				os.Remove(path)
			} else {
				requireCandidateRetired(t, s, db, DerivedID(work.ID, "canonical-0"))
				requireCandidateRetired(t, s, db, DerivedID(work.ID, "canonical-report-0"))
			}
			durable, err := db.Work(context.Background(), work.ID)
			if err != nil {
				t.Fatal(err)
			}
			result, err = s.Execute(context.Background(), durable)
			if err != nil || result.(ImportResult).Items[0].State != "admitted" {
				t.Fatal("retry failed", result, err)
			}
			entries, err := db.Libraries(context.Background())
			if err != nil || len(entries) != 1 {
				t.Fatal("duplicate admission")
			}
			if _, err = db.Export(context.Background()); err != nil {
				t.Fatal("portable retry proof", err)
			}
		})
	}
}

func TestSidecarConversionWaitsForLeasesAndSharedCurrentReferences(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	path, _ := transcriptFixture(t)
	entry := seedLegacyAdmission(t, s, db, path)
	sidecar := *entry.SubtitlePublicationID
	lease, err := s.Artifacts.MaterializeBound(ctx, sidecar, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	other := seedLegacyAdmission(t, s, db, "")
	other.SubtitlePublicationID = &sidecar
	other, err = db.UpdateLibrary(ctx, contracts.ID(), other.Revision, other)
	if err != nil {
		t.Fatal(err)
	}
	result := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: "<managed>"})
	if result.State != "admitted" {
		t.Fatalf("shared conversion %+v", result)
	}
	pub, _ := db.Publication(ctx, sidecar)
	if pub.State != "available" {
		t.Fatal("shared source retired")
	}
	if _, err = db.Export(ctx); err != nil {
		t.Fatal("shared proof", err)
	}
	if err = s.Artifacts.Release(ctx, sidecar, lease.Lease.ID); err != nil {
		t.Fatal(err)
	}
	result = executeTranscript(t, s, db, Item{Record: other.ID, Transcript: "<managed>"})
	if result.State != "admitted" {
		t.Fatal("second conversion", result)
	}
	pub, _ = db.Publication(ctx, sidecar)
	if pub.State != "retired" {
		t.Fatal("last reference not retired")
	}
	if _, err = db.Export(ctx); err != nil {
		t.Fatal("retired proof", err)
	}
}

func TestFrozenStandaloneReplacementRejectsDelayedEdit(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	entry := seedLegacyAdmission(t, s, db, "")
	path, _ := transcriptFixture(t)
	first := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: path})
	if first.State != "admitted" {
		t.Fatal(first)
	}
	replace := true
	frozen := FreezeTargets(ctx, db, ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "off"}, Items: []Item{{Record: entry.Title, Transcript: path, Options: Options{ReplaceTranscript: &replace}}}})
	if frozen.Items[0].Record != entry.ID || frozen.Items[0].ExpectedRevision == nil || frozen.Items[0].ExpectedRecordingRevision == nil {
		t.Fatal("target not frozen")
	}
	newer := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: path, Options: Options{ReplaceTranscript: &replace}})
	work := claimLibraryWork(t, db, "media.import", frozen)
	result, err := s.Execute(ctx, work)
	if err != nil || result.(ImportResult).Items[0].Error != "conflict" {
		t.Fatal("delayed edit overwritten", result, err)
	}
	current, _ := db.Recording(ctx, entry.ID)
	if current.Revision != newer.RecordingRevision {
		t.Fatal("stale candidate changed current")
	}
}

func TestImportedIntervalsUseActualAudioBounds(t *testing.T) {
	_, raw := transcriptFixture(t)
	var doc map[string]json.RawMessage
	json.Unmarshal(raw, &doc)
	var cues []map[string]json.RawMessage
	json.Unmarshal(doc["cues"], &cues)
	cues[0]["speaker_attributions"] = json.RawMessage(`[{"speaker_id":"local voice","start_milliseconds":1000,"end_milliseconds":2000}]`)
	doc["cues"], _ = json.Marshal(cues)
	data, _ := json.Marshal(doc)
	duration := int64(1000000)
	entry := catalog.LibraryEntry{DurationUS: &duration, Facts: json.RawMessage(`{"streams":[{"index":0,"codec_type":"audio","start_time":"0","duration":"1"}]}`)}
	if err := validateAudioAttribution(entry, data); err == nil {
		t.Fatal("out-of-audio interval accepted")
	}
	entry.Facts = json.RawMessage(`{"canonical":{"tracks":[{"index":0,"source_start_numerator":"1","source_start_denominator":"1"}]},"streams":[{"index":0,"codec_type":"audio","start_time":"0","duration":"1"}]}`)
	if err := validateAudioAttribution(entry, data); err != nil {
		t.Fatal("original offset ignored", err)
	}
	entry.DurationUS = nil
	entry.Facts = json.RawMessage(`{"streams":[{"index":0,"codec_type":"audio","start_time":"0"}]}`)
	if err := validateAudioAttribution(entry, data); err != nil {
		t.Fatal("unknown duration treated as zero", err)
	}
	entry.DurationUS = &duration
	entry.Facts = json.RawMessage(`{"streams":[]}`)
	cues[0]["speaker_attributions"] = json.RawMessage(`[{"speaker_id":"local voice"}]`)
	doc["cues"], _ = json.Marshal(cues)
	data, _ = json.Marshal(doc)
	if err := validateAudioAttribution(entry, data); err != nil {
		t.Fatal("native cue conflict blocked untimed participation", err)
	}
	duration = 0
	entry.Facts = json.RawMessage(`{"streams":[]}`)
	cues[0]["speaker_attributions"] = json.RawMessage(`[{"speaker_id":"local voice","start_milliseconds":0,"end_milliseconds":1}]`)
	doc["cues"], _ = json.Marshal(cues)
	data, _ = json.Marshal(doc)
	if err := validateAudioAttribution(entry, data); err == nil {
		t.Fatal("zero duration treated as unavailable")
	}
}
