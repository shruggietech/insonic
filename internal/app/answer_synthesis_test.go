// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"strings"
	"testing"
	"time"
)

func answerSynthesisApp(t *testing.T) (*App, answers.Package, assistance.SynthesisConfig) {
	t.Helper()
	a := configuredApp(t)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	desktopRecording(t, a, desktopMedia(t, a, false), 0, doc)
	p, err := a.prepareAnswer(context.Background(), answers.Request{Version: contracts.Version, Question: "What was said?"})
	if err != nil {
		t.Fatal(err)
	}
	c := assistance.DefaultConfig()
	c.Enabled = true
	c.Endpoint = "http://127.0.0.1:1"
	c.Model = "fixture"
	cfg, err := assistance.NormalizeSynthesis(assistance.SynthesisConfig{Provider: c})
	if err != nil {
		t.Fatal(err)
	}
	return a, p.Package, cfg
}
func fixtureAnswer(in assistance.Request) (answers.Envelope, error) {
	var input answers.SynthesisRequest
	if err := json.Unmarshal(in.Context, &input); err != nil {
		return answers.Envelope{}, err
	}
	if len(input.Evidence) == 0 {
		return answers.Envelope{}, contracts.Fail("invalid_request")
	}
	item := input.Evidence[0]
	return answers.Envelope{Version: contracts.Version, PackageID: input.PackageID, State: "partial", Claims: []answers.Claim{{ID: "claim", Kind: "quote", Text: item.Text, Support: "supported", Citations: []answers.Citation{{EvidenceID: item.ID, Reference: item.Reference, Quote: item.Text}}}}}, nil
}
func TestAnswerSynthesisRepairAndUnverifiable(t *testing.T) {
	for _, repairSucceeds := range []bool{false, true} {
		a, p, c := answerSynthesisApp(t)
		calls := 0
		a.AnswerFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
			calls++
			e, err := fixtureAnswer(in)
			if err != nil {
				return assistance.Generated{}, err
			}
			if calls == 1 || !repairSucceeds {
				e.Claims[0].Citations[0].Quote = "fabricated"
			}
			raw, _ := json.Marshal(e)
			return assistance.Generated{Raw: raw, ObservedRoute: "local"}, nil
		}
		result, err := a.synthesizeAnswer(context.Background(), p, c, nil)
		if err != nil || calls != 2 || len(result.Attempts) != 2 {
			t.Fatal(calls, result, err)
		}
		if repairSucceeds && result.Answer.State != "partial" || !repairSucceeds && result.Answer.State != "insufficient" {
			t.Fatal(result)
		}
		if result.Attempts[0].ResponseDigest == "" {
			t.Fatal("missing attempt identity")
		}
	}
}
func TestAnswerSynthesisNoRerouteAndSourceFence(t *testing.T) {
	a, p, c := answerSynthesisApp(t)
	calls := 0
	a.AnswerFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
		calls++
		return assistance.Generated{}, contracts.Condition("unavailable", "provider_auth")
	}
	if _, err := a.synthesizeAnswer(context.Background(), p, c, nil); err == nil || calls != 1 {
		t.Fatal("transport failure repaired or ignored", calls, err)
	}
	a.AnswerFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
		calls++
		return assistance.Generated{Raw: []byte("{}"), ObservedRoute: "hosted"}, nil
	}
	if _, err := a.synthesizeAnswer(context.Background(), p, c, nil); err == nil {
		t.Fatal("changed route accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls = 0
	if _, err := a.synthesizeAnswer(ctx, p, c, nil); err == nil || calls != 0 {
		t.Fatal("cancelled inference", calls, err)
	}
}

func TestAnswerSynthesisTypedRepairGuidance(t *testing.T) {
	for _, field := range []string{"reference", "limitations", "version"} {
		t.Run(field, func(t *testing.T) {
			a, p, c := answerSynthesisApp(t)
			calls := 0
			var deadline time.Time
			var schema string
			a.AnswerFixture = func(ctx context.Context, in assistance.Request, c assistance.Config) (assistance.Generated, error) {
				calls++
				current, _ := ctx.Deadline()
				if calls == 1 {
					deadline = current
					schema = string(in.Schema)
				} else if current != deadline || string(in.Schema) != schema {
					t.Fatal("repair changed enclosing deadline or contract")
				}
				var input map[string]any
				json.Unmarshal(in.Context, &input)
				if calls == 2 {
					detail, ok := input["repair_detail"].(map[string]any)
					if !ok {
						t.Fatal("missing actionable detail")
					}
					v, ok := detail["validation"].(map[string]any)
					if !ok || !strings.Contains(v["pointer"].(string), field) {
						t.Fatal("missing field pointer", detail)
					}
					if field != "version" && v["expected"] != map[string]string{"reference": "object", "limitations": "string"}[field] {
						t.Fatal("missing expected type", v)
					}
					if strings.Contains(string(in.Context), "provider-private-sentinel") {
						t.Fatal("provider output echoed")
					}
				}
				answer, err := fixtureAnswer(in)
				if err != nil {
					return assistance.Generated{}, err
				}
				raw, _ := json.Marshal(answer)
				if calls == 1 {
					var bad map[string]any
					json.Unmarshal(raw, &bad)
					switch field {
					case "reference":
						bad["claims"].([]any)[0].(map[string]any)["citations"].([]any)[0].(map[string]any)["reference"] = "provider-private-sentinel"
					case "limitations":
						bad["limitations"] = []any{map[string]any{"text": "provider-private-sentinel"}}
					case "version":
						bad["version"] = "1"
					}
					raw, _ = json.Marshal(bad)
				}
				return assistance.Generated{Raw: raw, ObservedRoute: "local"}, nil
			}
			out, err := a.synthesizeAnswer(context.Background(), p, c, nil)
			if err != nil || calls != 2 || out.Answer.State != "partial" {
				t.Fatal(calls, out, err)
			}
		})
	}
}
