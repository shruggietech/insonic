// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
)

type windowExecution struct {
	Plan   processing.ProcessingPlan
	Window func(context.Context, int) (*audioExecution, error)
	Finish func(context.Context) error
	Close  func() error
}

func electWindowOptions(o RecordingOptions, entry catalog.LibraryEntry) (RecordingOptions, error) {
	// This runs only during new admission. Existing payloads without windows
	// retain the historical whole-recording election on replay and retry.
	if o.Windows == nil && entry.DurationUS != nil && *entry.DurationUS > 30_000_000 {
		o.Windows = &processing.WindowOptions{}
	}
	if o.Windows == nil {
		return o, nil
	}
	normalized, err := processing.NormalizeWindowOptions(*o.Windows)
	if err != nil {
		return o, err
	}
	if len(normalized.Tracks) > 0 && (o.Audio.StreamIndex != nil || o.Audio.Channel != nil) {
		return o, contracts.Fail("invalid_request")
	}
	if len(normalized.Tracks) == 0 {
		var facts library.Facts
		if json.Unmarshal(entry.Facts, &facts) == nil {
			for _, stream := range facts.Streams {
				if stream.Kind == "audio" && (o.Audio.StreamIndex == nil || stream.Index == *o.Audio.StreamIndex) {
					normalized.Tracks = []processing.TrackSelection{{StreamIndex: stream.Index, Channel: o.Audio.Channel}}
					break
				}
			}
		}
		if len(normalized.Tracks) == 0 && (o.Audio.StreamIndex != nil || o.Audio.Channel != nil) {
			return o, contracts.Fail("unsupported_audio")
		}
	}
	o.Windows = &normalized
	return o, nil
}

// Host bindings may change after portable restore; their pinned bytes and all
// semantic election fields remain identity-bearing.
func windowConfigurationDigest(p recordingPayload) string {
	fields := map[string]any{"options": p.Options, "election": p.Election, "models": p.ModelDigests, "tools": p.Tools}
	if p.Repair != nil {
		fields["repair"] = []any{p.Repair.Fence, p.Repair.Policy, p.Repair.Coverage, p.Repair.EvidenceDigest, p.Repair.Selection.Targets}
	}
	raw := configurationBytes(fields)
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return ""
	}
	var scrub func(any)
	scrub = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, item := range x {
				if k == "path" || k == "executable" || k == "portable_files" || k == "package_companions" {
					delete(x, k)
				} else {
					scrub(item)
				}
			}
		case []any:
			for _, item := range x {
				scrub(item)
			}
		}
	}
	scrub(value)
	return documentHash(configurationBytes(value))
}

type processingWindowOutput struct {
	SchemaVersion    string                       `json:"schema_version"`
	Kind             string                       `json:"kind"`
	Version          int                          `json:"version"`
	PlanDigest       string                       `json:"plan_digest"`
	WindowID         string                       `json:"window_id"`
	Recognition      processing.RecognitionResult `json:"recognition"`
	Diarization      processing.DiarizationResult `json:"diarization"`
	AudioProvenance  any                          `json:"audio_provenance"`
	AudioDiagnostics json.RawMessage              `json:"audio_diagnostics"`
}

const maxWindowResultBytes = 16 << 20

func (a *App) loadWindowCheckpoint(ctx context.Context, c catalog.ProcessingWindowCheckpoint, planDigest, windowID string) (processingWindowOutput, error) {
	var output processingWindowOutput
	if c.State != "ready" || c.OutputSize < 1 || c.OutputSize > maxWindowResultBytes {
		return output, contracts.Fail("invalid_request")
	}
	artifacts, e := a.artifactService()
	if e != nil {
		return output, e
	}
	m, e := artifacts.MaterializeBound(ctx, c.PublicationID, c.OutputSize)
	if e != nil {
		return output, e
	}
	defer func() {
		clean, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer stop()
		artifacts.Release(clean, m.PublicationID, m.Lease.ID)
		artifacts.Prune(clean, m.PublicationID)
	}()
	f, e := os.Open(m.Path)
	if e != nil {
		return output, contracts.Fail("unavailable")
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, c.OutputSize+1))
	if e != nil || int64(len(raw)) != c.OutputSize || documentHash(raw) != c.OutputDigest {
		return output, contracts.Fail("conflict")
	}
	if strictPayload(raw, &output) != nil || output.Kind != "processing-window" || output.Version != 1 || output.SchemaVersion != contracts.Version || output.PlanDigest != planDigest || output.WindowID != windowID {
		return output, contracts.Fail("conflict")
	}
	return output, nil
}

