// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"github.com/shruggietech/insonic/schemas"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func discoveryApp(t *testing.T) (*App, catalog.Recording, assistance.Config) {
	return discoveryAppCues(t, 1)
}
func discoveryAppCues(t *testing.T, count int) (*App, catalog.Recording, assistance.Config) {
	t.Helper()
	a := configuredApp(t)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var doc map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&doc); err != nil {
		t.Fatal(err)
	}
	cue := doc["cues"].([]any)[0].(map[string]any)
	payload := cue["payload"].(map[string]any)
	payload["plain_text"] = "I am Alex."
	payload["raw_text"] = "I am Alex."
	payload["lines"] = []string{"I am Alex."}
	cue["speaker_attributions"] = []any{map[string]any{"speaker_id": "voice-1", "start_milliseconds": 0, "end_milliseconds": 1000}}
	cues := []any{}
	for i := 0; i < count; i++ {
		b, _ := json.Marshal(cue)
		var copy map[string]any
		json.Unmarshal(b, &copy)
		copy["id"] = fmt.Sprintf("cue-%06d", i)
		copy["ordinal"] = i
		copy["source_order"] = i
		cues = append(cues, copy)
	}
	doc["cues"] = cues
	doc["document"].(map[string]any)["cue_count"] = count
	doc["stats"].(map[string]any)["cue_count"] = count
	raw, _ = json.Marshal(doc)
	r := desktopRecording(t, a, desktopMedia(t, a, false), 0, raw)
	c := assistance.DefaultConfig()
	c.Enabled = true
	c.Endpoint = "http://127.0.0.1:1"
	c.Model = "fixture"
	return a, r, c
}
func discoveryResponse(in assistance.Request) (assistance.Generated, error) {
	var c sd.Context
	if e := json.Unmarshal(in.Context, &c); e != nil {
		return assistance.Generated{}, e
	}
	observations := []any{}
	for _, cue := range c.Cues {
		if len(cue.Voices) > 0 {
			observations = append(observations, map[string]any{"class": "self-identification", "name": "Alex", "voice_id": cue.Voices[0], "references": []any{map[string]any{"cue_id": cue.ID, "quote": cue.Text}}, "uncertain": false})
		}
	}
	b, _ := json.Marshal(map[string]any{"version": 1, "observations": observations})
	return assistance.Generated{Raw: b, ObservedRoute: "local"}, nil
}
func TestSpeakerDiscoveryRuntimeJourney(t *testing.T) {
	a, r, c := discoveryApp(t)
	var calls atomic.Int64
	a.DiscoveryFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
		calls.Add(1)
		if strings.Contains(string(in.Context), "acoustic") {
			t.Error("prediction sent as extractor evidence")
		}
		return discoveryResponse(in)
	}
	input, _ := json.Marshal(DiscoveryInput{Configuration: &c})
	req := contracts.Request{RequestID: contracts.ID(), ItemID: r.ID, Data: input}
	w, e := a.requestSpeakerDiscovery(context.Background(), req)
	if e != nil {
		t.Fatal(e)
	}
	done := awaitWork(t, a, w.ID)
	if done.State != "succeeded" {
		t.Fatal(done.Error, string(done.Result))
	}
	current, e := a.Catalog.SpeakerDiscovery(context.Background(), r.ID)
	if e != nil {
		t.Fatal(e)
	}
	packageBytes, _ := json.Marshal(map[string]any{"schema_version": contracts.Version, "kind": "speaker-discovery", "workspace_id": a.Workspace.Config.WorkspaceID, "discovery": current})
	if err := schemas.ValidateSpeakerDiscovery(packageBytes); err != nil {
		t.Fatal(err)
	}
	var aggregate sd.Aggregate
	json.Unmarshal(current.Aggregate, &aggregate)
	if aggregate.Decisions[0].Outcome != "supported-contextual" || aggregate.Decisions[0].Name != "Alex" {
		t.Fatal(aggregate)
	}
	again, e := a.requestSpeakerDiscovery(context.Background(), req)
	if e != nil || again.ID != w.ID || calls.Load() != 1 {
		t.Fatal("request repeated provider", e, calls.Load())
	}
	inspect := realRequest(a, "speakers.hypotheses", r.ID, nil)
	if inspect.Error != nil {
		t.Fatal(inspect.Error)
	}
	response := realRequest(a, "speakers.resolve", r.ID, DiscoveryResolutionInput{ExpectedRevision: current.Revision})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	mappings, e := a.Catalog.SpeakerMappings(context.Background(), r.ID)
	if e != nil || len(mappings) != 0 {
		t.Fatal("discovery assigned speaker", mappings, e)
	}
}
func TestSpeakerDiscoveryRejectsProviderProtocolAndRouting(t *testing.T) {
	for _, mode := range []string{"foreign", "duplicate", "hosted"} {
		t.Run(mode, func(t *testing.T) {
			a, r, c := discoveryApp(t)
			a.DiscoveryFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
				v, e := discoveryResponse(in)
				switch mode {
				case "foreign":
					v.Raw = []byte(strings.ReplaceAll(string(v.Raw), "cue-000000", "foreign"))
				case "duplicate":
					v.Raw = []byte(`{"version":1,"version":1,"observations":[]}`)
				case "hosted":
					v.ObservedRoute = "hosted"
				}
				return v, e
			}
			b, _ := json.Marshal(DiscoveryInput{Configuration: &c})
			w, e := a.requestSpeakerDiscovery(context.Background(), contracts.Request{RequestID: contracts.ID(), ItemID: r.ID, Data: b})
			if e != nil {
				t.Fatal(e)
			}
			if done := awaitWork(t, a, w.ID); done.State != "failed" {
				t.Fatal(done)
			}
			if _, e = a.Catalog.SpeakerDiscovery(context.Background(), r.ID); e == nil {
				t.Fatal("partial discovery published")
			}
		})
	}
}

