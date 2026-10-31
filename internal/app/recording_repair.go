// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
)

type RecordingRepairOptions struct {
	ExpectedRevision int64                       `json:"expected_revision"`
	SourceRevision   int64                       `json:"source_revision"`
	DocumentDigest   string                      `json:"document_digest"`
	Automatic        bool                        `json:"automatic,omitempty"`
	Ranges           []processing.TrackSelection `json:"ranges,omitempty"`
	Policy           *processing.RepairPolicy    `json:"policy,omitempty"`
	Coverage         *processing.CoverageOptions `json:"coverage,omitempty"`
	Processing       RecordingOptions            `json:"processing,omitempty"`
}
type recordingRepairElection struct {
	Fence          catalog.RecordingRepairFence `json:"fence"`
	Policy         processing.RepairPolicy      `json:"policy"`
	Coverage       processing.CoverageOptions   `json:"coverage"`
	EvidenceDigest string                       `json:"evidence_digest"`
	Selection      processing.RepairSelection   `json:"selection"`
	PlanDigest     string                       `json:"plan_digest,omitempty"`
}

func recordingWindowIdentity(p recordingPayload, entry catalog.LibraryEntry, previous catalog.Recording) processing.PlanIdentity {
	return processing.PlanIdentity{SourceID: entry.ID, SourceDigest: entry.Digest, SourceRevision: entry.Revision, DocumentDigest: previous.DocumentDigest, DocumentRevision: p.Expected, ConfigurationDigest: windowConfigurationDigest(p), ModelDigest: documentHash(configurationBytes(p.ModelDigests)), AdapterID: "insonic-windowed-processing", AdapterVersion: "1", DevicePolicy: p.Options.Recognition.Device + "/" + p.Options.Attribution.Device}
}
func (a *App) queueRecordingRepair(req contracts.Request) (any, error) {
	var request RecordingRepairOptions
	if strictPayload(req.Data, &request) != nil || request.ExpectedRevision < 1 || request.SourceRevision < 1 || request.Automatic == (len(request.Ranges) > 0) {
		return nil, contracts.Fail("invalid_request")
	}
	digest := documentHash(configurationBytes(request))
	if old, e := a.Catalog.Work(a.ctx, req.RequestID); e == nil {
		var p recordingPayload
		if old.Kind != "recordings.repair" || strictPayload(old.Payload, &p) != nil || p.MediaID != req.ItemID || p.RequestDigest != digest {
			return nil, contracts.Fail("conflict")
		}
		return a.modelWorkView(old), nil
	}
	previous, e := a.Catalog.Recording(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	entry, e := a.Catalog.Library(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	if previous.Revision != request.ExpectedRevision || previous.DocumentDigest != request.DocumentDigest || previous.SourceRevision != request.SourceRevision || entry.Revision != request.SourceRevision || entry.Digest != previous.SourceDigest {
		return nil, contracts.Condition("conflict", "revision_stale")
	}
	basis, e := recordingCoverageInput(previous)
	if e != nil {
		return nil, e
	}
	if len(basis.Tracks) == 0 {
		return nil, contracts.Condition("conflict", "source_map_incompatible")
	}
	options := request.Processing
	if options.Transcription != "" && options.Transcription != "generate" {
		return nil, contracts.Fail("invalid_request")
	}
	options.Transcription = "generate"
	if options.Diarization == "" {
		options.Diarization = "run"
	}
	options, election, e := a.electRecordingOptions(options)
	if e != nil {
		return nil, e
	}
	if election != nil {
		if request.Policy == nil {
			request.Policy = election.Definition.Repair
		}
		if request.Coverage == nil {
			request.Coverage = election.Definition.Coverage
		}
	}
	policy := processing.RepairPolicy{}
	if request.Policy != nil {
		policy = *request.Policy
	}
	policy, e = processing.NormalizeRepairPolicy(policy)
	if e != nil {
		return nil, e
	}
	coverage := processing.CoverageOptions{}
	if request.Coverage != nil {
		coverage = *request.Coverage
	}
	if coverage.After != 0 {
		return nil, contracts.Fail("invalid_request")
	}
	coverage, e = processing.NormalizeCoverageOptions(coverage)
	if e != nil {
		return nil, e
	}
	// The accepted lanes and native clock are immutable. Only window size/context
	// can be elected for a repair; target ranges are a separate bounded selection.
	if options.Audio.StreamIndex != nil || options.Audio.Channel != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if options.Windows == nil {
		var provenance struct {
			Plan struct {
				ID string `json:"id"`
			} `json:"processing_plan"`
		}
		if json.Unmarshal(previous.Provenance, &provenance) == nil && provenance.Plan.ID != "" {
			if old, err := a.Catalog.ProcessingPlan(a.ctx, provenance.Plan.ID); err == nil {
				var plan processing.ProcessingPlan
				if json.Unmarshal(old.Plan, &plan) == nil {
					options.Windows = &plan.Options
				}
			}
		}
	}
	windows := processing.WindowOptions{}
	if options.Windows != nil {
		windows = *options.Windows
	}
	if len(windows.Tracks) > 0 {
		if len(windows.Tracks) != len(basis.Tracks) {
			return nil, contracts.Condition("conflict", "source_map_incompatible")
		}
		for i, s := range windows.Tracks {
			t := basis.Tracks[i]
			if s.StreamIndex != t.StreamIndex || (s.Channel == nil) != (t.Channel == nil) || s.Channel != nil && *s.Channel != *t.Channel || s.StartSample != t.StartSample || s.EndSample != nil && *s.EndSample != t.EndSample {
				return nil, contracts.Condition("conflict", "source_map_incompatible")
			}
		}
	}
	windows.Tracks = nil
	for _, t := range basis.Tracks {
		end := t.EndSample
		windows.Tracks = append(windows.Tracks, processing.TrackSelection{StreamIndex: t.StreamIndex, Channel: t.Channel, StartSample: t.StartSample, EndSample: &end})
	}
	windows, e = processing.NormalizeWindowOptions(windows)
	if e != nil {
		return nil, e
	}
	options.Windows = &windows
	if e = validateRecordingOptions(options); e != nil {
		return nil, e
	}
	report, e := processing.EvaluateCoverage(basis, coverage)
	if e != nil {
		return nil, e
	}
	targets := []processing.RepairTarget{}
	if request.Automatic {
		page := report
		for {
			for _, r := range page.Regions {
				if !r.Uncertain && (r.Reason == "uncovered_speech" || r.Reason == "repeated_text" || r.Reason == "boundary_inconsistency") {
					targets = append(targets, processing.RepairTarget{TrackOrdinal: r.TrackOrdinal, CoverageSpan: r.CoverageSpan})
				}
			}
			if len(targets) > 32 {
				return nil, contracts.Fail("output_limit")
			}
			if page.Next == nil {
				break
			}
			next := coverage
			next.After = *page.Next
			next.Limit = 1000
			page, e = processing.EvaluateCoverage(basis, next)
			if e != nil {
				return nil, e
			}
		}
	} else {
		targets, e = processing.RepairTargetsFromRanges(basis.Tracks, request.Ranges)
		if e != nil {
			return nil, e
		}
	}
	mappings, e := a.Catalog.SpeakerMappings(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	repair := &recordingRepairElection{Fence: catalog.RecordingRepairFence{ExpectedRevision: previous.Revision, DocumentDigest: previous.DocumentDigest, SourceRevision: entry.Revision, SourceDigest: entry.Digest, MappingDigest: catalog.SpeakerMappingsDigest(mappings)}, Policy: policy, Coverage: coverage, EvidenceDigest: report.EvidenceDigest, Selection: processing.RepairSelection{Targets: targets, Scopes: []processing.RepairTarget{}, Windows: []processing.RepairWindow{}, RemovedCueIDs: []string{}}}
	tools, e := a.electedProcessingTools()
	if e != nil {
		return nil, e
	}
	options, election, models, e := a.electProcessingModels(options, election)
	if e != nil {
		return nil, e
	}
	p := recordingPayload{Repair: repair, RequestDigest: digest, Tools: tools, Election: election, ModelSelections: models, MediaID: req.ItemID, Expected: previous.Revision, SourceRevision: entry.Revision, Options: options}
	if len(models) > 0 {
		p.ModelDigests = map[string]string{}
	}
	for _, m := range models {
		p.ModelDigests[m.Target.ID] = m.Digest
	}
	if len(targets) > 0 {
		plan, err := processing.BuildWindowPlan(recordingWindowIdentity(p, entry, previous), windows, basis.Tracks)
		if err != nil {
			return nil, err
		}
		repair.Selection, err = processing.NormalizeRepairSelection(plan, basis.Cues, targets, policy)
		if err != nil {
			return nil, err
		}
		// Normalization may merge targets; rebuild identity once with that stable set.
		plan, err = processing.BuildWindowPlan(recordingWindowIdentity(p, entry, previous), windows, basis.Tracks)
		if err != nil {
			return nil, err
		}
		repair.Selection, err = processing.NormalizeRepairSelection(plan, basis.Cues, repair.Selection.Targets, policy)
		if err != nil {
			return nil, err
		}
		repair.PlanDigest = plan.Digest
		p.ModelDependencies, e = a.queueModelSelections(a.ctx, models)
		if e != nil {
			return nil, e
		}
	}
	work, e := a.Catalog.EnqueueWork(a.ctx, req.RequestID, "recordings.repair", configurationBytes(p))
	if e == nil {
		a.recoverWork()
	}
	return a.modelWorkView(work), e
}

func (a *App) processRecordingRepair(ctx context.Context, claim catalog.Work) (any, error) {
	var p recordingPayload
	if strictPayload(claim.Payload, &p) != nil || p.Repair == nil || !contracts.ValidID(p.MediaID) {
		return nil, contracts.Fail("invalid_request")
	}
	if result, ok, e := a.Catalog.RecordingRepairDecision(ctx, claim.ID, p.MediaID); e != nil {
		return nil, e
	} else if ok {
		return result, nil
	}
	if p.Options.Windows == nil || p.Options.Transcription != "generate" || validateRecordingOptions(p.Options) != nil || validateElection(p) != nil || validateModelElection(p) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	entry, e := a.Catalog.Library(ctx, p.MediaID)
	if e != nil {
		return nil, e
	}
	previous, e := a.Catalog.Recording(ctx, p.MediaID)
	if e != nil {
		return nil, e
	}
	f := p.Repair.Fence
	mappings, e := a.Catalog.SpeakerMappings(ctx, p.MediaID)
	if e != nil {
		return nil, e
	}
	if previous.Revision != f.ExpectedRevision || previous.DocumentDigest != f.DocumentDigest || previous.SourceRevision != f.SourceRevision || previous.SourceDigest != f.SourceDigest || entry.Revision != f.SourceRevision || entry.Digest != f.SourceDigest || catalog.SpeakerMappingsDigest(mappings) != f.MappingDigest {
		return nil, contracts.Condition("conflict", "revision_stale")
	}
	basis, e := recordingCoverageInput(previous)
	if e != nil {
		return nil, e
	}
	before, e := processing.EvaluateCoverage(basis, p.Repair.Coverage)
	if e != nil {
		return nil, e
	}
	if before.EvidenceDigest != p.Repair.EvidenceDigest {
		return nil, contracts.Fail("conflict")
	}
	finish := func(candidate *catalog.Recording, after processing.CoverageReport, reason string) (any, error) {
		attempts, err := a.Catalog.RepairWindowAttempts(ctx, claim.ID)
		if err != nil {
			return nil, err
		}
		status := "unchanged"
		candidateDigest := ""
		if candidate != nil {
			status = "accepted"
			candidateDigest = candidate.DocumentDigest
		}
		decision := map[string]any{"kind": "recording-repair", "schema_version": contracts.Version, "status": status, "reason": reason, "work_id": claim.ID, "source_revision": f.SourceRevision, "source_sha256": f.SourceDigest, "previous_document_sha256": f.DocumentDigest, "evidence_sha256": before.EvidenceDigest, "options_sha256": before.OptionsDigest, "selection": p.Repair.Selection, "policy": p.Repair.Policy, "plan_digest": p.Repair.PlanDigest, "attempts": attempts, "before": before.Tracks, "after": after.Tracks, "candidate_sha256": candidateDigest}
		out, err := a.Catalog.CommitRecordingRepair(ctx, claim, f, candidate, configurationBytes(decision))
		if err != nil {
			return nil, err
		}
		if err = a.Catalog.RecoverProcessingWindowCleanup(ctx, claim.ID); err != nil {
			return nil, err
		}
		if lib, err := a.libraryCatalogService(); err == nil {
			lib.Cleanup(ctx, p.MediaID)
		}
		return out, nil
	}
	if len(p.Repair.Selection.Windows) == 0 {
		return finish(nil, before, "no_repairable_regions")
	}
	if e = a.verifyElectedModels(ctx, p.ModelSelections); e != nil {
		return nil, e
	}
	selection := &windowCandidateSelection{Select: func(plan processing.ProcessingPlan) (map[string]bool, error) {
		if plan.Digest != p.Repair.PlanDigest {
			return nil, contracts.Condition("conflict", "source_map_incompatible")
		}
		normalized, err := processing.NormalizeRepairSelection(plan, basis.Cues, p.Repair.Selection.Targets, p.Repair.Policy)
		if err != nil {
			return nil, err
		}
		if documentHash(configurationBytes(normalized)) != documentHash(configurationBytes(p.Repair.Selection)) {
			return nil, contracts.Fail("conflict")
		}
		chosen := map[string]bool{}
		for _, w := range normalized.Windows {
			chosen[w.ID] = true
		}
		return chosen, nil
	}, BeforeAttempt: func(ctx context.Context, w processing.WindowPlan) error {
		_, err := a.Catalog.ConsumeRepairWindowAttempt(ctx, claim, catalog.ProcessingPlanID(claim.ID, p.Repair.PlanDigest), w.ID)
		return err
	}}
	built, e := a.buildWindowedRecordingCandidate(ctx, claim, p, entry, previous, selection)
	if e != nil {
		return nil, e
	}
	candidate := built.Record
	if candidate.State != "ready" {
		return finish(nil, before, "candidate_has_no_timed_cues")
	}
	generated, e := recordingCoverageInput(candidate)
	if e != nil {
		return nil, e
	}
	chosen := []string{}
	for _, c := range generated.Cues {
		if processing.RepairContains(p.Repair.Selection.Scopes, c.TrackOrdinal, c.StartUS, c.EndUS) {
			chosen = append(chosen, c.ID)
			continue
		}
		for _, scope := range p.Repair.Selection.Scopes {
			if scope.TrackOrdinal == c.TrackOrdinal && c.StartUS < scope.EndUS && c.EndUS > scope.StartUS {
				return finish(nil, before, "candidate_crosses_scope_boundary")
			}
		}
	}
	if len(chosen) == 0 {
		return finish(nil, before, "candidate_has_no_in_scope_cues")
	}
	executor, e := a.recordingExecutorWithTools(p.Tools)
	if e != nil {
		return nil, e
	}
	cueIDs := map[string]string{}
	if previous.State == "ready" {
		merged, err := subtitles.MergeRepairDocument(ctx, executor.Subtitles, previous.Document, candidate.Document, p.Repair.Selection.RemovedCueIDs, chosen, "repair-"+claim.ID)
		if err != nil {
			return nil, err
		}
		candidate.Document = merged.Document
		cueIDs = merged.CueIDs
	} else {
		// No accepted timed cues exist. Candidate output still must stay entirely in scope.
		if len(chosen) != len(generated.Cues) {
			return finish(nil, before, "candidate_crosses_scope_boundary")
		}
		for _, id := range chosen {
			cueIDs[id] = id
		}
	}
	candidate.DocumentDigest = documentHash(candidate.Document)
	candidate.SourceMap = previous.SourceMap
	candidate.MappedAudioPublicationID = previous.MappedAudioPublicationID
	if e = mergeRepairProvenance(previous, &candidate, p, cueIDs); e != nil {
		return nil, e
	}
	afterInput, e := recordingCoverageInput(candidate)
	if e != nil {
		return nil, e
	}
	after, e := processing.EvaluateCoverage(afterInput, p.Repair.Coverage)
	if e != nil {
		return nil, e
	}
	improved, reason := processing.CompareCoverage(before, after)
	if !improved {
		return finish(nil, after, reason)
	}
	return finish(&candidate, after, reason)
}
func mergeRepairProvenance(previous catalog.Recording, candidate *catalog.Recording, p recordingPayload, ids map[string]string) error {
	var old, next map[string]json.RawMessage
	if json.Unmarshal(previous.Provenance, &old) != nil || json.Unmarshal(candidate.Provenance, &next) != nil {
		return contracts.Fail("invalid_request")
	}
	var previousLinks, newLinks recordingCoverageProvenance
	if json.Unmarshal(previous.Provenance, &previousLinks) != nil || json.Unmarshal(candidate.Provenance, &newLinks) != nil {
		return contracts.Fail("invalid_request")
	}
	removed := map[string]bool{}
	for _, id := range p.Repair.Selection.RemovedCueIDs {
		removed[id] = true
	}
	links := []recordingCueSource{}
	for _, s := range previousLinks.CueSources {
		if !removed[s.CueID] {
			links = append(links, s)
		}
	}
	for _, s := range newLinks.CueSources {
		if id, ok := ids[s.CueID]; ok {
			s.CueID = id
			links = append(links, s)
		}
	}
	var document struct {
		Cues []struct {
			ID      string `json:"id"`
			Ordinal int    `json:"ordinal"`
		} `json:"cues"`
	}
	if json.Unmarshal(candidate.Document, &document) != nil {
		return contracts.Fail("invalid_request")
	}
	ordinals := map[string]int{}
	for _, cue := range document.Cues {
		ordinals[cue.ID] = cue.Ordinal
	}
	for i := range links {
		ordinal, ok := ordinals[links[i].CueID]
		if !ok {
			return contracts.Fail("invalid_request")
		}
		links[i].Source.Ordinal = ordinal
	}
	voices := append([]windowVoiceSource{}, previousLinks.VoiceSources...)
	seen := map[string]bool{}
	for _, s := range voices {
		seen[s.LocalSpeakerID] = true
	}
	for _, s := range newLinks.VoiceSources {
		if !seen[s.LocalSpeakerID] {
			voices = append(voices, s)
			seen[s.LocalSpeakerID] = true
		}
	}
	old["cue_sources"] = configurationBytes(links)
	old["voice_sources"] = configurationBytes(voices)
	// Acoustic evidence is the fixed comparison basis. Candidate claims cannot
	// erase an original speech denominator or manufacture measured improvement.
	old["repair"] = configurationBytes(map[string]any{"plan_digest": p.Repair.PlanDigest, "selection": p.Repair.Selection, "pipeline_election": p.Election, "candidate_provenance": next})
	candidate.Provenance = configurationBytes(old)
	return nil
}