func (a *App) saveWindowCheckpoint(ctx context.Context, claim catalog.Work, input recordingPayload, plan catalog.ProcessingPlan, window processing.WindowPlan, output processingWindowOutput) (catalog.ProcessingWindowCheckpoint, error) {
	raw, e := json.Marshal(output)
	if e != nil || len(raw) > maxWindowResultBytes {
		return catalog.ProcessingWindowCheckpoint{}, contracts.Fail("output_limit")
	}
	artifacts, e := a.artifactService()
	if e != nil {
		return catalog.ProcessingWindowCheckpoint{}, e
	}
	dir, e := os.MkdirTemp(a.Workspace.Control, "processing-")
	if e != nil {
		return catalog.ProcessingWindowCheckpoint{}, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(dir)
	if e = workspace.SecureDirectory(dir, true); e != nil {
		return catalog.ProcessingWindowCheckpoint{}, e
	}
	// The existing exclusive-runtime recovery recognizes this owned scratch.
	if e = os.WriteFile(dir+string(os.PathSeparator)+".insonic-processing-session", []byte("insonic-processing-scratch-v1\n"), 0600); e != nil {
		return catalog.ProcessingWindowCheckpoint{}, e
	}
	f, e := os.CreateTemp(dir, "window-*.json")
	if e != nil {
		return catalog.ProcessingWindowCheckpoint{}, contracts.Fail("unavailable")
	}
	_, e = f.Write(raw)
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil || closeErr != nil {
		return catalog.ProcessingWindowCheckpoint{}, contracts.Fail("unavailable")
	}
	publicationID := catalog.ProcessingWindowPublicationID(claim.ID, plan.PlanDigest, window.ID, claim.Generation)
	var pub catalog.Publication
	if input.Destination != nil {
		pub, e = artifacts.PublishTo(ctx, *input.Destination, publicationID, f.Name(), "processing-window")
	} else {
		pub, e = artifacts.Publish(ctx, publicationID, f.Name(), "processing-window")
	}
	if e != nil {
		return catalog.ProcessingWindowCheckpoint{}, e
	}
	c := catalog.ProcessingWindowCheckpoint{ID: catalog.ProcessingCheckpointID(plan.ID, window.ID, "combined"), PlanID: plan.ID, WorkID: claim.ID, WindowID: window.ID, Stage: "combined", Attempt: claim.Generation, PublicationID: pub.ID, OutputDigest: documentHash(raw), OutputSize: int64(len(raw)), State: "ready"}
	c, e = a.Catalog.CommitProcessingWindowCheckpoint(ctx, claim, c)
	if e != nil {
		clean, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer stop()
		a.Catalog.QueueDerivedCleanup(clean, library.DerivedID(pub.ID, "uncommitted-window-cleanup"), input.MediaID, pub.ID)
	}
	return c, e
}

type windowCandidateSelection struct {
	Select        func(processing.ProcessingPlan) (map[string]bool, error)
	BeforeAttempt func(context.Context, processing.WindowPlan) error
}
type windowRecordingCandidate struct {
	Record           catalog.Recording
	Plan             processing.ProcessingPlan
	PlanDigest       string
	CompletedWindows int
	Notices          []string
}

func (a *App) processWindowedRecording(ctx context.Context, claim catalog.Work, input recordingPayload, entry catalog.LibraryEntry, previous catalog.Recording) (any, error) {
	candidate, e := a.buildWindowedRecordingCandidate(ctx, claim, input, entry, previous, nil)
	if e != nil {
		return nil, e
	}
	current, e := a.Catalog.CommitRecording(ctx, claim, input.Expected, candidate.Record)
	if e != nil {
		return nil, e
	}
	if lib, err := a.libraryCatalogService(); err == nil {
		lib.Cleanup(ctx, entry.ID)
	}
	return map[string]any{"recording_id": current.ID, "revision": current.Revision, "document_digest": current.DocumentDigest, "state": current.State, "quality_diagnostics": current.Diagnostics, "plan_digest": candidate.PlanDigest, "completed_windows": candidate.CompletedWindows, "notices": candidate.Notices}, nil
}
func (a *App) buildWindowedRecordingCandidate(ctx context.Context, claim catalog.Work, input recordingPayload, entry catalog.LibraryEntry, previous catalog.Recording, selection *windowCandidateSelection) (*windowRecordingCandidate, error) {
	mode := input.Options.Transcription
	if mode == "" {
		if entry.SubtitlePublicationID != nil || previous.State == "ready" {
			mode = "supplied"
		} else {
			mode = "generate"
		}
	}
	reuseTranscript := mode == "reuse" || mode == "supplied" && entry.SubtitlePublicationID == nil
	suppliedTimeline := mode == "supplied" && entry.SubtitlePublicationID != nil || reuseTranscript && suppliedWindowTranscript(previous)
	execution, e := a.recordingExecutorWithTools(input.Tools)
	if e != nil {
		return nil, e
	}
	if execution.PrepareWindows == nil {
		return nil, contracts.Fail("unsupported_capability")
	}
	identity := recordingWindowIdentity(input, entry, previous)
	windows, e := execution.PrepareWindows(ctx, entry, *input.Options.Windows, identity)
	if e != nil {
		return nil, e
	}
	defer windows.Close()
	selected := map[string]bool{}
	for _, w := range windows.Plan.Windows {
		selected[w.ID] = true
	}
	if selection != nil && selection.Select != nil {
		selected, e = selection.Select(windows.Plan)
		if e != nil {
			return nil, e
		}
	}
	selectedCount, lastSelected := 0, -1
	for i, w := range windows.Plan.Windows {
		if selected[w.ID] {
			selectedCount++
			lastSelected = i
		}
	}
	if selectedCount == 0 {
		return nil, contracts.Fail("invalid_request")
	}
	if reuseTranscript || input.Options.Diarization == "reuse" {
		if previous.State != "ready" {
			return nil, contracts.Fail("not_found")
		}
		if input.Options.Diarization == "reuse" {
			e = compatibleWindowReuse(previous, windows.Plan)
		} else {
			e = compatibleWindowTranscriptReuse(previous, windows.Plan)
		}
		if e != nil {
			return nil, e
		}
	}
	planRaw, e := json.Marshal(windows.Plan)
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	plan := catalog.ProcessingPlan{ID: catalog.ProcessingPlanID(claim.ID, windows.Plan.Digest), WorkID: claim.ID, MediaID: entry.ID, SourceRevision: entry.Revision, SourceDigest: entry.Digest, ExpectedRevision: input.Expected, DocumentDigest: previous.DocumentDigest, ConfigurationDigest: identity.ConfigurationDigest, PlanDigest: windows.Plan.Digest, Plan: planRaw, Attempt: claim.Generation, State: "active"}
	plan, e = a.Catalog.CommitProcessingPlan(ctx, claim, plan)
	if e != nil {
		return nil, e
	}
	checkpoints, e := a.Catalog.ProcessingWindowCheckpoints(ctx, plan.ID)
	if e != nil {
		return nil, e
	}
	ready := map[string]catalog.ProcessingWindowCheckpoint{}
	for _, c := range checkpoints {
		if c.State == "ready" && c.Stage == "combined" {
			ready[c.WindowID] = c
		}
	}
	var evidence []processing.WindowEvidence
	diagnostics := []any{}
	totalBytes := 0
	for index, window := range windows.Plan.Windows {
		if !selected[window.ID] {
			continue
		}
		if ctx.Err() != nil {
			return nil, contracts.Fail("cancelled")
		}
		progress, _ := json.Marshal(map[string]any{"state": "processing", "plan_digest": plan.PlanDigest, "completed_windows": len(evidence), "total_windows": selectedCount, "active_window_id": window.ID})
		if _, e = a.Catalog.CheckpointWork(ctx, claim, "window-processing", "running", progress, leaseTTL); e != nil {
			return nil, e
		}
		var output processingWindowOutput
		var checkpoint catalog.ProcessingWindowCheckpoint
		if c, ok := ready[window.ID]; ok {
			checkpoint = c
			output, e = a.loadWindowCheckpoint(ctx, c, plan.PlanDigest, window.ID)
			if e != nil {
				return nil, e
			}
		} else {
			session, err := windows.Window(ctx, index)
			if err != nil {
				return nil, err
			}
			output = processingWindowOutput{SchemaVersion: contracts.Version, Kind: "processing-window", Version: 1, PlanDigest: plan.PlanDigest, WindowID: window.ID, Recognition: processing.RecognitionResult{NoSpeech: true, Provenance: map[string]any{}}, Diarization: processing.DiarizationResult{NoSpeech: true, Provenance: map[string]any{}}, AudioProvenance: session.Provenance, AudioDiagnostics: flattenDiagnostics([]any{session.Diagnostics})}
			if input.Options.Diarization == "reuse" {
				output.Diarization.NoSpeech = false
				output.Diarization.Provenance = map[string]any{"mode": "reuse"}
			}
			for {
				if selection != nil && selection.BeforeAttempt != nil {
					if e = selection.BeforeAttempt(ctx, window); e != nil {
						break
					}
				}
				if mode == "generate" {
					output.Recognition, e = a.electedRecognize(ctx, session, input)
				}
				if e == nil && input.Options.Diarization != "reuse" {
					output.Diarization, e = a.electedDiarize(ctx, session, input)
				}
				if e == nil || selection == nil || selection.BeforeAttempt == nil || ctx.Err() != nil {
					break
				}
				code := contracts.SafeError(e).Code
				if code != "unavailable" && code != "operation_failed" && code != "invalid_engine_output" {
					break
				}
			}
			closeErr := session.Close()
			if e != nil {
				return nil, e
			}
			if closeErr != nil {
				return nil, closeErr
			}
			if index == lastSelected && windows.Finish != nil {
				if e = windows.Finish(ctx); e != nil {
					return nil, e
				}
			}
			if e = processing.ValidateWindowEvidence(processing.WindowEvidence{Window: window, OutputDigest: documentHash(configurationBytes(output)), Recognition: output.Recognition, Diarization: output.Diarization}); e != nil {
				return nil, e
			}
			checkpoint, e = a.saveWindowCheckpoint(ctx, claim, input, plan, window, output)
			if e != nil {
				return nil, e
			}
		}
		totalBytes += int(checkpoint.OutputSize)
		if totalBytes > 64<<20 {
			return nil, contracts.Fail("output_limit")
		}
		evidence = append(evidence, processing.WindowEvidence{Window: window, OutputDigest: checkpoint.OutputDigest, Recognition: output.Recognition, Diarization: output.Diarization})
		diagnostics = append(diagnostics, output.AudioDiagnostics)
	}
	if windows.Finish != nil {
		if e = windows.Finish(ctx); e != nil {
			return nil, e
		}
	}
	stitched, e := processing.ReconcileWindows(evidence)
	if e != nil {
		return nil, e
	}
	diagnostics = append(diagnostics, stitched.Diagnostics)
	var doc json.RawMessage
	state := "ready"
	provenance := map[string]any{"window_observations": stitched.Recognition.Provenance["window_observations"], "processing_plan": map[string]any{"id": plan.ID, "digest": plan.PlanDigest, "window_count": len(windows.Plan.Windows)}, "reconciliation_digest": stitched.Digest, "transcription": stitched.Recognition.Provenance}
	delete(stitched.Recognition.Provenance, "window_observations")
	switch mode {
	case "generate":
		if stitched.Recognition.NoSpeech {
			state = "no-speech"
		} else if len(bytes.TrimSpace(stitched.Recognition.SRT)) == 0 {
			state = "no-timed-subtitles"
		} else {
			doc, e = execution.Subtitles.Ingest(ctx, stitched.Recognition.SRT, "srt")
		}
	case "supplied":
		doc, e = a.suppliedDocument(ctx, entry, execution.Subtitles, input.Options.SubtitleFormat)
		provenance["transcription"] = map[string]any{"mode": "supplied", "publication_id": entry.SubtitlePublicationID}
	case "reuse":
		doc = previous.Document
		provenance["transcription"] = map[string]any{"mode": "reuse", "document_digest": previous.DocumentDigest}
	default:
		return nil, contracts.Fail("invalid_request")
	}
	if e != nil {
		return nil, e
	}
	cueTracks := map[string]int{}
	voiceTracks := map[string]int{}
	voiceSources := []windowVoiceSource{}
	var prior struct {
		VoiceSources []windowVoiceSource `json:"voice_sources"`
		CueSources   []struct {
			CueID  string               `json:"cue_id"`
			Source processing.CueSource `json:"source"`
		} `json:"cue_sources"`
	}
	json.Unmarshal(previous.Provenance, &prior)
	if reuseTranscript && !suppliedTimeline {
		provenance["cue_sources"] = prior.CueSources
		for _, c := range prior.CueSources {
			cueTracks[c.CueID] = c.Source.TrackOrdinal
		}
	}
	if mode == "generate" && len(doc) > 0 {
		cues, err := subtitles.DocumentCues(doc)
		if err != nil {
			return nil, err
		}
		if len(cues) != len(stitched.CueSources) {
			return nil, contracts.Fail("invalid_timing")
		}
		for i, c := range cues {
			cueTracks[c.ID] = stitched.CueSources[i].TrackOrdinal
		}
	}
	var turns []subtitles.Turn
	var participation []subtitles.Participation
	if input.Options.Diarization == "reuse" {
		if previous.State != "ready" {
			return nil, contracts.Fail("not_found")
		}
		voiceSources = prior.VoiceSources
		turns, participation, e = currentTurns(previous.Document)
		if len(windows.Plan.Tracks) == 1 && len(voiceSources) == 0 {
			seen := map[string]bool{}
			for _, turn := range turns {
				if !seen[turn.SpeakerID] {
					voiceSources = append(voiceSources, windowVoiceSource{turn.SpeakerID, 0})
					seen[turn.SpeakerID] = true
				}
			}
			for _, p := range participation {
				if !seen[p.SpeakerID] {
					voiceSources = append(voiceSources, windowVoiceSource{p.SpeakerID, 0})
					seen[p.SpeakerID] = true
				}
			}
		}
		for _, s := range voiceSources {
			voiceTracks[s.LocalSpeakerID] = s.TrackOrdinal
		}
		if e == nil {
			for lane, track := range windows.Plan.Tracks {
				selected := []subtitles.Turn{}
				for _, turn := range turns {
					source, ok := voiceTracks[turn.SpeakerID]
					if !ok {
						return nil, contracts.Condition("conflict", "source_map_incompatible")
					}
					if source == lane {
						selected = append(selected, turn)
					}
				}
				quality, err := reusedQuality(selected, windowTrackMapping(track), input.Options.Attribution.Quality)
				if err != nil {
					return nil, err
				}
				diagnostics = append(diagnostics, quality)
			}
		}
		if e == nil && input.Repair == nil {
			var reuseDiagnostics []subtitles.Diagnostic
			participation, reuseDiagnostics, e = reuseParticipation(previous.Document, doc, participation)
			diagnostics = append(diagnostics, reuseDiagnostics)
		}
		if input.Repair != nil {
			participation = nil
		}
		provenance["diarization"] = map[string]any{"mode": "reuse", "document_digest": previous.DocumentDigest}
	} else {
		turns, e = localTurns(entry.ID, claim.ID, stitched.Turns)
		for _, s := range stitched.VoiceSources {
			id := library.DerivedID(entry.ID, claim.ID+"/speaker/"+s.Label)
			voiceSources = append(voiceSources, windowVoiceSource{id, s.TrackOrdinal})
			voiceTracks[id] = s.TrackOrdinal
		}
		provenance["diarization"] = map[string]any{"mode": "window-local-voices", "window_count": len(evidence)}
	}
	if e != nil {
		return nil, e
	}
	provenance["voice_sources"] = voiceSources
	final := json.RawMessage("null")
	digest := ""
	if len(doc) > 0 {
		usable, err := hasTimedCues(doc)
		if err != nil {
			return nil, err
		}
		if usable {
			duration, err := processing.WindowSourceDurationNS(windows.Plan)
			if err != nil {
				return nil, err
			}
			var assembled subtitles.Assembly
			if suppliedTimeline {
				assembled, err = assembleSuppliedWindowDocument(ctx, execution.Subtitles, doc, turns, participation, cueTracks, voiceTracks)
			} else {
				assembled, err = assembleWindowDocument(ctx, execution.Subtitles, doc, duration, turns, participation, cueTracks, voiceTracks)
			}
			if err != nil {
				return nil, err
			}
			final = assembled.Document
			digest = documentHash(final)
			state = "ready"
			diagnostics = append(diagnostics, assembled.Diagnostics)
		} else {
			state = "no-timed-subtitles"
		}
	} else if state == "ready" {
		state = "no-timed-subtitles"
	}
	if mode == "generate" && state == "ready" {
		var document struct {
			Cues []struct {
				ID string `json:"id"`
			} `json:"cues"`
		}
		if json.Unmarshal(final, &document) != nil || len(document.Cues) != len(stitched.CueSources) {
			return nil, contracts.Fail("invalid_timing")
		}
		links := make([]map[string]any, 0, len(document.Cues))
		for i, cue := range document.Cues {
			links = append(links, map[string]any{"cue_id": cue.ID, "source": stitched.CueSources[i]})
		}
		provenance["cue_sources"] = links
	}
	if input.Election != nil {
		provenance["pipeline_election"] = input.Election
	}
	provenance["model_elections"] = modelSelectionSummaries(input.ModelSelections)
	sourceMap, _ := json.Marshal(windowSourceMap{Policy: "windowed-canonical-source-clock-v1", SourceDigest: entry.Digest, Tracks: windows.Plan.Tracks, SuppliedDocument: suppliedTimeline})
	recognitionStatus, diarizationStatus := "unavailable", "executed"
	if mode == "generate" {
		recognitionStatus = "executed"
	}
	if input.Options.Diarization == "reuse" {
		diarizationStatus = "reused"
	}
	activity, err := processing.CoverageFromWindows(identity, documentHash(sourceMap), evidence, recognitionStatus, diarizationStatus)
	if err != nil {
		return nil, err
	}
	var priorActivity recordingCoverageProvenance
	if json.Unmarshal(previous.Provenance, &priorActivity) == nil {
		activity, err = processing.RetainReusedCoverage(activity, priorActivity.Evidence, recognitionStatus != "executed", diarizationStatus != "executed")
		if err != nil {
			return nil, err
		}
	}
	provenance["coverage_evidence"] = activity
	prov, _ := json.Marshal(provenance)
	record := catalog.Recording{ID: entry.ID, SourceRevision: entry.Revision, SourceDigest: entry.Digest, State: state, Document: final, DocumentDigest: digest, SourceMap: sourceMap, Provenance: prov, Diagnostics: flattenDiagnostics(diagnostics)}
	notices := []string{}
	seenNotices := map[string]bool{}
	for _, v := range evidence {
		for _, stage := range []map[string]any{v.Recognition.Provenance, v.Diarization.Provenance} {
			for _, notice := range processing.DeviceProvenanceNotices(stage) {
				if !seenNotices[notice] {
					notices = append(notices, notice)
					seenNotices[notice] = true
				}
			}
		}
	}
	if bytes.Contains(record.Diagnostics, []byte(`"recognition_diarization_disagreement"`)) {
		notices = append(notices, "Recognition returned text while diarization found no speech; the transcript is retained with a disagreement diagnostic.")
	}
	if bytes.Contains(record.Diagnostics, []byte(`"window_word_timing_unavailable"`)) {
		notices = append(notices, "Some windows lack usable word timing; segment timing was retained with boundary uncertainty.")
	}
	if bytes.Contains(record.Diagnostics, []byte(`"window_overlap_ambiguous"`)) {
		notices = append(notices, "Conflicting boundary observations were retained; inspect the recorded ambiguity diagnostics.")
	}
	return &windowRecordingCandidate{Record: record, Plan: windows.Plan, PlanDigest: plan.PlanDigest, CompletedWindows: len(evidence), Notices: notices}, nil
}