type lostDiscoveryAcceptance struct {
	catalog.Catalog
	lost bool
}

func (s *lostDiscoveryAcceptance) AcceptSpeakerDiscovery(ctx context.Context, w catalog.Work, f catalog.SpeakerDiscoverySnapshot, a sd.Aggregate) (catalog.Work, error) {
	out, e := s.Catalog.AcceptSpeakerDiscovery(ctx, w, f, a)
	if e == nil && !s.lost {
		s.lost = true
		return catalog.Work{}, contracts.Fail("unavailable")
	}
	return out, e
}
func TestSpeakerDiscoveryLostAcceptanceDoesNotRepeatProvider(t *testing.T) {
	a, r, c := discoveryApp(t)
	a.Catalog = &lostDiscoveryAcceptance{Catalog: a.Catalog}
	var calls atomic.Int64
	a.DiscoveryFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
		calls.Add(1)
		return discoveryResponse(in)
	}
	input, _ := json.Marshal(DiscoveryInput{Configuration: &c})
	w, e := a.requestSpeakerDiscovery(context.Background(), contracts.Request{RequestID: contracts.ID(), ItemID: r.ID, Data: input})
	if e != nil {
		t.Fatal(e)
	}
	if done := awaitWork(t, a, w.ID); done.State != "succeeded" || calls.Load() != 1 {
		t.Fatal(done, calls.Load())
	}
}
func TestSpeakerDiscoveryDeadlineAndConcurrentCorrection(t *testing.T) {
	for _, mode := range []string{"deadline", "correction", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			a, r, c := discoveryApp(t)
			entered := make(chan struct{})
			release := make(chan struct{})
			if mode == "deadline" {
				c.Limits.TimeoutMS = 100
			}
			a.DiscoveryFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
				close(entered)
				if mode == "deadline" {
					<-ctx.Done()
					return assistance.Generated{}, assistance.ProviderContextError(ctx)
				}
				select {
				case <-release:
					return discoveryResponse(in)
				case <-ctx.Done():
					return assistance.Generated{}, assistance.ProviderContextError(ctx)
				}
			}
			input, _ := json.Marshal(DiscoveryInput{Configuration: &c})
			w, e := a.requestSpeakerDiscovery(context.Background(), contracts.Request{RequestID: contracts.ID(), ItemID: r.ID, Data: input})
			if e != nil {
				t.Fatal(e)
			}
			select {
			case <-entered:
			case <-time.After(qualificationtiming.Duration("readiness_seconds")):
				t.Fatal("provider did not start")
			}
			if mode == "correction" {
				if _, e = a.Catalog.PutSpeaker(context.Background(), contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "New canonical authority"}}); e != nil {
					t.Fatal(e)
				}
				close(release)
			}
			if mode == "cancel" {
				response := realRequest(a, "work.cancel", w.ID, nil)
				if response.Error != nil {
					t.Fatal(response.Error)
				}
				close(release)
			}
			var done catalog.Work
			if mode == "cancel" {
				done, e = a.Catalog.Work(context.Background(), w.ID)
				if e != nil {
					t.Fatal(e)
				}
			} else {
				done = awaitWork(t, a, w.ID)
			}
			if done.State != "failed" && done.State != "cancelled" {
				t.Fatal(done)
			}
			if _, e = a.Catalog.SpeakerDiscovery(context.Background(), r.ID); e == nil {
				t.Fatal("stale or cancelled output published")
			}
		})
	}
}

