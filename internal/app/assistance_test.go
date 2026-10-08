// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"testing"
	"time"
)

func assistanceConfig() assistance.Config {
	c := assistance.DefaultConfig()
	c.Enabled = true
	c.Endpoint = "http://127.0.0.1:1/query"
	c.Model = "fixture"
	return c
}
func testProposal() assistance.Proposal {
	return assistance.Proposal{ContractVersion: "1", Explanation: "List current media", Query: contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "media-list", Pagination: &contracts.QueryPagination{Limit: 5}}}}
}
func TestAssistanceSuggestAutoRunAndDirectParity(t *testing.T) {
	a := configuredApp(t)
	desktopMedia(t, a, false)
	calls := 0
	a.AssistanceFixture = func(ctx context.Context, r assistance.Request, c assistance.Config) (assistance.Proposal, error) {
		calls++
		if len(r.Context) != 0 {
			t.Error("default leaked excerpts")
		}
		if len(r.Schema) == 0 || r.Capabilities["adapter_id"] != "ladybugdb" {
			t.Error(r)
		}
		return testProposal(), nil
	}
	config := assistanceConfig()
	saved := realRequest(a, "query.assistance-set", "", map[string]any{"expected_revision": 0, "configuration": config})
	if saved.Error != nil {
		t.Fatal(saved.Error)
	}
	shown := realRequest(a, "query.assistance-show", "", nil)
	if shown.Error != nil {
		t.Fatal(shown.Error)
	}
	suggest := realRequest(a, "query.assist", "", map[string]any{"prompt": "list recordings"})
	if suggest.Error != nil {
		t.Fatal(suggest.Error)
	}
	out := suggest.Result.(assistanceResult)
	if out.Execution != nil || calls != 1 {
		t.Fatal(out, calls)
	}
	auto := realRequest(a, "query.assist", "", map[string]any{"prompt": "list recordings", "mode": "auto-run"})
	if auto.Error != nil {
		t.Fatal(auto.Error)
	}
	direct := realRequest(a, "query.run", "", testProposal().Query)
	if direct.Error != nil {
		t.Fatal(direct.Error)
	}
	assertCurrentQueryParity(t, auto.Result.(assistanceResult).Execution, direct.Result)
	if _, rev, e := a.Catalog.NamedSetting(a.ctx, "query-assistance"); e != nil || rev == 0 {
		t.Fatal(rev, e)
	}
}
func TestAssistanceRejectsUnsupportedAndStaleBeforeExecution(t *testing.T) {
	for _, q := range []contracts.QueryInput{{Definition: contracts.QueryDefinition{Mode: "native", Dialect: "ladybug-cypher", Text: "MATCH (n) DELETE n"}}, {Definition: contracts.QueryDefinition{Mode: "native", Dialect: "arcade-sql", Text: "SELECT FROM Entity"}}, {Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "media-list", Pagination: &contracts.QueryPagination{Limit: 500}}}} {
		a := configuredApp(t)
		a.AssistanceFixture = func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
			p := testProposal()
			p.Query = q
			return p, nil
		}
		if r := realRequest(a, "query.assist", "", map[string]any{"prompt": "run", "mode": "auto-run", "configuration": assistanceConfig()}); r.Error == nil {
			t.Fatal(q, r)
		}
	}
	a := configuredApp(t)
	cfg := assistanceConfig()
	if r := realRequest(a, "query.assistance-set", "", map[string]any{"expected_revision": 0, "configuration": cfg}); r.Error != nil {
		t.Fatal(r.Error)
	}
	_, rev, _ := a.Catalog.NamedSetting(a.ctx, "query-assistance")
	started := make(chan struct{})
	release := make(chan struct{})
	a.AssistanceFixture = func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
		close(started)
		<-release
		return testProposal(), nil
	}
	done := make(chan contracts.Response, 1)
	go func() {
		done <- realRequest(a, "query.assist", "", map[string]any{"prompt": "run", "mode": "auto-run"})
	}()
	<-started
	cfg.Mode = "auto-run"
	if r := realRequest(a, "query.assistance-set", "", map[string]any{"expected_revision": rev, "configuration": cfg}); r.Error != nil {
		t.Fatal(r.Error)
	}
	close(release)
	select {
	case r := <-done:
		if r.Error == nil || r.Error.Code != "conflict" {
			t.Fatal(r)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked assistance")
	}
}
func TestAssistanceCancellationDisabledAndWorkspace(t *testing.T) {
	a := configuredApp(t)
	if r := realRequest(a, "query.assist", "", map[string]any{"prompt": "test"}); r.Error == nil {
		t.Fatal("unconfigured ran")
	}
	a.AssistanceFixture = func(ctx context.Context, r assistance.Request, c assistance.Config) (assistance.Proposal, error) {
		<-ctx.Done()
		return assistance.Proposal{}, contracts.Fail("cancelled")
	}
	data, _ := json.Marshal(map[string]any{"prompt": "test", "configuration": assistanceConfig()})
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "query.assist", Data: data}
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer stop()
	r := a.DispatchContext(ctx, req)
	if r.Error == nil || r.Error.Code != "cancelled" {
		t.Fatal(r)
	}
	req.WorkspaceID = contracts.ID()
	if r = a.Dispatch(req); r.Error == nil || r.Error.Code != "workspace_mismatch" {
		t.Fatal(r)
	}
}

