// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	"github.com/shruggietech/insonic/internal/workspace"
)

type windowFault func(context.Context, int, int32) (processing.DiarizationResult, error)

func installWindowFaultFixture(t *testing.T, a *App, calls *[3]atomic.Int32, fault windowFault) {
	t.Helper()
	doc, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, PrepareWindows: func(_ context.Context, _ catalog.LibraryEntry, options processing.WindowOptions, identity processing.PlanIdentity) (*windowExecution, error) {
			plan, err := processing.BuildWindowPlan(identity, options, []processing.TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 48000, StartNumerator: "0", StartDenominator: "1", EndSample: 48000}})
			if err != nil {
				return nil, err
			}
			return &windowExecution{Plan: plan, Close: func() error { return nil }, Window: func(_ context.Context, index int) (*audioExecution, error) {
				return &audioExecution{SourceMap: plan.Windows[index].Mapping, Provenance: map[string]any{}, Diagnostics: []processing.Diagnostic{}, Close: func() error { return nil }, Diarize: func(ctx context.Context, _ string, _ processing.DiarizationOptions) (processing.DiarizationResult, error) {
					count := calls[index].Add(1)
					if fault != nil {
						return fault(ctx, index, count)
					}
					return processing.DiarizationResult{NoSpeech: true, Provenance: map[string]any{}}, nil
				}}, nil
			}}, nil
		}}, nil
	}
}

func startWindowFaultWork(t *testing.T, a *App, mediaID string) string {
	t.Helper()
	zero := int64(0)
	response := realRequest(a, "recordings.process", mediaID, RecordingOptions{Transcription: "supplied", Diarization: "run", DiarizationModelID: contracts.ID(), Windows: &processing.WindowOptions{CoreUS: 1_000_000, ContextUS: &zero}})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	return response.Result.(map[string]any)["work_id"].(string)
}

func onlyReadyWindow(t *testing.T, a *App, workID string) (catalog.ProcessingPlan, catalog.ProcessingWindowCheckpoint) {
	t.Helper()
	plans, err := a.Catalog.ProcessingPlans(context.Background(), workID)
	if err != nil || len(plans) != 1 {
		t.Fatalf("frozen plans: %#v %v", plans, err)
	}
	rows, err := a.Catalog.ProcessingWindowCheckpoints(context.Background(), plans[0].ID)
	if err != nil || len(rows) != 1 || rows[0].State != "ready" {
		t.Fatalf("verified window checkpoint: %#v %v", rows, err)
	}
	return plans[0], rows[0]
}

func newWindowFaultApp(t *testing.T) (*workspace.Workspace, *App, string) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "window faults")
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	return w, a, importRecordingFixture(t, a)
}

func TestWindowCancellationRestartPreservesVerifiedProducer(t *testing.T) {
	w, a, mediaID := newWindowFaultApp(t)
	var calls [3]atomic.Int32
	entered := make(chan struct{})
	installWindowFaultFixture(t, a, &calls, func(ctx context.Context, index int, count int32) (processing.DiarizationResult, error) {
		if index == 1 && count == 1 {
			close(entered)
			<-ctx.Done()
			return processing.DiarizationResult{}, contracts.Fail("cancelled")
		}
		return processing.DiarizationResult{NoSpeech: true, Provenance: map[string]any{}}, nil
	})
	workID := startWindowFaultWork(t, a, mediaID)
	select {
	case <-entered:
	case <-time.After(qualificationtiming.Duration("readiness_seconds")):
		t.Fatal("second window did not start")
	}
	if response := realRequest(a, "work.cancel", workID, nil); response.Error != nil {
		t.Fatal(response.Error)
	}
	plan, checkpoint := onlyReadyWindow(t, a, workID)
	if _, err := a.Catalog.Recording(context.Background(), mediaID); err == nil {
		t.Fatal("cancelled candidate became current")
	}
	a.Close()
	restarted, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	installWindowFaultFixture(t, restarted, &calls, nil)
	retainedPlan, retained := onlyReadyWindow(t, restarted, workID)
	if !reflect.DeepEqual(retainedPlan, plan) || retained != checkpoint {
		t.Fatal("restart changed frozen identity or producing attempt")
	}
	if response := realRequest(restarted, "work.retry", workID, nil); response.Error != nil {
		t.Fatal(response.Error)
	}
	done := awaitWork(t, restarted, workID)
	if done.State != "succeeded" || calls[0].Load() != 1 || calls[1].Load() != 2 || calls[2].Load() != 1 {
		t.Fatalf("restart recomputed verified work: %s %s; calls %d %d %d", done.State, done.Result, calls[0].Load(), calls[1].Load(), calls[2].Load())
	}
	rows, err := restarted.Catalog.ProcessingWindowCheckpoints(context.Background(), plan.ID)
	if err != nil || len(rows) != 3 {
		t.Fatal(rows, err)
	}
	found := false
	for _, row := range rows {
		if row.ID == checkpoint.ID {
			row.State = checkpoint.State
			found = row == checkpoint
		}
	}
	if !found {
		t.Fatal("retry relabeled the retained producer")
	}
}

