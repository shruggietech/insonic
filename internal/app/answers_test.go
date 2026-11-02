// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

// Preparing an answer must never regress to the existing whole-library query
// hydration path. Other catalog methods remain the real backend implementation.
type answerNoExport struct{ catalog.Catalog }

func (answerNoExport) Export(context.Context) (catalog.Snapshot, error) {
	panic("unbounded answer catalog export")
}

func TestAnswerPreparationCacheAndCurrentAuthority(t *testing.T) {
	a := configuredApp(t)
	doc, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	entry := desktopMedia(t, a, false)
	recording := desktopRecording(t, a, entry, 0, doc)
	second := desktopMedia(t, a, false)
	desktopRecording(t, a, second, 0, doc)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	a.Catalog = answerNoExport{a.Catalog}
	r := answers.Request{Version: contracts.Version, Question: "What did the recordings say?"}
	first, err := a.prepareAnswer(context.Background(), r)
	if err != nil || first.Package.ID == "" || len(first.Package.Items) < 2 || first.CacheHit || first.Freshness != "current" {
		t.Fatal(first, err)
	}
	next, err := a.prepareAnswer(context.Background(), r)
	if err != nil || !next.CacheHit || next.Package.ID != first.Package.ID {
		t.Fatal("compatible cache miss", next, err)
	}
	if first.Package.Observation.GraphState != "unavailable" || first.Package.State != "partial" {
		t.Fatal("graph absence hidden", first.Package)
	}
	shown, err := a.inspectAnswer(context.Background(), first.Package.ID)
	if err != nil || shown.Freshness != "current" {
		t.Fatal(shown, err)
	}
	if shown.Usage.SourceBytes == 0 || shown.Usage.Rows == 0 || shown.Usage.Rounds < 2 {
		t.Fatal("inspection omitted actual work", shown.Usage)
	}
	item := first.Package.Items[0]
	e := answers.Envelope{Version: contracts.Version, PackageID: first.Package.ID, State: "partial", Claims: []answers.Claim{{ID: "claim", Text: item.Text, Support: "supported", Citations: []answers.Citation{{EvidenceID: item.ID, Reference: item.Reference, Quote: item.Text}}}}}
	if err = a.validateAnswer(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	before, err := a.Catalog.Recording(context.Background(), entry.ID)
	if err != nil || before.Revision != recording.Revision || before.DocumentDigest != recording.DocumentDigest {
		t.Fatal("preparation mutated source", err)
	}
	desktopRecording(t, a, entry, recording.Revision, doc)
	stale, err := a.inspectAnswer(context.Background(), first.Package.ID)
	if err != nil || stale.Freshness != "stale" || stale.Package.ID != first.Package.ID {
		t.Fatal("historical package lost", stale, err)
	}
	if err = a.validateAnswer(context.Background(), e); err == nil {
		t.Fatal("stale answer citations accepted")
	}
	changed, err := a.prepareAnswer(context.Background(), r)
	if err != nil || changed.CacheHit || changed.Package.ID == first.Package.ID {
		t.Fatal("source replacement reused cache", changed, err)
	}
	r.Provider.Model = "other-model"
	model, err := a.prepareAnswer(context.Background(), r)
	if err != nil || model.CacheHit || model.Package.Key == changed.Package.Key {
		t.Fatal("model identity reused cache", model, err)
	}
	// A malformed pointer is a diagnosed miss, not a successful package read.
	name := "answer-context:" + model.Package.Key
	_, revision, err := a.Catalog.NamedSetting(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Catalog.PutNamedSetting(context.Background(), contracts.ID(), name, revision, json.RawMessage(`{"invalid":true}`))
	if err != nil {
		t.Fatal(err)
	}
	repaired, err := a.prepareAnswer(context.Background(), r)
	if err != nil || repaired.CacheHit || repaired.Package.ID == model.Package.ID {
		t.Fatal("corrupt index reused", repaired, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	stopped, err := a.prepareAnswer(cancelled, r)
	if err == nil || contracts.SafeError(err).Code != "cancelled" || stopped.Package.State != "cancelled" || stopped.Package.ID != "" {
		t.Fatal("cancelled preparation represented as complete", stopped, err)
	}
}

type cancelAnswerEpoch struct {
	catalog.Catalog
	calls  int
	cancel context.CancelFunc
}

func (s *cancelAnswerEpoch) AnswerEpoch(ctx context.Context, ids []string, rows, bytes int) (catalog.AnswerEpoch, error) {
	s.calls++
	if s.calls == 3 {
		s.cancel()
	}
	return s.Catalog.AnswerEpoch(ctx, ids, rows, bytes)
}
func TestAnswerCancellationAfterPublicationReturnsFailureAndHistoricalPackage(t *testing.T) {
	a := configuredApp(t)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	desktopRecording(t, a, desktopMedia(t, a, false), 0, doc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.Catalog = &cancelAnswerEpoch{Catalog: a.Catalog, cancel: cancel}
	out, err := a.prepareAnswer(ctx, answers.Request{Version: contracts.Version, Question: "What was said?"})
	if err == nil || contracts.SafeError(err).Code != "cancelled" || out.Package.ID == "" || out.Freshness != "unknown" || !slices.Contains(out.Diagnostics, "cancelled") {
		t.Fatal(out, err)
	}
	if _, err := a.inspectAnswer(context.Background(), out.Package.ID); err != nil {
		t.Fatal("retained package lost after cancellation", err)
	}
}

func TestAnswerAggregateBudgetAndUnmatchedContinuation(t *testing.T) {
	a := configuredApp(t)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	for range 3 {
		entry := desktopMedia(t, a, false)
		desktopRecording(t, a, entry, 0, doc)
	}
	r := answers.Request{Version: contracts.Version, Question: "missing", Filters: answers.Filters{Text: "no-cue-contains-this"}, Budget: answers.Budget{Rows: 32, Rounds: 12}}
	out, err := a.prepareAnswer(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Package.Items) != 0 || out.Package.Usage.Rows > 32 || out.Package.Usage.Rounds > 12 || out.Package.State == "complete" {
		t.Fatal("budget/empty state", out)
	}
	r.Budget.Rows = 1
	out, err = a.prepareAnswer(context.Background(), r)
	if err != nil || out.Package.State != "partial" || out.Package.Usage.Rows > 1 {
		t.Fatal("thin identity scan escaped budget", out, err)
	}
}

func TestAnswerImportedPackageCannotAuthorizeInventedQuote(t *testing.T) {
	a := configuredApp(t)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	entry := desktopMedia(t, a, false)
	desktopRecording(t, a, entry, 0, doc)
	r := answers.Request{Version: contracts.Version, Question: "What was said?"}
	view, err := a.prepareAnswer(context.Background(), r)
	if err != nil || len(view.Package.Items) == 0 {
		t.Fatal(view, err)
	}
	fake := view.Package
	fake.ID = contracts.ID()
	original := fake.Items[0].Text
	fake.Items[0].Text = original + " invented sentence"
	fake.Context, err = answers.BuildContext(fake.Request, fake.Items)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.saveAnswerPackage(context.Background(), fake, 0); err != nil {
		t.Fatal(err)
	}
	item := fake.Items[0]
	e := answers.Envelope{Version: contracts.Version, PackageID: fake.ID, State: "partial", Claims: []answers.Claim{{ID: "first", Text: original, Support: "supported", Citations: []answers.Citation{{EvidenceID: item.ID, Reference: item.Reference, Quote: original}}}, {ID: "second", Text: "invented sentence", Support: "supported", Citations: []answers.Citation{{EvidenceID: item.ID, Reference: item.Reference, Quote: "invented sentence"}}}}}
	if err = a.validateAnswer(context.Background(), e); err == nil {
		t.Fatal("second quote bypassed current-source validation")
	}
}

func TestAnswerContinuationVisitsEveryRecordingWithoutDuplicateEvidence(t *testing.T) {
	a := configuredApp(t)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	for range 4 {
		entry := desktopMedia(t, a, false)
		desktopRecording(t, a, entry, 0, doc)
	}
	r := answers.Request{Version: contracts.Version, Question: "Research", Budget: answers.Budget{Items: 1}}
	seen := map[string]bool{}
	for page := 0; page < 10; page++ {
		out, err := a.prepareAnswer(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range out.Package.Items {
			if seen[item.ID] {
				t.Fatal("duplicate continuation", item.ID)
			}
			seen[item.ID] = true
		}
		if out.Usage.Rows > out.Package.Request.Budget.Rows || out.Usage.SourceBytes > out.Package.Request.Budget.SourceBytes || out.Usage.Rounds > out.Package.Request.Budget.Rounds {
			t.Fatal("continuation budget", out.Usage)
		}
		if out.Package.Next == "" {
			break
		}
		r.Cursor = out.Package.Next
	}
	if len(seen) != 4 {
		t.Fatal("continuation lost recordings", len(seen))
	}
	r.Question = "changed scope"
	if _, err := a.prepareAnswer(context.Background(), r); err == nil {
		t.Fatal("cursor reused across changed request")
	}
}
