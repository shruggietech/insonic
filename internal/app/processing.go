// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/schemas"
)

type ProcessingTools struct {
	Kind       string            `json:"kind"`
	Version    string            `json:"schema_version"`
	Cueson     subtitles.Tool    `json:"cueson"`
	Processing processing.Config `json:"processing"`
}
type RecordingOptions struct {
	PipelineID         string                        `json:"pipeline_id,omitempty"`
	PipelineRevision   int64                         `json:"pipeline_revision,omitempty"`
	Overrides          pipeline.Override             `json:"overrides,omitempty"`
	Context            catalog.ContextFilter         `json:"context,omitempty"`
	Quality            *processing.QualityConfig     `json:"quality,omitempty"`
	Transcription      string                        `json:"transcription,omitempty"`
	Diarization        string                        `json:"diarization,omitempty"`
	RecognitionModelID string                        `json:"recognition_model_id,omitempty"`
	DiarizationModelID string                        `json:"diarization_model_id,omitempty"`
	SubtitleFormat     string                        `json:"subtitle_format,omitempty"`
	Audio              processing.AudioOptions       `json:"audio,omitempty"`
	Recognition        processing.RecognitionOptions `json:"recognition,omitempty"`
	Attribution        processing.DiarizationOptions `json:"attribution,omitempty"`
}
type recordingPayload struct {
	RequestDigest  string             `json:"request_digest,omitempty"`
	Tools          *ProcessingTools   `json:"tools,omitempty"`
	Election       *recordingElection `json:"election,omitempty"`
	ModelDigests   map[string]string  `json:"model_digests,omitempty"`
	MediaID        string             `json:"media_id"`
	Expected       int64              `json:"expected_revision"`
	SourceRevision int64              `json:"source_revision"`
	Options        RecordingOptions   `json:"options"`
	AssemblyDigest string             `json:"assembly_digest,omitempty"`
}
type AssemblyInput struct {
	SubtitleFormat string            `json:"subtitle_format,omitempty"`
	Turns          []processing.Turn `json:"turns"`
	Participation  []struct {
		CueID string `json:"cue_id"`
		Label string `json:"label"`
	} `json:"participation,omitempty"`
}
type subtitleEngine interface {
	Ingest(context.Context, []byte, string) (json.RawMessage, error)
	Assemble(context.Context, []byte, *int64, []subtitles.Turn, []subtitles.Participation) (subtitles.Assembly, error)
	Export(context.Context, []byte, string, bool) (subtitles.Export, error)
}
type audioExecution struct {
	Path        string
	SourceMap   any
	Provenance  any
	Diagnostics any
	Recognize   func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error)
	Diarize     func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error)
	Close       func() error
}
type recordingExecution struct {
	Subtitles subtitleEngine
	Prepare   func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error)
}

