// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestWindowElectionRejectsUnmatchedExplicitAudioSelection(t *testing.T) {
	zero, video, missing, selected, channel := 0, 1, 9, 2, 1
	facts, _ := json.Marshal(library.Facts{Streams: []library.Stream{{Index: zero, Kind: "audio"}, {Index: video, Kind: "video"}, {Index: selected, Kind: "audio"}}})
	entry := catalog.LibraryEntry{Facts: facts}
	for _, index := range []*int{&video, &missing} {
		_, err := electWindowOptions(RecordingOptions{Windows: &processing.WindowOptions{}, Audio: processing.AudioOptions{StreamIndex: index, Channel: &channel}}, entry)
		if err == nil || contracts.SafeError(err).Code != "unsupported_audio" {
			t.Fatalf("unmatched stream defaulted: %v", err)
		}
	}
	got, err := electWindowOptions(RecordingOptions{Windows: &processing.WindowOptions{}, Audio: processing.AudioOptions{StreamIndex: &selected, Channel: &channel}}, entry)
	if err != nil || len(got.Windows.Tracks) != 1 || got.Windows.Tracks[0].StreamIndex != selected || got.Windows.Tracks[0].Channel == nil || *got.Windows.Tracks[0].Channel != channel {
		t.Fatalf("explicit source lost: %+v %v", got, err)
	}
	if _, err = electWindowOptions(RecordingOptions{Windows: &processing.WindowOptions{}, Audio: processing.AudioOptions{Channel: &channel}}, catalog.LibraryEntry{Facts: json.RawMessage(`{"streams":[]}`)}); err == nil {
		t.Fatal("channel election defaulted without an audio stream")
	}
}

func TestWindowElectionPreservesShortRequestsAndZeroContext(t *testing.T) {
	short, long := int64(10_000_000), int64(60_000_000)
	for _, tc := range []struct {
		duration *int64
		want     bool
	}{{&short, false}, {&long, true}, {nil, false}} {
		got, err := electWindowOptions(RecordingOptions{}, catalog.LibraryEntry{DurationUS: tc.duration})
		if err != nil || (got.Windows != nil) != tc.want {
			t.Fatalf("election: %#v %v", got, err)
		}
	}
	zero := int64(0)
	got, err := electWindowOptions(RecordingOptions{Windows: &processing.WindowOptions{CoreUS: 1_000_000, ContextUS: &zero}}, catalog.LibraryEntry{})
	if err != nil || got.Windows == nil || got.Windows.ContextUS == nil || *got.Windows.ContextUS != 0 {
		t.Fatalf("explicit zero lost: %#v %v", got, err)
	}
	if _, err = electWindowOptions(RecordingOptions{Windows: &processing.WindowOptions{CoreUS: -1}}, catalog.LibraryEntry{}); err == nil {
		t.Fatal("invalid limits accepted")
	}
}

func TestWindowIdentityExcludesReboundPathsButIncludesPinnedBytes(t *testing.T) {
	p := recordingPayload{MediaID: contracts.ID(), SourceRevision: 1, Options: RecordingOptions{Transcription: "generate"}, Tools: &ProcessingTools{}}
	p.Tools.Processing.Worker.Path = "A:/source/worker.py"
	p.Tools.Processing.Worker.SHA256 = strings.Repeat("a", 64)
	p.Tools.Cueson.Executable = "A:/source/cueson.exe"
	p.Tools.Cueson.ExecutableSHA256 = strings.Repeat("c", 64)
	first := windowConfigurationDigest(p)
	p.Tools.Processing.Worker.Path = "B:/restored/worker.py"
	p.Tools.Cueson.Executable = "B:/restored/cueson.exe"
	if got := windowConfigurationDigest(p); got != first {
		t.Fatal("portable path changed identity")
	}
	p.Tools.Cueson.ExecutableSHA256 = strings.Repeat("d", 64)
	if windowConfigurationDigest(p) == first {
		t.Fatal("changed Cueson bytes reused identity")
	}
	p.Tools.Cueson.ExecutableSHA256 = strings.Repeat("c", 64)
	p.Tools.Processing.Worker.SHA256 = strings.Repeat("b", 64)
	if windowConfigurationDigest(p) == first {
		t.Fatal("changed worker bytes reused identity")
	}
	p.Tools.Processing.Worker.SHA256 = strings.Repeat("a", 64)
	p.Options.Recognition.Language = "fr"
	if windowConfigurationDigest(p) == first {
		t.Fatal("semantic options omitted")
	}
}

