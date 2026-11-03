// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	"github.com/shruggietech/insonic/internal/voicemodels"
	"github.com/shruggietech/insonic/internal/workspace"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSpeakerWorkflowAppliesAndReplays(t *testing.T) {
	a, r, c := discoveryApp(t)
	ctx := context.Background()
	_, e := a.Catalog.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Alex"}})
	if e != nil {
		t.Fatal(e)
	}
	var calls atomic.Int64
	a.DiscoveryFixture = func(_ context.Context, in assistance.Request, _ assistance.Config) (assistance.Generated, error) {
		calls.Add(1)
		return discoveryResponse(in)
	}
	input, _ := json.Marshal(SpeakerWorkflowInput{Configuration: &pipeline.SpeakerWorkflow{Provider: c, Apply: true}})
	req := contracts.Request{RequestID: contracts.ID(), ItemID: r.ID, Data: input}
	w, e := a.requestSpeakerWorkflow(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	done := awaitWork(t, a, w.ID)
	if done.State != "succeeded" {
		t.Fatal(done.Error, string(done.Result))
	}
	ms, e := a.Catalog.SpeakerMappings(ctx, r.ID)
	if e != nil || len(ms) != 1 || ms[0].Origin != "contextual" {
		t.Fatal(ms, e)
	}
	if _, e = a.requestSpeakerWorkflow(ctx, req); e != nil || calls.Load() != 1 {
		t.Fatal(e, calls.Load())
	}
}

func TestSpeakerWorkflowSavedEnrollmentRetryAndProfileFence(t *testing.T) {
	for _, correction := range []bool{false, true} {
		t.Run(map[bool]string{false: "retry", true: "owner-profile-correction"}[correction], func(t *testing.T) {
			a := configuredApp(t)
			base := configureSpeakerEnrollmentFixture(t, a)
			r, _ := speakerRuntimeRecording(t, a)
			person, e := a.Catalog.PutSpeaker(a.ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Research"}})
			if e != nil {
				t.Fatal(e)
			}
			c := assistance.DefaultConfig()
			c.Enabled = true
			c.Endpoint = "http://127.0.0.1:1"
			c.Model = "fixture"
			var calls atomic.Int64
			a.DiscoveryFixture = func(_ context.Context, in assistance.Request, _ assistance.Config) (assistance.Generated, error) {
				calls.Add(1)
				v, e := discoveryResponse(in)
				v.Raw = []byte(strings.ReplaceAll(string(v.Raw), "Alex", "Research"))
				return v, e
			}
			original := a.speakerFactory
			var fail atomic.Bool
			fail.Store(true)
			a.speakerFactory = func(s *voicemodels.Service) *voicemodels.Service {
				s = original(s)
				embed := s.Embed
				s.Embed = func(ctx context.Context, path string, m processing.SourceMap, model, digest string) ([]float64, error) {
					if fail.Load() {
						return nil, contracts.Fail("unavailable")
					}
					return embed(ctx, path, m, model, digest)
				}
				return s
			}
			adapter := voicemodels.Adapter{ID: "pyannote-profile", ContractVersion: "1", Mode: "local", Architecture: "pyannote", OutputKinds: []string{"voice-embedding"}, Consumers: []string{"voice-matching"}}
			tr := voicemodels.TrainingConfiguration{Adapter: adapter, BaseModelID: base, Kind: "voice-embedding", Device: "cpu"}
			training, _ := json.Marshal(tr)
			def := pipeline.Definition{SpeakerWorkflow: &pipeline.SpeakerWorkflow{Provider: c, Apply: true, Enroll: true, SelectProfile: true}, SpeakerTraining: training}
			raw, _ := json.Marshal(def)
			saved, e := a.Catalog.PutPipeline(a.ctx, contracts.ID(), 0, catalog.Pipeline{ID: contracts.ID(), Name: "Speaker workflow", Preset: "local", Configuration: raw})
			if e != nil {
				t.Fatal(e)
			}
			response := realRequest(a, "speakers.run", r.ID, SpeakerWorkflowInput{PipelineID: saved.ID, PipelineRevision: saved.Revision})
			view := configuredResult(t, response)
			id := view["work_id"].(string)
			failed := awaitWork(t, a, id)
			if failed.State != "failed" {
				t.Fatal(failed.State, string(failed.Result))
			}
			var progress speakerWorkflowResult
			json.Unmarshal(failed.Result, &progress)
			if progress.Stage != "enrollment" || progress.Application == nil || progress.Application.Applied != 1 {
				child, _ := a.Catalog.Work(a.ctx, progress.Awaiting)
				t.Fatal(string(failed.Result), "child", string(child.Result))
			}
			ms, _ := a.Catalog.SpeakerMappings(a.ctx, r.ID)
			if len(ms) != 1 {
				t.Fatal(ms)
			}
			mappingRevision := ms[0].Revision
			if correction {
				if _, e = a.Catalog.SetSpeakerProfile(a.ctx, contracts.ID(), person.Speaker.ID, 0, ""); e != nil {
					t.Fatal(e)
				}
			}
			fail.Store(false)
			if _, e = a.Catalog.RetryWork(a.ctx, contracts.ID(), id); e != nil {
				t.Fatal(e)
			}
			a.recoverWork()
			done := awaitWork(t, a, id)
			want := "succeeded"
			if correction {
				want = "failed"
			}
			if done.State != want {
				t.Fatal(done.State, string(done.Result))
			}
			if calls.Load() != 1 {
				t.Fatal("discovery repeated", calls.Load())
			}
			ms, _ = a.Catalog.SpeakerMappings(a.ctx, r.ID)
			if len(ms) != 1 || ms[0].Revision != mappingRevision {
				t.Fatal("mapping reaccepted", ms)
			}
			outputs, e := a.Catalog.SpeakerOutputs(a.ctx, person.Speaker.ID)
			if e != nil || len(outputs) != 1 {
				t.Fatal(outputs, e)
			}
			profile, e := a.Catalog.SpeakerProfile(a.ctx, person.Speaker.ID)
			if e != nil {
				t.Fatal(e)
			}
			if correction && profile.State != "cleared" || !correction && profile.VersionID != outputs[0].ID {
				t.Fatal(profile)
			}
			if _, e = a.Catalog.Export(a.ctx); e != nil {
				t.Fatal("portable workflow", e)
			}
		})
	}
}
func TestSpeakerWorkflowFourParentsYieldWithoutStarvation(t *testing.T) {
	a, r, c := discoveryApp(t)
	a.DiscoveryFixture = func(_ context.Context, in assistance.Request, _ assistance.Config) (assistance.Generated, error) {
		return discoveryResponse(in)
	}
	ids := []string{}
	for i := 0; i < 4; i++ {
		rec := r
		if i > 0 {
			rec = desktopRecording(t, a, desktopMedia(t, a, false), 0, r.Document)
		}
		raw, _ := json.Marshal(SpeakerWorkflowInput{Configuration: &pipeline.SpeakerWorkflow{Provider: c}})
		w, e := a.requestSpeakerWorkflow(a.ctx, contracts.Request{RequestID: contracts.ID(), ItemID: rec.ID, Data: raw})
		if e != nil {
			t.Fatal(e)
		}
		ids = append(ids, w.ID)
	}
	for _, id := range ids {
		done := awaitWork(t, a, id)
		if done.State != "succeeded" {
			t.Fatal(done.State, string(done.Result))
		}
	}
}
func TestSpeakerWorkflowCancellationAndRestart(t *testing.T) {
	a, r, c := discoveryApp(t)
	started := make(chan struct{}, 1)
	a.DiscoveryFixture = func(ctx context.Context, _ assistance.Request, _ assistance.Config) (assistance.Generated, error) {
		started <- struct{}{}
		<-ctx.Done()
		return assistance.Generated{}, ctx.Err()
	}
	raw, _ := json.Marshal(SpeakerWorkflowInput{Configuration: &pipeline.SpeakerWorkflow{Provider: c, Apply: true}})
	w, e := a.requestSpeakerWorkflow(a.ctx, contracts.Request{RequestID: contracts.ID(), ItemID: r.ID, Data: raw})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-started:
	case <-time.After(qualificationtiming.Duration("readiness_seconds")):
		t.Fatal("child not started")
	}
	if _, e = a.Catalog.CancelWork(a.ctx, contracts.ID(), w.ID); e != nil {
		observed, _ := a.Catalog.Work(a.ctx, w.ID)
		t.Fatal(e, observed.State, string(observed.Result))
	}
	a.recoverWork()
	a.Close()
	reopened, e := workspace.Open(a.Workspace.Root)
	if e != nil {
		t.Fatal(e)
	}
	next, e := New(reopened)
	if e != nil {
		t.Fatal(e)
	}
	defer next.Close()
	next.DiscoveryFixture = func(_ context.Context, in assistance.Request, _ assistance.Config) (assistance.Generated, error) {
		return discoveryResponse(in)
	}
	if _, e = next.Catalog.RetryWork(next.ctx, contracts.ID(), w.ID); e != nil {
		t.Fatal(e)
	}
	next.recoverWork()
	done := awaitWork(t, next, w.ID)
	if done.State != "succeeded" {
		t.Fatal(done.State, string(done.Result))
	}
}