func TestWindowCorruptCheckpointCannotReplaceCurrent(t *testing.T) {
	_, a, mediaID := newWindowFaultApp(t)
	var initialCalls [3]atomic.Int32
	installWindowFaultFixture(t, a, &initialCalls, nil)
	initialID := startWindowFaultWork(t, a, mediaID)
	if done := awaitWork(t, a, initialID); done.State != "succeeded" {
		t.Fatal(done.State, string(done.Result))
	}
	current, err := a.Catalog.Recording(context.Background(), mediaID)
	if err != nil {
		t.Fatal(err)
	}
	var calls [3]atomic.Int32
	installWindowFaultFixture(t, a, &calls, func(_ context.Context, index int, count int32) (processing.DiarizationResult, error) {
		if index == 1 && count == 1 {
			return processing.DiarizationResult{}, contracts.Fail("operation_failed")
		}
		return processing.DiarizationResult{NoSpeech: true, Provenance: map[string]any{}}, nil
	})
	workID := startWindowFaultWork(t, a, mediaID)
	if done := awaitWork(t, a, workID); done.State != "failed" {
		t.Fatal(done.State, string(done.Result))
	}
	_, checkpoint := onlyReadyWindow(t, a, workID)
	publication, err := a.Catalog.Publication(context.Background(), checkpoint.PublicationID)
	if err != nil {
		t.Fatal(err)
	}
	objectRoot := a.Workspace.Config.Profiles.Storage.Configuration["root"].(string)
	if !filepath.IsAbs(objectRoot) {
		objectRoot = filepath.Join(a.Workspace.Control, objectRoot)
	}
	objectPath := filepath.Join(objectRoot, filepath.FromSlash(publication.Key))
	raw, err := os.ReadFile(objectPath)
	if err != nil || len(raw) == 0 {
		t.Fatal(err)
	}
	raw[0] ^= 1
	if err = os.WriteFile(objectPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if response := realRequest(a, "work.retry", workID, nil); response.Error != nil {
		t.Fatal(response.Error)
	}
	done := awaitWork(t, a, workID)
	if done.State != "failed" || calls[0].Load() != 1 || calls[1].Load() != 1 || calls[2].Load() != 0 {
		t.Fatalf("corrupt checkpoint trusted or silently recomputed: %s %s", done.State, done.Result)
	}
	after, err := a.Catalog.Recording(context.Background(), mediaID)
	if err != nil || !reflect.DeepEqual(after, current) {
		t.Fatal("corrupt checkpoint replaced current document", err)
	}
}

func TestWindowResumeRetainsObservedDeviceRecovery(t *testing.T) {
	_, a, mediaID := newWindowFaultApp(t)
	var calls [3]atomic.Int32
	installWindowFaultFixture(t, a, &calls, func(_ context.Context, index int, count int32) (processing.DiarizationResult, error) {
		if index == 1 && count == 1 {
			return processing.DiarizationResult{}, contracts.Fail("operation_failed")
		}
		provenance := map[string]any{}
		if index == 0 {
			provenance = map[string]any{"requested_device": "auto", "device": "cpu", "attempted_devices": []string{"cuda", "cuda", "cpu"}, "fallback_reason": "cuda_resource_exhausted", "resource_adjustments": []contracts.ResourceAdjustment{{Attempt: 2, PreviousBatchSize: 4, BatchSize: 1, Reason: "cuda_resource_exhausted"}}}
		}
		return processing.DiarizationResult{NoSpeech: true, Provenance: provenance}, nil
	})
	workID := startWindowFaultWork(t, a, mediaID)
	if done := awaitWork(t, a, workID); done.State != "failed" {
		t.Fatal(done.State, string(done.Result))
	}
	plan, checkpoint := onlyReadyWindow(t, a, workID)
	output, err := a.loadWindowCheckpoint(context.Background(), checkpoint, plan.PlanDigest, checkpoint.WindowID)
	if err != nil || !strings.Contains(strings.Join(processing.DeviceProvenanceNotices(output.Diarization.Provenance), " "), "completed on CPU") {
		t.Fatal("observed fallback missing from verified checkpoint", err)
	}
	if response := realRequest(a, "work.retry", workID, nil); response.Error != nil {
		t.Fatal(response.Error)
	}
	done := awaitWork(t, a, workID)
	var result struct {
		Notices []string `json:"notices"`
	}
	if done.State != "succeeded" || json.Unmarshal(done.Result, &result) != nil || !strings.Contains(strings.Join(result.Notices, " "), "completed on CPU") || calls[0].Load() != 1 {
		t.Fatalf("resume lost fallback notice or reran successful window: %s %s", done.State, done.Result)
	}
	current, err := a.Catalog.Recording(context.Background(), mediaID)
	if err != nil || !strings.Contains(string(current.Provenance), "cuda_resource_exhausted") || !strings.Contains(string(current.Provenance), "resource_adjustments") {
		t.Fatal("accepted provenance lost recovery history", string(current.Provenance), err)
	}
}