func TestWindowProcessingRetryRetainsVerifiedWindows(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "window resume")
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	doc, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	id := importRecordingFixture(t, a)
	var calls [3]atomic.Int32
	var fail atomic.Bool
	fail.Store(true)
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, PrepareWindows: func(_ context.Context, entry catalog.LibraryEntry, options processing.WindowOptions, identity processing.PlanIdentity) (*windowExecution, error) {
			plan, e := processing.BuildWindowPlan(identity, options, []processing.TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 48000, StartNumerator: "0", StartDenominator: "1", EndSample: 48000}})
			if e != nil {
				return nil, e
			}
			return &windowExecution{Plan: plan, Close: func() error { return nil }, Window: func(_ context.Context, index int) (*audioExecution, error) {
				return &audioExecution{SourceMap: plan.Windows[index].Mapping, Provenance: map[string]any{}, Diagnostics: []processing.Diagnostic{}, Close: func() error { return nil }, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
					calls[index].Add(1)
					if index == 1 && fail.Swap(false) {
						return processing.DiarizationResult{}, contracts.Fail("operation_failed")
					}
					return processing.DiarizationResult{NoSpeech: true, Provenance: map[string]any{}}, nil
				}}, nil
			}}, nil
		}}, nil
	}
	zero := int64(0)
	r := realRequest(a, "recordings.process", id, RecordingOptions{Transcription: "supplied", Diarization: "run", DiarizationModelID: contracts.ID(), Windows: &processing.WindowOptions{CoreUS: 1_000_000, ContextUS: &zero}})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	workID := r.Result.(map[string]any)["work_id"].(string)
	first := awaitWork(t, a, workID)
	if first.State != "failed" {
		t.Fatalf("expected injected failure: %s %s", first.State, first.Result)
	}
	var failedProgress struct {
		Completed int    `json:"completed_windows"`
		Total     int    `json:"total_windows"`
		Active    string `json:"active_window_id"`
	}
	if json.Unmarshal(first.Result, &failedProgress) != nil || failedProgress.Completed != 1 || failedProgress.Total != 3 || !contracts.ValidID(failedProgress.Active) {
		t.Fatalf("failed window progress lost: %s", first.Result)
	}
	plans, e := a.Catalog.ProcessingPlans(context.Background(), workID)
	if e != nil || len(plans) != 1 {
		t.Fatalf("plans: %#v %v", plans, e)
	}
	checkpoints, e := a.Catalog.ProcessingWindowCheckpoints(context.Background(), plans[0].ID)
	if e != nil || len(checkpoints) != 1 || checkpoints[0].State != "ready" {
		t.Fatalf("checkpoint: %#v %v", checkpoints, e)
	}
	if r = realRequest(a, "work.retry", workID, nil); r.Error != nil {
		t.Fatal(r.Error)
	}
	done := awaitWork(t, a, workID)
	if done.State != "succeeded" {
		t.Fatalf("retry: %s %s", done.State, done.Result)
	}
	if calls[0].Load() != 1 || calls[1].Load() != 2 || calls[2].Load() != 1 {
		t.Fatalf("recomputed verified windows: %d %d %d", calls[0].Load(), calls[1].Load(), calls[2].Load())
	}
	current, e := a.Catalog.Recording(context.Background(), id)
	if e != nil || current.State != "ready" || !json.Valid(current.Provenance) || current.MappedAudioPublicationID != nil {
		t.Fatalf("current: %#v %v", current, e)
	}
	checkpoints, e = a.Catalog.ProcessingWindowCheckpoints(context.Background(), plans[0].ID)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range checkpoints {
		if c.State != "retired" {
			t.Fatal("accepted transcript retained checkpoint bytes", c)
		}
	}
}