func TestSpeakerDiscoveryChunkBoundsAndAtomicFailure(t *testing.T) {
	for _, mode := range []string{"partial", "budget", "malformed-second", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			a, r, c := discoveryAppCues(t, 3)
			c.Limits.MaxContextRows = 1
			limits := sd.Limits{MaxCues: 3, MaxCalls: 3, MaxObservations: 3}
			if mode == "partial" {
				limits.MaxCalls = 2
			}
			if mode == "budget" {
				limits.MaxObservations = 1
			}
			var calls atomic.Int64
			a.DiscoveryFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
				n := calls.Add(1)
				if mode == "malformed-second" && n == 2 {
					return assistance.Generated{Raw: []byte("{"), ObservedRoute: "local"}, nil
				}
				if mode == "oversize" {
					return assistance.Generated{Raw: []byte(strings.Repeat("x", c.Limits.MaxResponseBytes+1)), ObservedRoute: "local"}, nil
				}
				return discoveryResponse(in)
			}
			data, _ := json.Marshal(DiscoveryInput{Configuration: &c, Limits: limits})
			w, e := a.requestSpeakerDiscovery(context.Background(), contracts.Request{RequestID: contracts.ID(), ItemID: r.ID, Data: data})
			if e != nil {
				t.Fatal(e)
			}
			done := awaitWork(t, a, w.ID)
			current, e := a.Catalog.SpeakerDiscovery(context.Background(), r.ID)
			if mode == "partial" {
				if e != nil || done.State != "succeeded" || calls.Load() != 2 {
					t.Fatal(done, e, calls.Load())
				}
				var v sd.Aggregate
				json.Unmarshal(current.Aggregate, &v)
				if v.Coverage.Selected != 2 || v.Coverage.Total != 3 || v.Coverage.Omitted != 1 || len(v.Hypotheses) != 2 {
					t.Fatal(v)
				}
			} else if done.State != "failed" || e == nil {
				t.Fatal("partial output published", done, e)
			}
		})
	}
}

func TestSpeakerDiscoveryHTTPAndOllama(t *testing.T) {
	for _, adapter := range []string{"insonic-http", "ollama"} {
		for _, mode := range []string{"local", "hosted", "redirect", "cloud-local"} {
			if adapter == "insonic-http" && mode == "cloud-local" {
				continue
			}
			t.Run(adapter+"/"+mode, func(t *testing.T) {
				a, r, c := discoveryApp(t)
				var generationCalls atomic.Int64
				secretID := contracts.ID()
				a.secrets = discoverySecrets{ID: secretID}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Header.Get("Authorization") != "Bearer discovery-fixture-secret" {
						t.Error("selected credential missing")
					}
					if req.URL.Path == "/api/show" {
						if mode == "cloud-local" {
							w.Write([]byte(`{"capabilities":["completion"],"remote_model":"cloud"}`))
						} else {
							w.Write([]byte(`{"capabilities":["completion"],"details":{"format":"gguf"}}`))
						}
						return
					}
					generationCalls.Add(1)
					if mode == "redirect" {
						w.Header().Set("Location", "/followed")
						w.WriteHeader(302)
						return
					}
					raw := `{"version":1,"observations":[{"class":"self-identification","name":"Alex","voice_id":"voice-1","references":[{"cue_id":"cue-000000","quote":"I am Alex."}],"uncertain":false}]}`
					if adapter == "ollama" {
						json.NewEncoder(w).Encode(map[string]any{"done": true, "message": map[string]any{"role": "assistant", "content": raw}})
					} else {
						w.Write([]byte(raw))
					}
				}))
				defer server.Close()
				c.Endpoint = server.URL
				c.Adapter = adapter
				c.CredentialID = secretID
				if mode == "hosted" {
					c.Route = "hosted"
				}
				a.hostedClient = server.Client()
				data, _ := json.Marshal(DiscoveryInput{Configuration: &c})
				w, e := a.requestSpeakerDiscovery(context.Background(), contracts.Request{RequestID: contracts.ID(), ItemID: r.ID, Data: data})
				if e != nil {
					t.Fatal(e)
				}
				done := awaitWork(t, a, w.ID)
				current, e := a.Catalog.SpeakerDiscovery(context.Background(), r.ID)
				if mode == "redirect" || mode == "cloud-local" {
					if done.State != "failed" || e == nil || generationCalls.Load() > 1 {
						t.Fatal(done, e, generationCalls.Load())
					}
					if mode == "cloud-local" && generationCalls.Load() != 0 {
						t.Fatal("source sent before local route verification")
					}
				} else if done.State != "succeeded" || e != nil {
					t.Fatal(done, e)
				}
				persisted, _ := json.Marshal([]any{done, current})
				if strings.Contains(string(persisted), "discovery-fixture-secret") {
					t.Fatal("secret persisted")
				}
			})
		}
	}
}

type discoverySecrets struct{ ID string }

func (s discoverySecrets) Resolve(ctx context.Context, id string) ([]byte, error) {
	if id != s.ID {
		return nil, contracts.Fail("not_found")
	}
	return []byte("discovery-fixture-secret"), nil
}

func (s discoverySecrets) Status(context.Context, string) (string, error) { return "available", nil }
