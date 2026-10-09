// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
)

func readTranscript(path string, max int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, contracts.Fail("invalid_request")
	}
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return nil, contracts.Fail("operation_failed")
	}
	if int64(len(data)) > max {
		return nil, contracts.Fail("output_limit")
	}
	if len(data) == 0 {
		return nil, contracts.Fail("invalid_request")
	}
	return data, nil
}
func transcriptFormat(data []byte, hint, name string) (string, error) {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}))
	lower := strings.ToLower(string(trimmed[:min(len(trimmed), 512)]))
	if strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html") || bytes.IndexByte(trimmed, 0) >= 0 {
		return "", contracts.Fail("invalid_request")
	}
	if len(trimmed) > 0 && trimmed[0] == '{' {
		if hint != "" && hint != "cueson" {
			return "", contracts.Fail("invalid_request")
		}
		return "cueson", nil
	}
	if hint != "" {
		return hint, nil
	}
	if strings.HasPrefix(string(trimmed), "WEBVTT") {
		return "vtt", nil
	}
	if strings.Contains(lower, "[script info]") {
		if strings.Contains(strings.ToLower(string(data)), "[v4+ styles]") {
			return "ass", nil
		}
		return "ssa", nil
	}
	extension := strings.ToLower(filepath.Ext(name))
	if extension == ".srt" || extension == ".vtt" || extension == ".ass" || extension == ".ssa" {
		return strings.TrimPrefix(extension, "."), nil
	}
	if bytes.Contains(trimmed, []byte(" --> ")) {
		return "srt", nil
	}
	return "", contracts.Fail("invalid_request")
}
func (s *Service) transcript(ctx context.Context, item Item, options Options, directory string, entry *catalog.LibraryEntry) ([]byte, string, *Acquisition, error) {
	options = transcriptOptions(options)
	max := *options.AcquisitionMaxBytes
	path := item.Transcript
	var acquired *Acquisition
	if path == "<managed>" {
		if entry == nil || entry.SubtitlePublicationID == nil {
			return nil, "", nil, contracts.Fail("not_found")
		}
		materialized, err := s.Artifacts.MaterializeBound(ctx, *entry.SubtitlePublicationID, max)
		if err != nil {
			return nil, "", nil, err
		}
		defer s.Artifacts.Release(context.WithoutCancel(ctx), *entry.SubtitlePublicationID, materialized.Lease.ID)
		path = materialized.Path
	} else if hasRemoteScheme(path) {
		u, err := contracts.SourceURL(path)
		if err != nil {
			return nil, "", nil, err
		}
		adapterName := item.TranscriptAdapter
		if adapterName == "" {
			adapterName = u.Scheme
		}
		adapter := s.AcquisitionAdapters[adapterName]
		if adapter == nil {
			return nil, "", nil, contracts.Fail("unavailable")
		}
		path = filepath.Join(directory, "transcript"+filepath.Ext(u.Path))
		receipt, err := adapter.Acquire(ctx, item.Transcript, item.TranscriptCredentialID, path, options)
		if err != nil {
			return nil, "", nil, err
		}
		acquired = &receipt
	}
	data, err := readTranscript(path, max)
	if err != nil {
		return nil, "", acquired, err
	}
	format, err := transcriptFormat(data, options.TranscriptFormat, item.Transcript)
	return data, format, acquired, err
}
func (s *Service) document(ctx context.Context, data []byte, format, recording, mode string) (subtitles.Admission, error) {
	if format != "cueson" {
		if s.NativeIngest == nil {
			return subtitles.Admission{}, contracts.Fail("unavailable")
		}
		current, err := s.NativeIngest(ctx, data, format)
		if err != nil {
			return subtitles.Admission{}, err
		}
		data = current
	}
	result, err := subtitles.Preserve(data)
	if err != nil {
		return result, err
	}
	if mode == "" {
		mode = "auto"
	}
	if mode == "diarize" {
		mode = "off"
	}
	return subtitles.Attribute(result, recording, mode)
}
func (s *Service) target(ctx context.Context, selector string) (catalog.LibraryEntry, error) {
	if contracts.ValidID(selector) {
		return s.Catalog.Library(ctx, selector)
	}
	entries, err := s.Catalog.Libraries(ctx)
	if err != nil {
		return catalog.LibraryEntry{}, err
	}
	var target *catalog.LibraryEntry
	for _, entry := range entries {
		if entry.Title == selector {
			if target != nil {
				return catalog.LibraryEntry{}, contracts.Fail("conflict")
			}
			copy := entry
			target = &copy
		}
	}
	if target == nil {
		return catalog.LibraryEntry{}, contracts.Fail("not_found")
	}
	return *target, nil
}
func (s *Service) accepted(ctx context.Context, work catalog.Work, ordinal int) (ItemResult, bool, error) {
	raw, ok, err := s.Catalog.AcceptedAdmissionWork(ctx, work.ID, ordinal)
	if err != nil || !ok {
		return ItemResult{}, ok, err
	}
	var result struct {
		Result ItemResult `json:"admission_result"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Result.State == "" {
		return ItemResult{}, false, contracts.Fail("operation_failed")
	}
	if result.Result.MediaID != "" {
		if err = s.admissionCleanup(ctx, result.Result.MediaID); err != nil {
			return ItemResult{}, false, err
		}
	}
	return result.Result, true, nil
}
func (s *Service) admit(ctx context.Context, work catalog.Work, ordinal int, item Item, options Options, known *libraryLookup) (out ItemResult, returned error) {
	phase := "prepare"
	defer func() {
		if returned != nil {
			out.Notices = append(out.Notices, "admission_stage:"+phase)
		}
	}()
	if s.legacyFixture && s.Tools.FFmpeg.Path == "" && item.Record == "" {
		item.Subtitle = item.Transcript
		return s.admitLegacy(ctx, work, ordinal, item, options, known)
	}
	if options.Attribution == "" {
		options.Attribution = "auto"
	}
	if result, accepted, err := s.accepted(ctx, work, ordinal); err != nil || accepted {
		return result, err
	}
	if item.AcceptedReceipt != "" {
		return out, contracts.Fail("conflict")
	}
	directory, err := os.MkdirTemp(s.Artifacts.Workspace.Control, "admission-")
	if err != nil {
		return out, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(directory)
	if item.TargetError != "" {
		return out, contracts.Fail(item.TargetError)
	}
	var replacing *catalog.LibraryEntry
	var previous *catalog.Recording
	if targetedMedia(item, options) {
		target, current, e := s.replacementTarget(ctx, item, options)
		if e != nil {
			return out, e
		}
		if !audioElection(options) {
			out = resultOf(ordinal, target)
			out.State = "skipped"
			e = s.Catalog.SkipAdmission(ctx, work, ordinal, marshal(out))
			return out, e
		}
		replacing = &target
		previous = current
		if options.ExistingTranscript == "keep" && (previous == nil || previous.DocumentDigest == "") {
			data, format, _, e := s.transcript(ctx, Item{Transcript: "<managed>"}, options, directory, &target)
			if e != nil {
				return out, e
			}
			doc, e := s.document(ctx, data, format, target.ID, "off")
			if e != nil {
				return out, e
			}
			// Stage compatibility conversion without changing accepted state. The
			// prior current revision remains the transaction's compare-and-swap fence.
			compatibilityOptions := options
			compatibilityOptions.Attribution = "off"
			previous, e = s.makeRecording(ctx, target, doc, compatibilityOptions, nil, nil)
			if e != nil {
				return out, e
			}
			previous.Revision = 0
			if current != nil {
				previous.Revision = current.Revision
			}
		}
		if options.ExistingTranscript == "" {
			if item.Transcript != "" || options.SubtitleStreamIndex != nil || options.SubtitleLanguage != "" {
				options.ExistingTranscript = "replace"
			} else {
				options.ExistingTranscript = "clear"
			}
		}
	}
	if item.Record != "" && replacing == nil {
		return s.admitTranscript(ctx, work, ordinal, item, options, directory)
	}
	if options.Copy != nil && !*options.Copy {
		return out, contracts.Fail("invalid_request")
	}
	source := item.Source
	locator := ""
	var acquisition *Acquisition
	if hasRemoteScheme(source) {
		u, err := contracts.SourceURL(source)
		if err != nil {
			return out, err
		}
		adapterName := item.AcquisitionAdapter
		if adapterName == "" {
			adapterName = u.Scheme
		}
		adapter := s.AcquisitionAdapters[adapterName]
		if adapter == nil {
			return out, contracts.Fail("unavailable")
		}
		source = filepath.Join(directory, "media"+filepath.Ext(u.Path))
		receipt, err := adapter.Acquire(ctx, item.Source, item.CredentialID, source, options)
		if err != nil {
			return out, err
		}
		acquisition = &receipt
		locator = receipt.SourceLocator
	} else {
		source, err = filepath.Abs(source)
		if err != nil {
			return out, contracts.Fail("invalid_request")
		}
	}
	phase = "source-stage"
	staged, digest, size, err := s.Artifacts.Stage(ctx, source)
	if err != nil {
		return out, err
	}
	path := staged.Name()
	staged.Close()
	defer os.Remove(path)
	entryID := DerivedID(work.ID, "media-"+strconv.Itoa(ordinal))
	if replacing != nil {
		entryID = replacing.ID
	}
	// Canonical source identity replaces the discarded local locator as the dedup key.
	if !item.NewEntry && replacing == nil {
		for _, existing := range known.content {
			var facts Facts
			json.Unmarshal(existing.Facts, &facts)
			if facts.Canonical != nil && facts.Canonical.SourceDigest == digest && facts.Canonical.SourceSize == size {
				item.Record = existing.ID
				if item.Transcript == "" && existing.SubtitlePublicationID != nil {
					item.Transcript = "<managed>"
				}
				if item.Transcript != "" {
					return s.admitTranscript(ctx, work, ordinal, item, options, directory)
				}
				out = resultOf(ordinal, existing)
				out.State = "skipped"
				if err = s.Catalog.SkipAdmission(ctx, work, ordinal, marshal(out)); err != nil {
					return out, err
				}
				return out, nil
			}
		}
	}
	phase = "metadata-capture"
	bundle, err := s.capture(ctx, path, source, options)
	if err != nil {
		return out, err
	}
	bundle.Facts.Acquisition = acquisition
	if acquisition != nil {
		bundle.Facts.FilesystemModified = ""
	}
	tracks := embeddedTracks(bundle.Facts.Streams)
	notices := []string{}
	var input []byte
	format := ""
	var transcriptAcquisition *Acquisition
	if replacing != nil && options.ExistingTranscript != "replace" {
		// Existing-document policy deliberately bypasses candidate embedded text.
	} else if item.Transcript != "" {
		phase = "transcript-acquisition"
		input, format, transcriptAcquisition, err = s.transcript(ctx, item, options, directory, nil)
		if len(tracks) > 0 {
			notices = append(notices, "explicit_transcript_selected;embedded_tracks_unused")
		}
	} else {
		var track *EmbeddedTrack
		track, err = selectEmbedded(tracks, options)
		if err == nil && track != nil {
			for i := range tracks {
				tracks[i].Selected = tracks[i].Index == track.Index
			}
			input, format, err = s.extractEmbedded(ctx, path, directory, *track)
		} else if err == nil && len(tracks) > 0 {
			notices = append(notices, "image_subtitles_require_ocr;no_text_selected")
		}
	}
	if err != nil {
		return out, err
	}
	if replacing != nil && options.ExistingTranscript == "replace" && len(input) == 0 {
		return out, contracts.Fail("invalid_request")
	}
	var document *subtitles.Admission
	if len(input) > 0 {
		phase = "transcript-validation"
		value, err := s.document(ctx, input, format, entryID, options.Attribution)
		if err != nil {
			return out, err
		}
		document = &value
	}
	phase = "canonical-audio"
	canonical, facts, err := s.canonicalize(ctx, path, directory, bundle)
	if err != nil {
		return out, err
	}
	bundle.Facts = facts
	candidates := []string{}
	defer func() {
		if returned != nil {
			for _, id := range candidates {
				s.discardUnusedCandidate(id)
			}
		}
	}()
	phase = "canonical-publication"
	publicationID, err := s.freshAdmissionCandidate(ctx, DerivedID(work.ID, "canonical-"+strconv.Itoa(ordinal)))
	if err != nil {
		return out, err
	}
	candidates = append(candidates, publicationID)
	publication, err := s.Artifacts.Publish(ctx, publicationID, canonical, "canonical-audio")
	if err != nil {
		return out, err
	}
	reportID, err := s.freshAdmissionCandidate(ctx, DerivedID(work.ID, "canonical-report-"+strconv.Itoa(ordinal)))
	if err != nil {
		return out, err
	}
	candidates = append(candidates, reportID)
	title := item.Title
	if title == "" && replacing != nil {
		title = replacing.Title
	}
	if title == "" {
		title = filepath.Base(source)
		if acquisition != nil {
			u, _ := url.Parse(locator)
			title = filepath.Base(u.Path)
		}
	}
	if title == "" || title == "." {
		title = "Imported recording"
	}
	scrubBundle(&bundle, source, path, item.Source, directory)
	var oldDates []Date
	if replacing != nil {
		json.Unmarshal(replacing.Dates, &oldDates)
	}
	dates := s.dates(bundle, options, oldDates)
	if document != nil {
		bundle.Facts.SubtitleState = "current"
	}
	phase = "metadata-publication"
	if err = s.publishBundle(ctx, reportID, bundle, directory); err != nil {
		return out, err
	}
	entry := catalog.LibraryEntry{ID: entryID, AssetID: publication.ArtifactID, Title: title, Class: "audio", Mode: "copy", SourceLocator: locator, Digest: publication.Digest, Size: publication.Size, OriginalPublicationID: &publication.ID, DurationUS: bundle.Facts.DurationUS, Facts: marshal(bundle.Facts), Metadata: marshal(bundle.Metadata), Dates: marshal(dates), ReportPublicationIDs: marshal([]string{reportID})}
	var recording *catalog.Recording
	expectedRecording := int64(0)
	if replacing != nil {
		entry.Revision = replacing.Revision
		if previous != nil {
			expectedRecording = previous.Revision
		}
		if options.ExistingTranscript == "keep" {
			if err = validateKeptSelection(*replacing, entry, previous); err != nil {
				return out, err
			}
			copy := *previous
			copy.SourceDigest = entry.Digest
			copy.SourceRevision = 1
			copy.MappedAudioPublicationID = nil
			mapping := map[string]json.RawMessage{}
			json.Unmarshal(previous.SourceMap, &mapping)
			keptMap := map[string]any{"policy": "supplied-document-source-clock;no-retiming", "kept_with_applicability_assertion": true}
			for _, key := range []string{"stream_index", "channel"} {
				if value, ok := mapping[key]; ok {
					keptMap[key] = value
				}
			}
			copy.SourceMap = marshal(keptMap)
			recording = &copy
			if entry.DurationUS == nil {
				notices = append(notices, "kept_transcript_audio_bounds_unknown")
			}
		} else if options.ExistingTranscript == "clear" {
			recording = &catalog.Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: 1, State: "untranscribed", Document: json.RawMessage(`null`), SourceMap: json.RawMessage(`{}`), Provenance: marshal(map[string]any{"mode": "explicit-audio-replacement;transcript-clear"}), Diagnostics: json.RawMessage(`[]`)}
		}
	}
	if document != nil {
		phase = "current-document"
		recording, err = s.makeRecording(ctx, entry, *document, options, tracks, transcriptAcquisition)
		if err != nil {
			return out, err
		}
	}
	phase = "source-recheck"
	currentDigest, currentSize, err := identity(ctx, source)
	if err != nil || currentDigest != digest || currentSize != size {
		return out, contracts.Fail("conflict")
	}
	out = resultOf(ordinal, entry)
	out.Notices = notices
	phase = "current-acceptance"
	if replacing != nil {
		out.State = "replaced"
	}
	if replacing != nil || options.KnownSpeakers != nil {
		ids := item.ResolvedSpeakerIDs
		if options.KnownSpeakers != nil && ids == nil {
			ids, err = s.Catalog.ResolveSpeakers(ctx, options.KnownSpeakers)
			if err != nil {
				return out, err
			}
		}
		rosterRevision := int64(0)
		if item.ExpectedRosterRevision != nil {
			rosterRevision = *item.ExpectedRosterRevision
		} else if replacing != nil {
			roster, e := s.Catalog.Roster(ctx, entry.ID)
			if e != nil {
				return out, e
			}
			rosterRevision = roster.Revision
		}
		entry, recording, err = s.Catalog.CommitElectedAdmission(ctx, work, ordinal, expectedRecording, entry, recording, catalog.AdmissionElection{ReplaceAudio: replacing != nil, ExpectedRosterRevision: rosterRevision, RosterPolicy: options.ExistingRoster, SpeakerIDs: ids}, marshal(out))
	} else {
		entry, recording, err = s.Catalog.CommitAdmission(ctx, work, ordinal, 0, entry, recording, marshal(out))
	}
	if err != nil {
		return out, err
	}
	if recording != nil {
		out.RecordingRevision = recording.Revision
		out.DocumentDigest = recording.DocumentDigest
	}
	if err = s.admissionCleanup(ctx, entry.ID); err != nil {
		return out, err
	}
	return out, nil
}
func (s *Service) makeRecording(ctx context.Context, entry catalog.LibraryEntry, input subtitles.Admission, options Options, tracks []EmbeddedTrack, acquisition *Acquisition) (*catalog.Recording, error) {
	sourceMap := marshal(map[string]any{"policy": "supplied-document-source-clock;no-retiming", "embedded_tracks": tracks})
	provenance := marshal(map[string]any{"mode": "import", "original_schema_version": input.OriginalVersion, "translated": input.Translated, "attribution": options.Attribution, "attribution_basis": input.AttributionBasis, "transcript_acquisition": acquisition})
	if options.Attribution == "diarize" {
		if s.Diarize == nil {
			return nil, contracts.Fail("unavailable")
		}
		originalProvenance := provenance
		var err error
		input, provenance, err = s.Diarize(ctx, entry, input.Document, options.DiarizationModelID)
		if err != nil {
			return nil, err
		}
		var detail struct {
			SourceMap json.RawMessage `json:"source_map"`
		}
		if json.Unmarshal(provenance, &detail) != nil || len(detail.SourceMap) == 0 {
			return nil, contracts.Fail("invalid_request")
		}
		var mapping map[string]json.RawMessage
		if json.Unmarshal(detail.SourceMap, &mapping) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		mapping["supplied_document"] = json.RawMessage(`true`)
		sourceMap = marshal(mapping)
		var processing map[string]json.RawMessage
		json.Unmarshal(provenance, &processing)
		processing["import_provenance"] = originalProvenance
		provenance = marshal(processing)
	}
	if err := validateAudioAttribution(entry, input.Document); err != nil {
		return nil, err
	}
	// Match the catalog's JSON serialization before hashing. This compacts JSON
	// whitespace/escapes without decoding numbers or changing source/native data.
	input.Document, _ = json.Marshal(input.Document)
	sum, _ := identityBytes(input.Document)
	return &catalog.Recording{ID: entry.ID, SourceRevision: max(entry.Revision, 1), SourceDigest: entry.Digest, State: "ready", Document: input.Document, DocumentDigest: sum, SourceMap: sourceMap, Provenance: provenance, Diagnostics: json.RawMessage(`[]`)}, nil
}
func identityBytes(data []byte) (string, int64) {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), int64(len(data))
}
func (s *Service) admitTranscript(ctx context.Context, work catalog.Work, ordinal int, item Item, options Options, directory string) (ItemResult, error) {
	if item.TargetError != "" {
		return ItemResult{}, contracts.Fail(item.TargetError)
	}
	entry, err := s.target(ctx, item.Record)
	if err != nil {
		return ItemResult{}, err
	}
	previous, previousErr := s.Catalog.Recording(ctx, entry.ID)
	expected := int64(0)
	if previousErr == nil {
		expected = previous.Revision
	}
	if previousErr != nil && code(previousErr) != "not_found" {
		return ItemResult{}, previousErr
	}
	if item.ExpectedRevision != nil && entry.Revision != *item.ExpectedRevision || item.ExpectedRecordingRevision != nil && expected != *item.ExpectedRecordingRevision {
		return ItemResult{}, contracts.Fail("conflict")
	}
	if previousErr == nil && previous.State == "ready" && (options.ReplaceTranscript == nil || !*options.ReplaceTranscript) {
		out := resultOf(ordinal, entry)
		out.State = "skipped"
		out.RecordingRevision = previous.Revision
		out.DocumentDigest = previous.DocumentDigest
		err = s.Catalog.SkipAdmission(ctx, work, ordinal, marshal(out))
		return out, err
	}
	if item.Transcript == "" {
		return ItemResult{}, contracts.Fail("invalid_request")
	}
	data, format, acquired, err := s.transcript(ctx, item, options, directory, &entry)
	if err != nil {
		return ItemResult{}, err
	}
	document, err := s.document(ctx, data, format, entry.ID, options.Attribution)
	if err != nil {
		return ItemResult{}, err
	}
	recording, err := s.makeRecording(ctx, entry, document, options, []EmbeddedTrack{}, acquired)
	if err != nil {
		return ItemResult{}, err
	}
	var facts Facts
	json.Unmarshal(entry.Facts, &facts)
	facts.SubtitleState = "current"
	entry.Facts = marshal(facts)
	entry.SubtitlePublicationID = nil
	out := resultOf(ordinal, entry)
	if previousErr == nil {
		out.State = "replaced"
	}
	entry, recording, err = s.Catalog.CommitAdmission(ctx, work, ordinal, expected, entry, recording, marshal(out))
	if err != nil {
		return out, err
	}
	out.RecordingRevision = recording.Revision
	out.DocumentDigest = recording.DocumentDigest
	if err = s.admissionCleanup(ctx, entry.ID); err != nil {
		return out, err
	}
	return out, nil
}

// An interrupted unaccepted Matroska encode may contain a randomized container
// UID. Retire an unreferenced prior candidate before publishing a recomputation;
// accepted receipts are checked first and current-reference fences remain final.
func (s *Service) freshAdmissionCandidate(ctx context.Context, base string) (string, error) {
	id, err := s.candidateAttempt(ctx, base)
	if err != nil {
		return "", err
	}
	if p, err := s.Catalog.Publication(ctx, id); err == nil {
		if p.State == "pending" {
			if _, err = s.Artifacts.Reconcile(ctx, id); err != nil {
				return "", err
			}
		}
		if _, err = s.Artifacts.RetireCurrent(ctx, id); err != nil {
			return "", err
		}
		return s.candidateAttempt(ctx, base)
	} else if code(err) != "not_found" {
		return "", err
	}
	return id, nil
}

// Live leases and another current reference leave durable cleanup pending.
// Genuine storage failures remain retryable failures rather than being hidden.
func (s *Service) admissionCleanup(ctx context.Context, id string) error {
	err := s.Cleanup(ctx, id)
	if code(err) == "conflict" {
		return nil
	}
	return err
}

// FreezeTargets resolves standalone selectors and observed revisions before work
// is queued. Per-item failures remain batch results, and cannot later bind a
// newly created or newly ambiguous title.
func FreezeTargets(ctx context.Context, db catalog.Catalog, request ImportRequest) ImportRequest {
	service := Service{Catalog: db}
	for i := range request.Items {
		item := &request.Items[i]
		item.TargetError = ""
		options := merged(request.Defaults, item.Options)
		if options.KnownSpeakers != nil {
			ids, e := db.ResolveSpeakers(ctx, options.KnownSpeakers)
			if e != nil {
				item.TargetError = code(e)
				continue
			}
			item.ResolvedSpeakerIDs = ids
		}
		if item.Record == "" {
			continue
		}
		entry, err := service.target(ctx, item.Record)
		if err != nil {
			item.TargetError = code(err)
			continue
		}
		current, err := db.Recording(ctx, entry.ID)
		revision := int64(0)
		if err == nil {
			revision = current.Revision
		} else if code(err) != "not_found" {
			item.TargetError = code(err)
			continue
		}
		if item.ExpectedRevision != nil && *item.ExpectedRevision != entry.Revision || item.ExpectedRecordingRevision != nil && *item.ExpectedRecordingRevision != revision {
			item.TargetError = "conflict"
			continue
		}
		roster, e := db.Roster(ctx, entry.ID)
		if e != nil {
			item.TargetError = code(e)
			continue
		}
		if item.ExpectedRosterRevision != nil && *item.ExpectedRosterRevision != roster.Revision {
			item.TargetError = "conflict"
			continue
		}
		rr := roster.Revision
		item.ExpectedRosterRevision = &rr
		item.Record = entry.ID
		observed := entry.Revision
		item.ExpectedRevision = &observed
		item.ExpectedRecordingRevision = &revision
	}
	return request
}

// Native cue conflicts remain source facts. Only new consumer interval claims
// are checked against known audio in the original presentation clock.
func validateAudioAttribution(entry catalog.LibraryEntry, document json.RawMessage) error {
	var doc struct {
		Cues []struct {
			Assignments []struct {
				Start *int64 `json:"start_milliseconds"`
				End   *int64 `json:"end_milliseconds"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
	}
	if json.Unmarshal(document, &doc) != nil {
		return contracts.Fail("invalid_request")
	}
	var facts Facts
	if json.Unmarshal(entry.Facts, &facts) != nil {
		return contracts.Fail("invalid_request")
	}
	type interval struct{ start, end *big.Rat }
	bounds := []interval{}
	complete := true
	for _, stream := range facts.Streams {
		if stream.Kind != "audio" {
			continue
		}
		start, err := streamStart(stream)
		clockKnown := err == nil || facts.Container == "wav" || facts.Container == "flac"
		if err != nil {
			start = new(big.Rat)
		}
		original := new(big.Rat).Set(start)
		if facts.Canonical != nil {
			for _, track := range facts.Canonical.Tracks {
				if track.Index == stream.Index {
					if r, ok := new(big.Rat).SetString(track.StartNumerator + "/" + track.StartDenominator); ok {
						original = r
						clockKnown = true
					}
				}
			}
		}
		if !clockKnown {
			complete = false
			continue
		}
		duration := new(big.Rat)
		known := false
		if stream.DurationTS != nil {
			if base, ok := new(big.Rat).SetString(stream.TimeBase); ok && base.Sign() > 0 {
				duration.Mul(base, new(big.Rat).SetInt64(*stream.DurationTS))
				known = duration.Sign() >= 0
			}
		}
		if !known {
			if r, ok := new(big.Rat).SetString(stream.Duration); ok && r.Sign() >= 0 {
				duration = r
				known = true
			}
		}
		if !known {
			var tag string
			json.Unmarshal(stream.Tags["DURATION"], &tag)
			parts := strings.Split(tag, ":")
			if len(parts) == 3 {
				h, ok1 := new(big.Rat).SetString(parts[0])
				m, ok2 := new(big.Rat).SetString(parts[1])
				sec, ok3 := new(big.Rat).SetString(parts[2])
				if ok1 && ok2 && ok3 {
					end := new(big.Rat).Add(new(big.Rat).Mul(h, big.NewRat(3600, 1)), new(big.Rat).Mul(m, big.NewRat(60, 1)))
					end.Add(end, sec)
					duration.Sub(end, start)
					known = duration.Sign() >= 0
				}
			}
		}
		if !known && entry.DurationUS != nil {
			duration.SetFrac64(*entry.DurationUS, 1000000)
			if strings.Contains(facts.Container, "matroska") || strings.Contains(facts.Container, "webm") {
				duration.Sub(duration, start)
			}
			known = duration.Sign() >= 0
		}
		if !known {
			complete = false
			continue
		}
		bounds = append(bounds, interval{original, new(big.Rat).Add(original, duration)})
	}
	if len(bounds) == 0 {
		if entry.DurationUS == nil {
			return nil
		}
		bounds = append(bounds, interval{new(big.Rat), big.NewRat(*entry.DurationUS, 1000000)})
	}
	if !complete {
		return nil
	}
	for _, cue := range doc.Cues {
		for _, claim := range cue.Assignments {
			if claim.Start == nil || claim.End == nil {
				continue
			}
			a, b := big.NewRat(*claim.Start, 1000), big.NewRat(*claim.End, 1000)
			inside := false
			for _, bound := range bounds {
				if a.Cmp(bound.start) >= 0 && b.Cmp(bound.end) <= 0 {
					inside = true
					break
				}
			}
			if !inside {
				return contracts.Fail("invalid_timing")
			}
		}
	}
	return nil
}
