// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAnswerDurableJourneyAndLinkedRetry(t *testing.T) {
	a, _, c := answerSynthesisApp(t)
	failed := true
	a.AnswerFixture = func(ctx context.Context, in assistance.Request, config assistance.Config) (assistance.Generated, error) {
		if failed {
			return assistance.Generated{}, contracts.Condition("unavailable", "provider_busy")
		}
		e, err := fixtureAnswer(in)
		raw, _ := json.Marshal(e)
		return assistance.Generated{Raw: raw, ObservedRoute: "local"}, err
	}
	input := AnswerWorkflowRequest{Request: answers.Request{Version: contracts.Version, Question: "What was said?"}, Configuration: &c}
	raw, _ := json.Marshal(input)
	req := contracts.Request{RequestID: contracts.ID(), Data: raw}
	first, err := a.requestAnswer(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	original := awaitWork(t, a, first.ID)
	if original.State != "failed" {
		t.Fatal(original)
	}
	failed = false
	retry, err := a.retryAnswer(context.Background(), contracts.ID(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retry.ID == first.ID {
		t.Fatal("retry overwrote history")
	}
	done := awaitWork(t, a, retry.ID)
	if done.State != "succeeded" {
		t.Fatal(string(done.Result), done.Error)
	}
	var payload answerPayload
	json.Unmarshal(done.Payload, &payload)
	if payload.RetryOf != first.ID || payload.Configuration.Provider.Model != "fixture" {
		t.Fatal(payload)
	}
	var progress answerProgress
	json.Unmarshal(done.Result, &progress)
	view, err := a.inspectGroundedAnswer(context.Background(), progress.AnswerID)
	if err != nil || !view.Accepted || view.Freshness != "current" || view.Answer.Answer.State != "partial" {
		t.Fatal(view, err)
	}
	bytes, _ := json.Marshal(view)
	for _, forbidden := range []string{"127.0.0.1", "credential_id", "source_path"} {
		if strings.Contains(string(bytes), forbidden) {
			t.Fatal("private export field", string(bytes))
		}
	}
	prior, _ := a.Catalog.Work(context.Background(), first.ID)
	if prior.State != "failed" {
		t.Fatal("prior attempt changed")
	}
	replay, err := a.requestAnswer(context.Background(), req)
	if err != nil || replay.ID != first.ID {
		t.Fatal("request replay changed election", err)
	}
	generic := realRequest(a, "work.retry", first.ID, nil)
	if generic.Error != nil {
		t.Fatal(generic.Error)
	}
	public, _ := json.Marshal(generic.Result)
	if strings.Contains(string(public), "127.0.0.1") || strings.Contains(string(public), "payload") {
		t.Fatal("generic retry exposed frozen private configuration", string(public))
	}
	id := generic.Result.(map[string]any)["work_id"].(string)
	if id == first.ID || awaitWork(t, a, id).State != "succeeded" {
		t.Fatal("generic linked retry failed")
	}
}

type lostAnswerAccept struct {
	catalog.Catalog
	lost bool
}

func (s *lostAnswerAccept) AcceptAnswer(ctx context.Context, w catalog.Work, c catalog.AnswerCandidate, r json.RawMessage) (catalog.Work, error) {
	out, err := s.Catalog.AcceptAnswer(ctx, w, c, r)
	if err == nil && !s.lost {
		s.lost = true
		return catalog.Work{}, contracts.Fail("unavailable")
	}
	return out, err
}
func TestAnswerUnknownCommitOutcomeReconciles(t *testing.T) {
	a, _, c := answerSynthesisApp(t)
	calls := 0
	a.AnswerFixture = func(ctx context.Context, in assistance.Request, config assistance.Config) (assistance.Generated, error) {
		calls++
		e, err := fixtureAnswer(in)
		raw, _ := json.Marshal(e)
		return assistance.Generated{Raw: raw, ObservedRoute: "local"}, err
	}
	lost := &lostAnswerAccept{Catalog: a.Catalog}
	a.Catalog = lost
	raw, _ := json.Marshal(AnswerWorkflowRequest{Request: answers.Request{Version: contracts.Version, Question: "q"}, Configuration: &c})
	work, err := a.requestAnswer(context.Background(), contracts.Request{RequestID: contracts.ID(), Data: raw})
	if err != nil {
		t.Fatal(err)
	}
	done := awaitWork(t, a, work.ID)
	if done.State != "succeeded" || !lost.lost || calls != 1 {
		t.Fatal(done, calls)
	}
}

func TestAnswerSourceReplacementDuringSynthesisAndPipelineReplay(t *testing.T) {
	a, _, c := answerSynthesisApp(t)
	pipelineID := contracts.ID()
	definition, _ := json.Marshal(map[string]any{"answer_synthesis": c})
	saved, err := a.Catalog.PutPipeline(context.Background(), contracts.ID(), 0, catalog.Pipeline{ID: pipelineID, Name: "Answer fixture", Preset: "local", Configuration: definition})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	a.AnswerFixture = func(ctx context.Context, in assistance.Request, config assistance.Config) (assistance.Generated, error) {
		calls++
		e, err := fixtureAnswer(in)
		if err != nil {
			return assistance.Generated{}, err
		}
		if calls == 1 {
			ref := e.Claims[0].Citations[0].Reference
			entry, err := a.Catalog.Library(ctx, ref.RecordingID)
			if err != nil {
				return assistance.Generated{}, err
			}
			current, err := a.Catalog.Recording(ctx, ref.RecordingID)
			if err != nil {
				return assistance.Generated{}, err
			}
			desktopRecording(t, a, entry, current.Revision, current.Document)
		}
		raw, _ := json.Marshal(e)
		return assistance.Generated{Raw: raw, ObservedRoute: "local"}, nil
	}
	raw, _ := json.Marshal(AnswerWorkflowRequest{Request: answers.Request{Version: contracts.Version, Question: "q"}, PipelineID: pipelineID, PipelineRevision: saved.Revision})
	req := contracts.Request{RequestID: contracts.ID(), Data: raw}
	first, err := a.requestAnswer(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	done := awaitWork(t, a, first.ID)
	if done.State != "failed" || calls != 1 {
		t.Fatal("stale source accepted or repaired", done, calls)
	}
	c.Provider.Model = "changed-default"
	saved.Configuration, _ = json.Marshal(map[string]any{"answer_synthesis": c})
	if _, err = a.Catalog.PutPipeline(context.Background(), contracts.ID(), saved.Revision, saved); err != nil {
		t.Fatal(err)
	}
	replay, err := a.requestAnswer(context.Background(), req)
	if err != nil || string(replay.Payload) != string(first.Payload) {
		t.Fatal("replay resolved changed default", err)
	}
	next, err := a.retryAnswer(context.Background(), contracts.ID(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	done = awaitWork(t, a, next.ID)
	if done.State != "succeeded" {
		t.Fatal(done)
	}
	var p answerPayload
	json.Unmarshal(done.Payload, &p)
	if p.Configuration.Provider.Model != "fixture" {
		t.Fatal("retry changed model", p)
	}
}
func TestAnswerRestartReconcilesCandidateWithoutProvider(t *testing.T) {
	a, p, c := answerSynthesisApp(t)
	// Bind the retained context to the exact frozen election used by durable work.
	request := p.Request
	request.Provider = synthesisProvider(c)
	prepared, err := a.prepareAnswer(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	p = prepared.Package
	payload := answerPayload{Request: p.Request, Configuration: c, RequestDigest: strings.Repeat("a", 64), PublicationID: contracts.ID()}
	raw, _ := json.Marshal(payload)
	id := contracts.ID()
	if _, err = a.Catalog.EnqueueWork(context.Background(), id, "answers.request", raw); err != nil {
		t.Fatal(err)
	}
	claim, err := a.Catalog.ClaimWork(context.Background(), id, a.Session, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := answers.SynthesisInput(p, nil)
	body, _ := json.Marshal(input)
	envelope, err := fixtureAnswer(assistance.Request{Context: body})
	if err != nil {
		t.Fatal(err)
	}
	result := answers.Grounded{Kind: answers.GroundedKind, Version: contracts.Version, ID: payload.PublicationID, WorkID: id, WorkspaceID: a.Workspace.Config.WorkspaceID, PackageID: p.ID, SourceEpoch: p.Observation.SourceEpoch, Answer: envelope, Provider: synthesisProvider(c), Route: "local", IndependentFamilies: p.IndependentFamilies, EvidenceCount: len(p.Items), Attempts: []answers.Attempt{}, Diagnostics: []string{}, Validation: "current-source-citation-authority;semantic-entailment-not-evaluated"}
	raw, _ = json.Marshal(result)
	progress := answerProgressFor(payload)
	progress.AnswerID = result.ID
	progress.PackageID = p.ID
	progress.Candidate = &catalog.AnswerCandidate{PublicationID: result.ID, PackageID: p.ID, Digest: documentHash(raw), Size: int64(len(raw)), SourceEpoch: p.Observation.SourceEpoch, MediaIDs: p.Request.Filters.MediaIDs}
	checkpoint, _ := json.Marshal(progress)
	if _, err = a.Catalog.CheckpointWork(context.Background(), claim, "publication", "running", checkpoint, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = a.saveGroundedAnswer(context.Background(), id, result.ID, raw); err != nil {
		t.Fatal(err)
	}
	// Simulate loss after byte publication but before the work reference persisted.
	if err = a.Catalog.ArtifactReference(context.Background(), result.ID, id, true); err != nil {
		t.Fatal(err)
	}
	w := a.Workspace
	a.Close()
	w.Config.Profiles.Graph.Adapter = "ladybugdb"
	b, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	completed := awaitWork(t, b, id)
	if completed.State != "succeeded" {
		t.Fatal("restart did not reconcile", completed)
	}
	view, err := b.inspectGroundedAnswer(context.Background(), result.ID)
	if err != nil || !view.Accepted || view.Freshness != "current" {
		t.Fatal(view, err)
	}
	// Accepted immutable output is still readable but visibly stale after replacement.
	item := p.Items[0]
	entry, err := b.Catalog.Library(context.Background(), item.Reference.RecordingID)
	if err != nil {
		t.Fatal(err)
	}
	current, err := b.Catalog.Recording(context.Background(), entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	desktopRecording(t, b, entry, current.Revision, doc)
	stale, err := b.inspectGroundedAnswer(context.Background(), result.ID)
	if err != nil || stale.Freshness != "stale" || !stale.Accepted {
		t.Fatal(stale, err)
	}
}
