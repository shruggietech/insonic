// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func nativeRepairFixture(t *testing.T) (*App, string, *subtitles.Driver, *[5]atomic.Int32, *atomic.Bool) {
	t.Helper()
	executable := os.Getenv("CUESON_EXECUTABLE")
	if executable == "" {
		t.Skip("pinned native Cueson not selected")
	}
	binary, e := os.ReadFile(executable)
	if e != nil {
		t.Fatal(e)
	}
	digest := sha256.Sum256(binary)
	driver, e := subtitles.New(subtitles.Tool{Executable: executable, ExecutableSHA256: hex.EncodeToString(digest[:])})
	if e != nil {
		t.Fatal(e)
	}
	w, e := workspace.Init(t.TempDir(), "native bounded repair")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { a.Close() })
	source, _ := filepath.Abs("../../tests/fixtures/media/speech.flac")
	id := seedLegacyMedia(t, a, source, "")
	calls := &[5]atomic.Int32{}
	repair := &atomic.Bool{}
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: driver, PrepareWindows: func(_ context.Context, _ catalog.LibraryEntry, options processing.WindowOptions, identity processing.PlanIdentity) (*windowExecution, error) {
			tracks := []processing.TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 80000, EndSample: 80000, StartNumerator: "0", StartDenominator: "1"}}
			plan, e := processing.BuildWindowPlan(identity, options, tracks)
			if e != nil {
				return nil, e
			}
			return &windowExecution{Plan: plan, Close: func() error { return nil }, Window: func(_ context.Context, index int) (*audioExecution, error) {
				mapping := plan.Windows[index].Mapping
				return &audioExecution{SourceMap: mapping, Provenance: map[string]any{}, Diagnostics: []processing.Diagnostic{}, Close: func() error { return nil }, Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
					calls[index].Add(1)
					segments := []processing.RecognitionSegment{}
					if repair.Load() || index%2 == 0 {
						segments = append(segments, processing.RecognitionSegment{StartUS: 100000, EndUS: 900000, Text: "A repeated phrase."})
					}
					return processing.RecognitionFromSegments(segments, len(segments) == 0, mapping)
				}, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
					return processing.DiarizationFromTurns([]processing.Turn{{StartUS: 100000, EndUS: 900000, Label: "speaker"}}, false, mapping)
				}}, nil
			}}, nil
		}}, nil
	}
	zero := int64(0)
	response := realRequest(a, "recordings.process", id, RecordingOptions{Transcription: "generate", Diarization: "run", RecognitionModelID: contracts.ID(), DiarizationModelID: contracts.ID(), Windows: &processing.WindowOptions{CoreUS: 1000000, ContextUS: &zero}})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	done := awaitWork(t, a, response.Result.(map[string]any)["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatal(done.State, string(done.Result))
	}
	repair.Store(true)
	return a, id, driver, calls, repair
}
func repairRequestFor(r catalog.Recording) RecordingRepairOptions {
	return RecordingRepairOptions{ExpectedRevision: r.Revision, SourceRevision: r.SourceRevision, DocumentDigest: r.DocumentDigest, Automatic: true, Processing: RecordingOptions{RecognitionModelID: contracts.ID(), DiarizationModelID: contracts.ID()}}
}
func TestNativeRecordingRepairDisjointGapsAndReplay(t *testing.T) {
	a, id, driver, calls, _ := nativeRepairFixture(t)
	ctx := context.Background()
	old, e := a.Catalog.Recording(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	rawBefore := configurationBytes(old)
	coverage, e := a.recordingCoverage(contracts.Request{ItemID: id})
	if e != nil {
		t.Fatal(e)
	}
	before := coverage.(processing.CoverageReport)
	if e = schemas.ValidateRecordingCoverage(configurationBytes(before)); e != nil {
		t.Fatal("coverage output schema", e, string(configurationBytes(before)))
	}
	if before.Tracks[0].Metrics.UncoveredObservedSpeechUS != 1600000 {
		t.Fatalf("wrong initial coverage: %+v", before.Tracks)
	}
	request := repairRequestFor(old)
	response := realRequest(a, "recordings.repair", id, request)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	workID := response.Result.(map[string]any)["work_id"].(string)
	done := awaitWork(t, a, workID)
	if done.State != "succeeded" {
		t.Fatal(done.State, string(done.Result))
	}
	if e = schemas.ValidateRecordingRepair(done.Result); e != nil {
		t.Fatal("repair output schema", e)
	}
	var decision struct {
		Status   string         `json:"status"`
		Attempts map[string]int `json:"attempts"`
	}
	if json.Unmarshal(done.Result, &decision) != nil || decision.Status != "accepted" || len(decision.Attempts) != 2 {
		t.Fatal(string(done.Result))
	}
	for i := range calls {
		want := int32(1)
		if i == 1 || i == 3 {
			want = 2
		}
		if calls[i].Load() != want {
			t.Fatalf("window %d invoked %d times, want %d", i, calls[i].Load(), want)
		}
	}
	current, e := a.Catalog.Recording(ctx, id)
	if e != nil || current.Revision <= old.Revision || string(current.SourceMap) != string(old.SourceMap) || current.SourceDigest != old.SourceDigest {
		t.Fatal("wrong accepted authority", e)
	}
	if e = driver.Validate(ctx, current.Document); e != nil {
		t.Fatal(e)
	}
	var links recordingCoverageProvenance
	if json.Unmarshal(current.Provenance, &links) != nil {
		t.Fatal("invalid cue provenance")
	}
	var ordered struct {
		Cues []struct {
			ID string `json:"id"`
		} `json:"cues"`
	}
	json.Unmarshal(current.Document, &ordered)
	for _, link := range links.CueSources {
		if link.Source.Ordinal < 0 || link.Source.Ordinal >= len(ordered.Cues) || ordered.Cues[link.Source.Ordinal].ID != link.CueID {
			t.Fatal("cue provenance ordinal differs from accepted document")
		}
	}
	var oldDoc, newDoc map[string]json.RawMessage
	json.Unmarshal(old.Document, &oldDoc)
	json.Unmarshal(current.Document, &newDoc)
	var oldCues, newCues []map[string]json.RawMessage
	json.Unmarshal(oldDoc["cues"], &oldCues)
	json.Unmarshal(newDoc["cues"], &newCues)
	if len(newCues) != 5 || string(oldDoc["source"]) != string(newDoc["source"]) {
		t.Fatal("source or cue count changed")
	}
	for i, c := range oldCues {
		delete(c, "ordinal")
		actual := newCues[i]
		delete(actual, "ordinal")
		if !reflect.DeepEqual(c, actual) {
			t.Fatal("unaffected cue changed", i)
		}
	}
	after, e := a.recordingCoverage(contracts.Request{ItemID: id})
	if e != nil || after.(processing.CoverageReport).Tracks[0].Metrics.UncoveredObservedSpeechUS != 0 {
		t.Fatal("repair did not fill gaps", e)
	}
	frozen, e := a.Catalog.Work(ctx, workID)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := a.processRecordingRepair(ctx, frozen)
	if e != nil || string(replay.(json.RawMessage)) != string(done.Result) {
		t.Fatal("accepted replay changed", e)
	}
	for i := range calls {
		want := int32(1)
		if i == 1 || i == 3 {
			want = 2
		}
		if calls[i].Load() != want {
			t.Fatal("replay reran inference")
		}
	}
	plans, e := a.Catalog.ProcessingPlans(ctx, workID)
	if e != nil || len(plans) != 1 || plans[0].State != "completed" {
		t.Fatal("accepted plan not retired", plans, e)
	}
	checkpoints, e := a.Catalog.ProcessingWindowCheckpoints(ctx, plans[0].ID)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range checkpoints {
		if c.State != "retired" {
			t.Fatal("checkpoint still current", c.State)
		}
	}
	// The same complete transcript is a valid candidate but no improvement.
	end := int64(16000)
	request = repairRequestFor(current)
	request.Automatic = false
	request.Ranges = []processing.TrackSelection{{StreamIndex: 0, EndSample: &end}}
	response = realRequest(a, "recordings.repair", id, request)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	done = awaitWork(t, a, response.Result.(map[string]any)["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatal(done.State, string(done.Result))
	}
	json.Unmarshal(done.Result, &decision)
	if decision.Status != "unchanged" {
		t.Fatal(string(done.Result))
	}
	unchanged, _ := a.Catalog.Recording(ctx, id)
	if string(configurationBytes(unchanged)) != string(configurationBytes(current)) {
		t.Fatal("non-improving candidate replaced current")
	}
	// A stale source/document request cannot schedule another window.
	response = realRequest(a, "recordings.repair", id, repairRequestFor(old))
	if response.Error == nil || response.Error.Code != "conflict" {
		t.Fatal("stale repair accepted")
	}
	if string(rawBefore) == string(configurationBytes(current)) {
		t.Fatal("fixture did not change")
	}
}

func TestNativeRecordingRepairInterruptedBudgetAndCheckpointReuse(t *testing.T) {
	for _, budget := range []int{1, 2} {
		t.Run(string(rune('0'+budget)), func(t *testing.T) {
			a, id, _, calls, _ := nativeRepairFixture(t)
			ctx := context.Background()
			old, _ := a.Catalog.Recording(ctx, id)
			before := configurationBytes(old)
			factory := a.recordingFactory
			var injected atomic.Bool
			injected.Store(true)
			a.recordingFactory = func() (*recordingExecution, error) {
				ex, e := factory()
				if e != nil {
					return nil, e
				}
				prepare := ex.PrepareWindows
				ex.PrepareWindows = func(ctx context.Context, entry catalog.LibraryEntry, o processing.WindowOptions, i processing.PlanIdentity) (*windowExecution, error) {
					w, e := prepare(ctx, entry, o, i)
					if e != nil {
						return nil, e
					}
					window := w.Window
					w.Window = func(ctx context.Context, index int) (*audioExecution, error) {
						audio, e := window(ctx, index)
						if e != nil {
							return nil, e
						}
						recognize := audio.Recognize
						audio.Recognize = func(ctx context.Context, m string, o processing.RecognitionOptions) (processing.RecognitionResult, error) {
							if index == 3 && injected.Swap(false) {
								return processing.RecognitionResult{}, contracts.Fail("invalid_request")
							}
							return recognize(ctx, m, o)
						}
						return audio, nil
					}
					return w, nil
				}
				return ex, nil
			}
			request := repairRequestFor(old)
			request.Policy = &processing.RepairPolicy{MaxAttempts: budget}
			r := realRequest(a, "recordings.repair", id, request)
			if r.Error != nil {
				t.Fatal(r.Error)
			}
			workID := r.Result.(map[string]any)["work_id"].(string)
			failed := awaitWork(t, a, workID)
			if failed.State != "failed" {
				t.Fatal(failed.State)
			}
			current, _ := a.Catalog.Recording(ctx, id)
			if string(configurationBytes(current)) != string(before) {
				t.Fatal("failed candidate changed current")
			}
			attempts, e := a.Catalog.RepairWindowAttempts(ctx, workID)
			if e != nil || len(attempts) != 2 {
				t.Fatal(attempts, e)
			}
			for _, n := range attempts {
				if n != 1 {
					t.Fatal("attempt not consumed before failure")
				}
			}
			if r = realRequest(a, "work.retry", workID, nil); r.Error != nil {
				t.Fatal(r.Error)
			}
			done := awaitWork(t, a, workID)
			if budget == 1 {
				if done.State != "failed" {
					t.Fatal("exhausted budget ran", done.State)
				}
				attempts, _ = a.Catalog.RepairWindowAttempts(ctx, workID)
				for _, n := range attempts {
					if n != 1 {
						t.Fatal("budget reset on retry")
					}
				}
				current, _ = a.Catalog.Recording(ctx, id)
				if string(configurationBytes(current)) != string(before) {
					t.Fatal("exhaustion changed current")
				}
			} else {
				if done.State != "succeeded" {
					t.Fatal(done.State, string(done.Result))
				}
				attempts, _ = a.Catalog.RepairWindowAttempts(ctx, workID)
				sum := 0
				for _, n := range attempts {
					sum += n
				}
				if sum != 3 {
					t.Fatal("wrong durable attempt accounting", attempts)
				}
			}
			if calls[1].Load() != 2 {
				t.Fatal("verified checkpoint recomputed", calls[1].Load())
			}
			for _, i := range []int{0, 2, 4} {
				if calls[i].Load() != 1 {
					t.Fatal("unselected window ran", i)
				}
			}
		})
	}
}

func TestNativeRecordingRepairCancelledRestartAndCorruptCheckpoint(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		name := "resume"
		if corrupt {
			name = "corrupt"
		}
		t.Run(name, func(t *testing.T) {
			a, id, _, calls, _ := nativeRepairFixture(t)
			ctx := context.Background()
			old, _ := a.Catalog.Recording(ctx, id)
			factory := a.recordingFactory
			entered := make(chan struct{})
			var blocked atomic.Bool
			a.recordingFactory = func() (*recordingExecution, error) {
				ex, e := factory()
				if e != nil {
					return nil, e
				}
				prepare := ex.PrepareWindows
				ex.PrepareWindows = func(ctx context.Context, entry catalog.LibraryEntry, o processing.WindowOptions, i processing.PlanIdentity) (*windowExecution, error) {
					w, e := prepare(ctx, entry, o, i)
					if e != nil {
						return nil, e
					}
					window := w.Window
					w.Window = func(ctx context.Context, index int) (*audioExecution, error) {
						audio, e := window(ctx, index)
						if e != nil {
							return nil, e
						}
						recognize := audio.Recognize
						audio.Recognize = func(ctx context.Context, m string, o processing.RecognitionOptions) (processing.RecognitionResult, error) {
							if index == 3 && !blocked.Swap(true) {
								close(entered)
								<-ctx.Done()
								return processing.RecognitionResult{}, contracts.Fail("cancelled")
							}
							return recognize(ctx, m, o)
						}
						return audio, nil
					}
					return w, nil
				}
				return ex, nil
			}
			request := repairRequestFor(old)
			request.Policy = &processing.RepairPolicy{MaxAttempts: 2}
			r := realRequest(a, "recordings.repair", id, request)
			if r.Error != nil {
				t.Fatal(r.Error)
			}
			workID := r.Result.(map[string]any)["work_id"].(string)
			select {
			case <-entered:
			case <-time.After(qualificationtiming.Duration("readiness_seconds")):
				t.Fatal("repair did not enter cancellation window")
			}
			if r = realRequest(a, "work.cancel", workID, nil); r.Error != nil {
				t.Fatal(r.Error)
			}
			plan, checkpoint := onlyReadyWindow(t, a, workID)
			workspace := a.Workspace
			a.Close()
			restarted, e := New(workspace)
			if e != nil {
				t.Fatal(e)
			}
			defer restarted.Close()
			restarted.recordingFactory = factory
			retained, ready := onlyReadyWindow(t, restarted, workID)
			if !reflect.DeepEqual(retained, plan) || ready != checkpoint {
				t.Fatal("restart changed checkpoint authority")
			}
			if corrupt {
				publication, e := restarted.Catalog.Publication(ctx, checkpoint.PublicationID)
				if e != nil {
					t.Fatal(e)
				}
				directory := workspace.Config.Profiles.Storage.Configuration["root"].(string)
				if !filepath.IsAbs(directory) {
					directory = filepath.Join(workspace.Control, directory)
				}
				path := filepath.Join(directory, filepath.FromSlash(publication.Key))
				raw, e := os.ReadFile(path)
				if e != nil || len(raw) == 0 {
					t.Fatal(e)
				}
				raw[0] ^= 1
				if e = os.WriteFile(path, raw, 0600); e != nil {
					t.Fatal(e)
				}
			}
			if r = realRequest(restarted, "work.retry", workID, nil); r.Error != nil {
				t.Fatal(r.Error)
			}
			done := awaitWork(t, restarted, workID)
			current, e := restarted.Catalog.Recording(ctx, id)
			if e != nil {
				t.Fatal(e)
			}
			if corrupt {
				if done.State != "failed" || !reflect.DeepEqual(current, old) {
					t.Fatal("corrupt checkpoint replaced current", done.State)
				}
				if calls[3].Load() != 1 {
					t.Fatal("corrupt checkpoint silently recomputed")
				}
			} else {
				if done.State != "succeeded" || current.Revision <= old.Revision {
					t.Fatal(done.State, string(done.Result))
				}
			}
			if calls[1].Load() != 2 {
				t.Fatal("completed window recomputed after restart")
			}
		})
	}
}