func readBoundedConfiguration(path string, out any) error {
	f, e := os.Open(path)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 {
		return contracts.Fail("output_limit")
	}
	if schemas.ValidateDocument(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	return strictPayload(raw, out)
}
func ValidateProcessingTools(c ProcessingTools) error {
	if c.Kind != "processing-tools" || c.Version != contracts.Version {
		return contracts.Fail("incompatible_version")
	}
	if e := processing.ValidateConfig(c.Processing); e != nil {
		return e
	}
	if _, e := subtitles.New(c.Cueson); e != nil {
		return e
	}
	return nil
}
func (a *App) recordingExecutor() (*recordingExecution, error) {
	return a.recordingExecutorWithTools(nil)
}
func (a *App) electedProcessingTools() (*ProcessingTools, error) {
	if a.recordingFactory != nil {
		return nil, nil
	}
	c, e := ReadProcessingTools(a.Workspace)
	if e != nil {
		return nil, e
	}
	if c.Processing.FFmpeg.Path == "" {
		lib, e := a.libraryService()
		if e != nil {
			return nil, e
		}
		c.Processing.FFmpeg = lib.Tools.FFmpeg
	}
	return &c, nil
}
func (a *App) recordingExecutorWithTools(elected *ProcessingTools) (*recordingExecution, error) {
	if a.recordingFactory != nil {
		return a.recordingFactory()
	}
	if elected == nil {
		var e error
		elected, e = a.electedProcessingTools()
		if e != nil {
			return nil, e
		}
	}
	c := *elected
	if e := ValidateProcessingTools(c); e != nil {
		return nil, e
	}
	driver, e := subtitles.New(c.Cueson)
	if e != nil {
		return nil, e
	}
	lib, e := a.libraryService()
	if e != nil {
		return nil, e
	}
	engine := &processing.Service{Artifacts: lib.Artifacts, Catalog: a.Catalog, Config: c.Processing}
	return &recordingExecution{Subtitles: driver, Prepare: func(ctx context.Context, entry catalog.LibraryEntry, options processing.AudioOptions) (*audioExecution, error) {
		s, e := engine.Prepare(ctx, entry, options)
		if e != nil {
			return nil, e
		}
		return &audioExecution{Path: s.MappedPath, SourceMap: s.SourceMap, Provenance: s.Provenance, Diagnostics: s.Diagnostics, Recognize: s.Recognize, Diarize: s.Diarize, Close: s.Close}, nil
	}}, nil
}
func validateRecordingOptions(o RecordingOptions) error {
	if o.PipelineRevision < 0 || o.PipelineID != "" && !contracts.ValidID(o.PipelineID) || o.PipelineRevision != 0 && o.PipelineID == "" {
		return contracts.Fail("invalid_request")
	}
	if o.Quality != nil && processing.ValidateQuality(*o.Quality) != nil {
		return contracts.Fail("invalid_request")
	}
	if o.Transcription != "" && o.Transcription != "supplied" && o.Transcription != "generate" && o.Transcription != "reuse" {
		return contracts.Fail("invalid_request")
	}
	if o.Diarization != "" && o.Diarization != "run" && o.Diarization != "reuse" {
		return contracts.Fail("invalid_request")
	}
	if o.Transcription == "reuse" && o.Diarization == "reuse" {
		return contracts.Fail("invalid_request")
	}
	if o.PipelineID == "" && o.Transcription == "generate" && !contracts.ValidID(o.RecognitionModelID) {
		return contracts.Fail("invalid_request")
	}
	if o.PipelineID == "" && o.Diarization != "reuse" && !contracts.ValidID(o.DiarizationModelID) {
		return contracts.Fail("invalid_request")
	}
	if o.RecognitionModelID != "" && !contracts.ValidID(o.RecognitionModelID) {
		return contracts.Fail("invalid_request")
	}
	if o.DiarizationModelID != "" && !contracts.ValidID(o.DiarizationModelID) {
		return contracts.Fail("invalid_request")
	}
	if o.SubtitleFormat != "" && o.SubtitleFormat != "srt" && o.SubtitleFormat != "vtt" {
		return contracts.Fail("invalid_request")
	}
	if o.Audio.Channel != nil && *o.Audio.Channel < 0 || o.Audio.StreamIndex != nil && *o.Audio.StreamIndex < 0 {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func documentHash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func durationNS(e catalog.LibraryEntry) (*int64, error) {
	if e.DurationUS == nil {
		return nil, nil
	}
	if *e.DurationUS > math.MaxInt64/1000 {
		return nil, contracts.Fail("invalid_request")
	}
	n := *e.DurationUS * 1000
	return &n, nil
}
func (a *App) subtitleInput(ctx context.Context, entry catalog.LibraryEntry) ([]byte, error) {
	if entry.SubtitlePublicationID == nil {
		return nil, contracts.Fail("not_found")
	}
	s, e := a.artifactService()
	if e != nil {
		return nil, e
	}
	m, e := s.MaterializeBound(ctx, *entry.SubtitlePublicationID, 8<<20)
	if e != nil {
		return nil, e
	}
	defer func() {
		clean, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer stop()
		s.Release(clean, m.PublicationID, m.Lease.ID)
		s.Prune(clean, m.PublicationID)
	}()
	f, e := os.Open(m.Path)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, (8<<20)+1))
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	if len(raw) > 8<<20 {
		return nil, contracts.Fail("output_limit")
	}
	return raw, nil
}
func nativeFormat(raw []byte, elected string) string {
	if elected != "" {
		return elected
	}
	// WebVTT permits an optional UTF-8 BOM at the start. Inspect a sliced view
	// without changing the source passed to Cueson or retained as source evidence.
	probe := bytes.TrimSpace(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf}))
	if bytes.HasPrefix(probe, []byte("WEBVTT")) {
		return "vtt"
	}
	return "srt"
}
func localTurns(recordingID, workID string, turns []processing.Turn) ([]subtitles.Turn, error) {
	out := []subtitles.Turn{}
	if len(turns) > 65536 {
		return nil, contracts.Fail("output_limit")
	}
	for _, t := range turns {
		if strings.TrimSpace(t.Label) == "" || len(t.Label) > 256 || t.StartUS < 0 || t.EndUS <= t.StartUS || t.EndUS > math.MaxInt64/1000 {
			return nil, contracts.Fail("invalid_request")
		}
		out = append(out, subtitles.Turn{SpeakerID: library.DerivedID(recordingID, workID+"/speaker/"+t.Label), StartNS: t.StartUS * 1000, EndNS: t.EndUS * 1000})
	}
	return out, nil
}
func (a *App) processRecording(ctx context.Context, claim catalog.Work) (any, error) {
	var input recordingPayload
	if strictPayload(claim.Payload, &input) != nil || !contracts.ValidID(input.MediaID) || validateRecordingOptions(input.Options) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if validateElection(input) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if result, accepted, e := a.Catalog.AcceptedRecordingWork(ctx, claim.ID, input.MediaID); e != nil {
		return nil, e
	} else if accepted {
		var value any
		json.Unmarshal(result, &value)
		return value, nil
	}
	entry, e := a.Catalog.Library(ctx, input.MediaID)
	if e != nil {
		return nil, e
	}
	if entry.Revision != input.SourceRevision {
		return nil, contracts.Fail("conflict")
	}
	previous, previousErr := a.Catalog.Recording(ctx, entry.ID)
	if input.Expected == 0 && previousErr == nil || input.Expected != 0 && (previousErr != nil || previous.Revision != input.Expected) {
		return nil, contracts.Fail("conflict")
	}
	execution, e := a.recordingExecutorWithTools(input.Tools)
	if e != nil {
		return nil, e
	}
	phase := func(name string) error {
		_, e := a.Catalog.CheckpointWork(ctx, claim, name, "running", json.RawMessage(`{"processing":true}`), leaseTTL)
		return e
	}
	if e = phase("mapped-audio"); e != nil {
		return nil, e
	}
	session, e := execution.Prepare(ctx, entry, input.Options.Audio)
	if e != nil {
		return nil, e
	}
	defer session.Close()
	if input.Options.Diarization == "reuse" {
		if previousErr != nil || compatibleReuseMap(previous.SourceMap, session.SourceMap) != nil {
			return nil, contracts.Fail("conflict")
		}
	}
	provenance := map[string]any{"audio": session.Provenance}
	diagnostics := []any{session.Diagnostics}
	var doc json.RawMessage
	state := "ready"
	mode := input.Options.Transcription
	if mode == "" {
		if entry.SubtitlePublicationID != nil || previousErr == nil && previous.State == "ready" {
			mode = "supplied"
		} else {
			mode = "generate"
		}
	}
	switch mode {
	case "reuse":
		if previousErr != nil || previous.State != "ready" {
			return nil, contracts.Fail("not_found")
		}
		doc = previous.Document
		var prior map[string]json.RawMessage
		json.Unmarshal(previous.Provenance, &prior)
		provenance["transcription"] = prior["transcription"]
		provenance["transcription_reuse_digest"] = previous.DocumentDigest
	case "supplied":
		doc, e = a.suppliedDocument(ctx, entry, execution.Subtitles, input.Options.SubtitleFormat)
		if e != nil {
			return nil, e
		}
		provenance["transcription"] = map[string]any{"mode": "supplied", "publication_id": entry.SubtitlePublicationID}
	case "generate":
		if e = phase("recognition"); e != nil {
			return nil, e
		}
		result, e := a.electedRecognize(ctx, session, input)
		if e != nil {
			return nil, e
		}
		provenance["transcription"] = result.Provenance
		diagnostics = append(diagnostics, result.Diagnostics)
		if result.NoSpeech {
			state = "no-speech"
		} else if len(bytes.TrimSpace(result.SRT)) == 0 {
			state = "no-timed-subtitles"
		} else {
			doc, e = execution.Subtitles.Ingest(ctx, result.SRT, "srt")
			if e != nil {
				return nil, e
			}
		}
	}
	var turns []subtitles.Turn
	var participation []subtitles.Participation
	if input.Options.Diarization == "reuse" {
		if previousErr != nil || previous.State != "ready" {
			return nil, contracts.Fail("not_found")
		}
		turns, participation, e = currentTurns(previous.Document)
		if e != nil {
			return nil, e
		}
		quality, err := reusedQuality(turns, session.SourceMap, input.Options.Attribution.Quality)
		if err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, quality)
		var reuseDiagnostics []subtitles.Diagnostic
		participation, reuseDiagnostics, e = reuseParticipation(previous.Document, doc, participation)
		if e != nil {
			return nil, e
		}
		diagnostics = append(diagnostics, reuseDiagnostics)
		provenance["diarization"] = map[string]any{"mode": "reuse", "document_digest": previous.DocumentDigest, "quality_basis": "current-embedded-millisecond-assignments", "quality": input.Options.Attribution.Quality}
	} else {
		if e = phase("diarization"); e != nil {
			return nil, e
		}
		result, e := a.electedDiarize(ctx, session, input)
		if e != nil {
			return nil, e
		}
		provenance["diarization"] = result.Provenance
		diagnostics = append(diagnostics, result.Diagnostics)
		turns, e = localTurns(entry.ID, claim.ID, result.Turns)
		if e != nil {
			return nil, e
		}
		if result.NoSpeech && len(doc) == 0 {
			state = "no-speech"
		}
		if !result.NoSpeech && len(doc) == 0 {
			state = "no-timed-subtitles"
		}
	}
	if len(doc) > 0 {
		usable, err := hasTimedCues(doc)
		if err != nil {
			return nil, err
		}
		if !usable {
			doc = nil
			state = "no-timed-subtitles"
		}
	} else if state == "ready" {
		state = "no-timed-subtitles"
	}
	if e = phase("assemble"); e != nil {
		return nil, e
	}
	final := json.RawMessage("null")
	digest := ""
	if len(doc) > 0 {
		duration, e := durationNS(entry)
		if e != nil {
			return nil, e
		}
		assembled, e := execution.Subtitles.Assemble(ctx, doc, duration, turns, participation)
		if e != nil {
			return nil, e
		}
		final = assembled.Document
		digest = documentHash(final)
		diagnostics = append(diagnostics, assembled.Diagnostics)
		state = "ready"
	}
	// Publish mapped audio only after all stages have produced an acceptable candidate.
	artifacts, e := a.artifactService()
	if e != nil {
		return nil, e
	}
	publicationID := library.DerivedID(claim.ID, "mapped-audio")
	pub, e := artifacts.Publish(ctx, publicationID, session.Path, "mapped-audio")
	if e != nil {
		return nil, e
	}
	committed := false
	defer func() {
		if !committed {
			clean, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer stop()
			a.Catalog.QueueDerivedCleanup(clean, library.DerivedID(claim.ID, "candidate-cleanup"), entry.ID, publicationID)
		}
	}()
	record := catalog.Recording{ID: entry.ID, SourceRevision: entry.Revision, SourceDigest: entry.Digest, State: state, Document: final, DocumentDigest: digest, MappedAudioPublicationID: &pub.ID}
	if input.Election != nil {
		provenance["pipeline_election"] = input.Election
	}
	record.SourceMap, _ = json.Marshal(session.SourceMap)
	record.Provenance, _ = json.Marshal(provenance)
	record.Diagnostics = flattenDiagnostics(diagnostics)
	current, e := a.Catalog.CommitRecording(ctx, claim, input.Expected, record)
	if e != nil {
		return nil, e
	}
	committed = true
	lib, e := a.libraryService()
	if e == nil {
		lib.Cleanup(ctx, entry.ID)
	}
	return map[string]any{"recording_id": current.ID, "revision": current.Revision, "document_digest": current.DocumentDigest, "state": current.State, "quality_diagnostics": current.Diagnostics}, nil
}
func flattenDiagnostics(groups []any) json.RawMessage {
	out := []map[string]json.RawMessage{}
	counts := []int64{}
	indices := map[string]int{}
	for _, group := range groups {
		raw, e := json.Marshal(group)
		if e != nil {
			continue
		}
		var items []map[string]json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			continue
		}
		for _, item := range items {
			var code string
			json.Unmarshal(item["code"], &code)
			if code == "" {
				continue
			}
			var count int64
			json.Unmarshal(item["count"], &count)
			if count < 1 {
				count = 1
			}
			if index, exists := indices[code]; exists {
				if count > math.MaxInt64-counts[index] {
					counts[index] = math.MaxInt64
				} else {
					counts[index] += count
				}
				continue
			}
			indices[code] = len(out)
			out = append(out, item)
			counts = append(counts, count)
		}
	}
	for index := range out {
		out[index]["count"], _ = json.Marshal(counts[index])
	}
	raw, _ := json.Marshal(out)
	return raw
}