func TestAssistanceExplicitContextAndBounds(t *testing.T) {
	a := configuredApp(t)
	desktopMedia(t, a, false)
	calls := 0
	a.AssistanceFixture = func(_ context.Context, r assistance.Request, _ assistance.Config) (assistance.Proposal, error) {
		calls++
		if len(r.Context) == 0 || len(r.Context) > 32768 {
			t.Error("missing or excessive elected excerpt")
		}
		return testProposal(), nil
	}
	q := testProposal().Query
	r := realRequest(a, "query.assist", "", map[string]any{"prompt": "list media", "configuration": assistanceConfig(), "context_query": q})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.Result.(assistanceResult).Provenance["context_included"] != true {
		t.Fatal("context not disclosed")
	}
	q.Definition.Pagination.Limit = 26
	r = realRequest(a, "query.assist", "", map[string]any{"prompt": "list", "configuration": assistanceConfig(), "context_query": q})
	if r.Error == nil || calls != 1 {
		t.Fatal("context exceeded limit")
	}
	c := assistanceConfig()
	c.Limits.MaxPromptBytes = 2
	r = realRequest(a, "query.assist", "", map[string]any{"prompt": "long", "configuration": c})
	if r.Error == nil || calls != 1 {
		t.Fatal("prompt exceeded limit")
	}
	c = assistanceConfig()
	c.Limits.MaxResultBytes = 1
	a.AssistanceFixture = func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
		return testProposal(), nil
	}
	r = realRequest(a, "query.assist", "", map[string]any{"prompt": "list", "configuration": c, "mode": "auto-run"})
	if r.Error == nil || r.Error.Code != "output_limit" {
		t.Fatal(r)
	}
}

func TestAssistanceRestartAndExactExcerpt(t *testing.T) {
	a := configuredApp(t)
	entry := desktopMedia(t, a, false)
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	desktopRecording(t, a, entry, 0, raw)
	cfg := assistanceConfig()
	saved := realRequest(a, "query.assistance-set", "", map[string]any{"expected_revision": 0, "configuration": cfg})
	if saved.Error != nil {
		t.Fatal(saved.Error)
	}
	_, rev, e := a.assistanceConfiguration(a.ctx)
	if e != nil {
		t.Fatal(e)
	}
	w := a.Workspace
	a.Close()
	reopened, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	got, now, e := reopened.assistanceConfiguration(reopened.ctx)
	if e != nil || now != rev || got.Endpoint != cfg.Endpoint || got.Model != cfg.Model {
		t.Fatal("settings did not survive restart", e)
	}
	reopened.AssistanceFixture = func(_ context.Context, input assistance.Request, _ assistance.Config) (assistance.Proposal, error) {
		var value struct {
			Items []map[string]any `json:"items"`
		}
		if e := json.Unmarshal(input.Context, &value); e != nil {
			t.Fatal(e)
		}
		found := false
		for _, row := range value.Items {
			if start, ok := row["source_start_us"]; ok {
				if _, ok := start.(string); !ok {
					t.Fatal("clock was coerced", start)
				}
				if _, ok := row["source_end_us"].(string); !ok {
					t.Fatal("end clock was coerced")
				}
				if row["media_id"] != entry.ID {
					t.Fatal("source identity changed")
				}
				found = true
			}
		}
		if !found {
			t.Fatal("no original clock in elected current excerpt")
		}
		return testProposal(), nil
	}
	q := testProposal().Query
	q.Definition.Operation = "text-search"
	out := realRequest(reopened, "query.assist", "", map[string]any{"prompt": "find evidence", "context_query": q})
	if out.Error != nil {
		t.Fatal(out.Error)
	}
}

// Projection receipts may advance catalog audit revision between equivalent reads.
// Every other field, including exact source values, must remain identical.
func assertCurrentQueryParity(t *testing.T, first, second any) {
	t.Helper()
	raw, _ := json.Marshal(first)
	later, _ := json.Marshal(second)
	var a, b map[string]json.RawMessage
	if json.Unmarshal(raw, &a) != nil || json.Unmarshal(later, &b) != nil {
		t.Fatal("invalid query result")
	}
	var before, after int64
	if json.Unmarshal(a["catalog_revision"], &before) != nil || json.Unmarshal(b["catalog_revision"], &after) != nil || after < before {
		t.Fatal("revision regressed")
	}
	delete(a, "catalog_revision")
	delete(b, "catalog_revision")
	raw, _ = json.Marshal(a)
	later, _ = json.Marshal(b)
	if string(raw) != string(later) {
		t.Fatal("query content differs", string(raw), string(later))
	}
}