// Assemble supplied source subtitles with ephemeral adapter results. Durable work
// records contain only identities, never the request's speaker turn list.
func (a *App) assembleRecording(req contracts.Request) (any, error) {
	var input AssemblyInput
	if strictPayload(req.Data, &input) != nil || len(input.Turns) > 65536 || len(input.Participation) > 65536 {
		return nil, contracts.Fail("invalid_request")
	}
	entry, e := a.Catalog.Library(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	previous, prevErr := a.Catalog.Recording(a.ctx, entry.ID)
	expected := int64(0)
	if prevErr == nil {
		expected = previous.Revision
	}
	canonical, _ := json.Marshal(input)
	assemblyDigest := documentHash(canonical)
	if old, err := a.Catalog.Work(a.ctx, req.RequestID); err == nil {
		var original recordingPayload
		if old.Kind != "recordings.assemble" || strictPayload(old.Payload, &original) != nil || original.MediaID != entry.ID || original.AssemblyDigest != assemblyDigest || original.SourceRevision != entry.Revision {
			return nil, contracts.Fail("conflict")
		}
		expected = original.Expected
	}
	payload, _ := json.Marshal(recordingPayload{MediaID: entry.ID, Expected: expected, SourceRevision: entry.Revision, AssemblyDigest: assemblyDigest})
	// Replayed requests reconcile their accepted receipt before choosing a new CAS.
	if result, accepted, e := a.Catalog.AcceptedRecordingWork(a.ctx, req.RequestID, entry.ID); e != nil {
		return nil, e
	} else if accepted {
		var value any
		json.Unmarshal(result, &value)
		return value, nil
	}
	work, e := a.Catalog.EnqueueWork(a.ctx, req.RequestID, "recordings.assemble", payload)
	if e != nil {
		return nil, e
	}
	if work.State == "failed" || work.State == "interrupted" {
		work, e = a.Catalog.RetryWork(a.ctx, contracts.ID(), work.ID)
		if e != nil {
			return nil, e
		}
	}
	claim, e := a.Catalog.ClaimWork(a.ctx, work.ID, a.Session, 30*time.Second)
	if e != nil {
		return nil, e
	}
	ctx, stop := context.WithTimeout(a.ctx, 25*time.Second)
	defer stop()
	result, e := a.assembleClaim(ctx, claim, entry, expected, input)
	if e != nil {
		_, _ = a.Catalog.CheckpointWork(context.WithoutCancel(a.ctx), claim, "assembly-failed", "failed", json.RawMessage(`{"state":"failed","error":"operation_failed","retry":"resubmit-ephemeral-input"}`), leaseTTL)
		return nil, e
	}
	raw, _ := json.Marshal(result)
	_, e = a.Catalog.CheckpointWork(ctx, claim, "complete", "succeeded", raw, leaseTTL)
	return result, e
}
func (a *App) assembleClaim(ctx context.Context, claim catalog.Work, entry catalog.LibraryEntry, expected int64, input AssemblyInput) (any, error) {
	execution, e := a.recordingExecutor()
	if e != nil {
		return nil, e
	}
	doc, e := a.suppliedDocument(ctx, entry, execution.Subtitles, input.SubtitleFormat)
	if e != nil {
		return nil, e
	}
	turns, e := localTurns(entry.ID, claim.ID, input.Turns)
	if e != nil {
		return nil, e
	}
	participation := []subtitles.Participation{}
	for _, p := range input.Participation {
		if p.CueID == "" || strings.TrimSpace(p.Label) == "" || len(p.Label) > 256 {
			return nil, contracts.Fail("invalid_request")
		}
		participation = append(participation, subtitles.Participation{CueID: p.CueID, SpeakerID: library.DerivedID(entry.ID, claim.ID+"/speaker/"+p.Label)})
	}
	duration, e := durationNS(entry)
	if e != nil {
		return nil, e
	}
	usable, e := hasTimedCues(doc)
	if e != nil {
		return nil, e
	}
	assembled := subtitles.Assembly{Document: json.RawMessage("null"), Diagnostics: []subtitles.Diagnostic{}}
	state, digest := "no-timed-subtitles", ""
	if usable {
		assembled, e = execution.Subtitles.Assemble(ctx, doc, duration, turns, participation)
		if e != nil {
			return nil, e
		}
		state, digest = "ready", documentHash(assembled.Document)
	}
	record := catalog.Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: entry.Revision, State: state, Document: assembled.Document, DocumentDigest: digest, SourceMap: json.RawMessage(`{"policy":"supplied-subtitle-source-clock"}`), Provenance: json.RawMessage(`{"transcription":"supplied","diarization":"supplied-ephemeral-results"}`)}
	record.Diagnostics = flattenDiagnostics([]any{assembled.Diagnostics})
	current, e := a.Catalog.CommitRecording(ctx, claim, expected, record)
	if e != nil {
		return nil, e
	}
	lib, e := a.libraryService()
	if e == nil {
		lib.Cleanup(ctx, entry.ID)
	}
	return map[string]any{"recording_id": current.ID, "revision": current.Revision, "document_digest": current.DocumentDigest, "state": current.State, "diagnostics": current.Diagnostics}, nil
}
func (a *App) recordingDispatch(req contracts.Request) (any, error) {
	switch req.Operation {
	case "recordings.assemble":
		return a.assembleRecording(req)
	case "recordings.process":
		var options RecordingOptions
		if strictPayload(req.Data, &options) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		requestDigest := documentHash(configurationBytes(options))
		if old, err := a.Catalog.Work(a.ctx, req.RequestID); err == nil {
			var original recordingPayload
			if old.Kind != "recordings.process" || strictPayload(old.Payload, &original) != nil || original.MediaID != req.ItemID {
				return nil, contracts.Fail("conflict")
			}
			if original.RequestDigest != "" {
				if original.RequestDigest != requestDigest {
					return nil, contracts.Fail("conflict")
				}
				return workView(old), nil
			}
		}
		entry, e := a.Catalog.Library(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		if options.Transcription == "" {
			if current, err := a.Catalog.Recording(a.ctx, entry.ID); entry.SubtitlePublicationID != nil || err == nil && current.State == "ready" {
				options.Transcription = "supplied"
			} else {
				options.Transcription = "generate"
			}
		}
		if options.Diarization == "" {
			options.Diarization = "run"
		}
		options, election, e := a.electRecordingOptions(options)
		if e != nil {
			return nil, e
		}
		if e = validateRecordingOptions(options); e != nil {
			return nil, e
		}
		previous, prevErr := a.Catalog.Recording(a.ctx, req.ItemID)
		expected := int64(0)
		if prevErr == nil {
			expected = previous.Revision
		}
		if options.Transcription == "reuse" || options.Diarization == "reuse" {
			if prevErr != nil || previous.State != "ready" {
				return nil, contracts.Fail("not_found")
			}
		}
		// Fixed payload from the first election keeps retry/CAS intent stable.
		if old, e := a.Catalog.Work(a.ctx, req.RequestID); e == nil {
			var p recordingPayload
			if old.Kind != "recordings.process" || strictPayload(old.Payload, &p) != nil || p.MediaID != req.ItemID {
				return nil, contracts.Fail("conflict")
			}
			elected, _ := json.Marshal(options)
			was, _ := json.Marshal(p.Options)
			if !bytes.Equal(elected, was) {
				return nil, contracts.Fail("conflict")
			}
			return workView(old), nil
		}
		tools, e := a.electedProcessingTools()
		if e != nil {
			return nil, e
		}
		modelDigests, e := a.electModelDigests(options)
		if e != nil {
			return nil, e
		}
		raw, _ := json.Marshal(recordingPayload{RequestDigest: requestDigest, Tools: tools, Election: election, ModelDigests: modelDigests, MediaID: req.ItemID, Expected: expected, SourceRevision: entry.Revision, Options: options})
		work, e := a.Catalog.EnqueueWork(a.ctx, req.RequestID, "recordings.process", raw)
		if e == nil {
			a.recoverWork()
		}
		return workView(work), e
	case "recordings.show":
		r, e := a.Catalog.Recording(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		mappings, e := a.Catalog.SpeakerMappings(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		raw, _ := json.Marshal(r)
		if len(raw) > 512<<10 || len(mappings) > 100 {
			return map[string]any{"kind": "recording", "schema_version": contracts.Version, "recording_id": r.ID, "revision": r.Revision, "state": r.State, "document_digest": r.DocumentDigest, "document_inline": false, "document_operation": "recordings.document", "mappings_operation": "recordings.mappings"}, nil
		}
		return map[string]any{"kind": "recording", "schema_version": contracts.Version, "recording": r, "mappings": mappings}, nil
	case "recordings.document":
		var page struct {
			Offset int    `json:"offset,omitempty"`
			Limit  int    `json:"limit,omitempty"`
			Digest string `json:"document_digest"`
		}
		if len(req.Data) > 0 && strictPayload(req.Data, &page) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if page.Limit == 0 {
			page.Limit = 64 << 10
		}
		if page.Offset < 0 || page.Limit < 1 || page.Limit > 64<<10 {
			return nil, contracts.Fail("invalid_request")
		}
		r, e := a.Catalog.Recording(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		if page.Digest != "" && page.Digest != r.DocumentDigest {
			return nil, contracts.Fail("conflict")
		}
		if page.Offset > len(r.Document) {
			return nil, contracts.Fail("invalid_request")
		}
		end := min(page.Offset+page.Limit, len(r.Document))
		return map[string]any{"document_digest": r.DocumentDigest, "revision": r.Revision, "encoding": "base64", "data": base64.StdEncoding.EncodeToString(r.Document[page.Offset:end]), "offset": page.Offset, "next_offset": end, "complete": end == len(r.Document)}, nil
	case "recordings.mappings":
		p, e := pageInput(req.Data)
		if e != nil {
			return nil, e
		}
		mappings, e := a.Catalog.SpeakerMappings(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		return pageViews(mappings, p, func(m catalog.SpeakerMapping) string { return m.ID }, func(m catalog.SpeakerMapping) any { return m }), nil
	case "recordings.map-speaker":
		var input struct {
			Revision        int64  `json:"revision"`
			MappingRevision int64  `json:"mapping_revision,omitempty"`
			LocalSpeakerID  string `json:"local_speaker_id"`
			SpeakerID       string `json:"speaker_id"`
			DocumentDigest  string `json:"document_digest"`
		}
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.SetSpeakerMapping(a.ctx, req.RequestID, input.Revision, catalog.SpeakerMapping{Revision: input.MappingRevision, RecordingID: req.ItemID, LocalSpeakerID: input.LocalSpeakerID, SpeakerID: input.SpeakerID, DocumentDigest: input.DocumentDigest})
	case "recordings.resolve-segment":
		var input struct {
			SegmentID string `json:"segment_id"`
			Revision  int64  `json:"segment_revision"`
		}
		if strictPayload(req.Data, &input) != nil || !contracts.ValidID(input.SegmentID) || input.Revision < 1 {
			return nil, contracts.Fail("invalid_request")
		}
		resolved, e := a.Catalog.ResolveSegment(a.ctx, input.SegmentID, input.Revision)
		if e != nil {
			return nil, e
		}
		if resolved.Reference.RecordingID != req.ItemID {
			return nil, contracts.Fail("not_found")
		}
		return resolved, nil
	case "recordings.export":
		return a.exportRecording(req)
	}
	return nil, contracts.Fail("invalid_request")
}
func (a *App) exportRecording(req contracts.Request) (any, error) {
	var input struct {
		Format         string `json:"format"`
		Strict         bool   `json:"strict,omitempty"`
		Destination    string `json:"destination"`
		DocumentDigest string `json:"document_digest,omitempty"`
	}
	if strictPayload(req.Data, &input) != nil || !filepath.IsAbs(input.Destination) || (input.Format != "cueson" && input.Format != "srt" && input.Format != "vtt") {
		return nil, contracts.Fail("invalid_request")
	}
	r, e := a.Catalog.Recording(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	if r.State != "ready" {
		return nil, contracts.Fail("not_found")
	}
	if input.DocumentDigest != "" && input.DocumentDigest != r.DocumentDigest {
		return nil, contracts.Fail("conflict")
	}
	execution, e := a.recordingExecutor()
	if e != nil {
		return nil, e
	}
	output, e := execution.Subtitles.Export(a.ctx, r.Document, input.Format, input.Strict)
	if e != nil {
		return nil, contracts.WithDiagnostics(e, subtitleDiagnostics(output.Diagnostics))
	}
	// An explicit external export creates a new file; never overwrite a source or
	// managed result. A same-directory temporary and exclusive hard link publish
	// the complete validated bytes atomically on supported filesystems.
	dir := filepath.Dir(input.Destination)
	f, e := os.CreateTemp(dir, ".insonic-export-*")
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	temp := f.Name()
	defer os.Remove(temp)
	defer f.Close()
	if _, e = f.Write(output.Bytes); e != nil {
		return nil, contracts.Fail("unavailable")
	}
	if e = f.Sync(); e != nil {
		return nil, contracts.Fail("unavailable")
	}
	if e = f.Close(); e != nil {
		return nil, contracts.Fail("unavailable")
	}
	latest, e := a.Catalog.Recording(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	if latest.DocumentDigest != r.DocumentDigest {
		return nil, contracts.Fail("conflict")
	}
	if e = os.Link(temp, input.Destination); e != nil {
		if os.IsExist(e) {
			return nil, contracts.Fail("conflict")
		}
		return nil, contracts.Fail("unavailable")
	}
	return map[string]any{"path": input.Destination, "bytes": len(output.Bytes), "document_digest": r.DocumentDigest, "diagnostics": contracts.AggregateDiagnostics(subtitleDiagnostics(output.Diagnostics))}, nil
}

func recordingRequestValid(req contracts.Request) bool {
	if !contracts.ValidID(req.ItemID) || req.JobID != "" || req.DurationMS != 0 || req.AfterGeneration != 0 {
		return false
	}
	switch req.Operation {
	case "recordings.roster.show":
		return len(req.Data) == 0
	case "recordings.roster.add", "recordings.roster.remove", "recordings.roster.replace", "recordings.roster.clear":
		return len(req.Data) > 0
	case "recordings.show":
		return len(req.Data) == 0
	case "recordings.document", "recordings.mappings":
		return true
	case "recordings.process", "recordings.assemble", "recordings.map-speaker", "recordings.export", "recordings.resolve-segment":
		return len(req.Data) > 0
	}
	return false
}
func (a *App) recoverDerivedCleanup() {
	ctx, stop := context.WithTimeout(a.ctx, 10*time.Second)
	defer stop()
	work, e := a.Catalog.Works(ctx)
	if e != nil {
		return
	}
	for _, w := range work {
		if w.Kind != "recordings.process" || (w.State != "failed" && w.State != "cancelled" && w.State != "succeeded") {
			continue
		}
		var p recordingPayload
		if strictPayload(w.Payload, &p) != nil {
			continue
		}
		id := library.DerivedID(w.ID, "mapped-audio")
		current, e := a.Catalog.Recording(ctx, p.MediaID)
		if e == nil && current.MappedAudioPublicationID != nil && *current.MappedAudioPublicationID == id {
			continue
		}
		a.Catalog.QueueDerivedCleanup(ctx, library.DerivedID(w.ID, "candidate-cleanup"), p.MediaID, id)
	}
	lib, e := a.libraryService()
	if e != nil {
		return
	}
	cleanups, e := a.Catalog.Cleanups(ctx)
	if e != nil {
		return
	}
	seen := map[string]bool{}
	for _, c := range cleanups {
		if c.State == "pending" && !seen[c.EntryID] {
			seen[c.EntryID] = true
			lib.Cleanup(ctx, c.EntryID)
		}
	}
}

func hasTimedCues(document []byte) (bool, error) {
	if len(document) == 0 {
		return false, nil
	}
	cues, e := subtitles.DocumentCues(document)
	if e != nil {
		return false, e
	}
	for _, cue := range cues {
		if cue.EndMS > cue.StartMS {
			return true, nil
		}
	}
	return false, nil
}
func subtitleDiagnostics(input []subtitles.Diagnostic) []contracts.Diagnostic {
	output := make([]contracts.Diagnostic, len(input))
	for i, item := range input {
		output[i] = contracts.Diagnostic{Code: item.Code, Pointer: item.Pointer, Message: item.Message, Count: 1}
	}
	return output
}

func (a *App) suppliedDocument(ctx context.Context, entry catalog.LibraryEntry, engine subtitleEngine, format string) (json.RawMessage, error) {
	if entry.SubtitlePublicationID == nil {
		current, err := a.Catalog.Recording(ctx, entry.ID)
		if err != nil {
			return nil, err
		}
		if current.State != "ready" || current.SourceDigest != entry.Digest {
			return nil, contracts.Fail("not_found")
		}
		if err = subtitles.ValidateDocument(current.Document); err != nil {
			return nil, err
		}
		return current.Document, nil
	}
	raw, err := a.subtitleInput(ctx, entry)
	if err != nil {
		return nil, err
	}
	return engine.Ingest(ctx, raw, nativeFormat(raw, format))
}
