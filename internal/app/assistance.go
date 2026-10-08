// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"github.com/shruggietech/insonic/schemas"
	"strings"
	"time"
)

const assistanceSetting = "query-assistance"

type assistanceResult struct {
	Proposal    contracts.QueryInput `json:"proposal"`
	Explanation string               `json:"explanation"`
	Validation  map[string]any       `json:"validation"`
	Provenance  map[string]any       `json:"provenance"`
	Execution   any                  `json:"execution,omitempty"`
}

func (a *App) assistanceConfiguration(ctx context.Context) (assistance.Config, int64, error) {
	raw, rev, e := a.Catalog.NamedSetting(ctx, assistanceSetting)
	if e != nil {
		return assistance.Config{}, rev, e
	}
	c := assistance.DefaultConfig()
	if len(raw) > 0 {
		if strictPayload(raw, &c) != nil {
			return c, rev, contracts.Fail("invalid_request")
		}
	}
	c, e = assistance.Normalize(c)
	return c, rev, e
}
func (a *App) assistanceShow(req contracts.Request) (any, error) {
	if len(req.Data) != 0 {
		return nil, contracts.Fail("invalid_request")
	}
	c, rev, e := a.assistanceConfiguration(a.ctx)
	return map[string]any{"configuration": c, "revision": rev, "workspace_id": a.Workspace.Config.WorkspaceID, "reachability": "not-probed"}, e
}
func (a *App) assistanceSet(req contracts.Request) (any, error) {
	var in struct {
		Expected      int64             `json:"expected_revision"`
		Configuration assistance.Config `json:"configuration"`
	}
	if strictPayload(req.Data, &in) != nil || in.Expected < 0 {
		return nil, contracts.Fail("invalid_request")
	}
	c, e := assistance.Normalize(in.Configuration)
	if e != nil {
		return nil, e
	}
	raw, _ := json.Marshal(c)
	receipt, e := a.Catalog.PutNamedSetting(a.ctx, req.RequestID, assistanceSetting, in.Expected, raw)
	if e != nil {
		return nil, e
	}
	return map[string]any{"configuration": c, "revision": receipt.Revision, "workspace_id": a.Workspace.Config.WorkspaceID, "reachability": "not-probed"}, nil
}
func (a *App) assistanceContext(ctx context.Context) (map[string]any, string, error) {
	if e := a.lockGraph(ctx); e != nil {
		return nil, "", e
	}
	defer a.graphMu.Unlock()
	profile, _ := json.Marshal(a.Workspace.Config.Profiles.Graph)
	caps := map[string]any{"adapter_id": a.Workspace.Config.Profiles.Graph.Adapter, "backend_version": "unknown", "state": "unavailable"}
	g, e := a.openGraph(ctx)
	if e == nil {
		caps = g.Capabilities()
	}
	return caps, documentHash(profile), nil
}
func (a *App) validateProposal(ctx context.Context, q contracts.QueryInput, c assistance.Config, adapter string) (string, error) {
	if e := q.Validate(); e != nil {
		return "rejected", e
	}
	if q.Definition.Mode == "normalized" {
		// No silent semantic rewrites: generation must choose an explicit bounded first page.
		if q.Definition.Pagination == nil || q.Definition.Pagination.Cursor != "" || q.Definition.Pagination.Limit > c.Limits.MaxRows {
			return "rejected", contracts.Fail("input_limit")
		}
		return "validated", nil
	}
	if (adapter == "ladybugdb" && q.Definition.Dialect != "ladybug-cypher") || (adapter == "arcadedb" && q.Definition.Dialect == "ladybug-cypher") {
		return "rejected", contracts.Fail("unsupported_capability")
	}
	if e := graph.ValidateNative(q.Definition.Dialect, q.Definition.Text, q.Parameters); e != nil {
		return "rejected", e
	}
	if e := a.lockGraph(ctx); e != nil {
		return "rejected", e
	}
	defer a.graphMu.Unlock()
	g, e := a.openGraph(ctx)
	if e != nil {
		return "pending", nil
	}
	_, e = g.Explain(ctx, q.Definition.Dialect, q.Definition.Text, q.Parameters)
	if e != nil {
		if typed, ok := e.(*contracts.Error); ok && (typed.Code == "unavailable" || typed.Code == "operation_failed") {
			return "pending", nil
		}
		return "rejected", e
	}
	return "validated", nil
}
func boundedExecution(value any, limit int, maxBytes int) (json.RawMessage, error) {
	raw, e := json.Marshal(value)
	if e != nil || len(raw) > maxBytes {
		return nil, contracts.Fail("output_limit")
	}
	var envelope struct {
		Items []json.RawMessage `json:"items"`
	}
	if json.Unmarshal(raw, &envelope) != nil || len(envelope.Items) > limit {
		return nil, contracts.Fail("output_limit")
	}
	return raw, nil
}
func (a *App) assist(ctx context.Context, req contracts.Request) (any, error) {
	var in struct {
		Prompt           string                `json:"prompt"`
		Configuration    *assistance.Config    `json:"configuration,omitempty"`
		SettingsRevision *int64                `json:"settings_revision,omitempty"`
		Mode             string                `json:"mode,omitempty"`
		ContextQuery     *contracts.QueryInput `json:"context_query,omitempty"`
	}
	if strictPayload(req.Data, &in) != nil || strings.TrimSpace(in.Prompt) == "" {
		return nil, contracts.Fail("invalid_request")
	}
	c, rev, e := a.assistanceConfiguration(ctx)
	if e != nil {
		return nil, e
	}
	if in.SettingsRevision != nil && *in.SettingsRevision != rev {
		return nil, contracts.Fail("conflict")
	}
	if in.Configuration != nil {
		c, e = assistance.Normalize(*in.Configuration)
		if e != nil {
			return nil, e
		}
	}
	if !c.Enabled {
		return nil, contracts.Fail("unavailable")
	}
	if len(in.Prompt) > c.Limits.MaxPromptBytes {
		return nil, contracts.Fail("input_limit")
	}
	mode := c.Mode
	if in.Mode != "" {
		mode = in.Mode
	}
	if mode != "suggest" && mode != "auto-run" {
		return nil, contracts.Fail("invalid_request")
	}
	ctx, stop := context.WithTimeout(ctx, time.Duration(c.Limits.TimeoutMS+c.Limits.QueryTimeoutMS+5000)*time.Millisecond)
	defer stop()
	caps, profile, e := a.assistanceContext(ctx)
	if e != nil {
		return nil, e
	}
	selectedAdapter, _ := caps["adapter_id"].(string)
	schema, _ := json.Marshal(map[string]any{"query_contract": json.RawMessage(schemas.QuerySchemaBytes()), "physical_types": map[string]any{"Entity": []string{"id", "workspace", "entity_id", "kind", "reference"}, "EvidenceLink": []string{"id", "kind"}}, "reference_values": "Identities only. Use normalized operations for hydrated current text and original clocks.", "allowed_dialects": assistanceDialects(selectedAdapter)})
	input := assistance.Request{ContractVersion: "1", Operation: "query-assistance", Model: c.Model, Prompt: in.Prompt, Schema: schema, Capabilities: caps, Limits: c.Limits}
	if in.ContextQuery != nil {
		q := *in.ContextQuery
		if q.Definition.Mode != "normalized" || q.Definition.Pagination == nil || q.Definition.Pagination.Limit > c.Limits.MaxContextRows {
			return nil, contracts.Fail("input_limit")
		}
		bounded, cancel := context.WithTimeout(ctx, time.Duration(c.Limits.QueryTimeoutMS)*time.Millisecond)
		value, e := a.runQuery(bounded, q, "query.run")
		cancel()
		if e != nil {
			return nil, e
		}
		input.Context, e = boundedExecution(value, c.Limits.MaxContextRows, c.Limits.MaxContextBytes)
		if e != nil {
			return nil, e
		}
	}
	generating, cancel := context.WithTimeout(ctx, time.Duration(c.Limits.TimeoutMS)*time.Millisecond)
	var p assistance.Proposal
	if a.AssistanceFixture != nil {
		p, e = a.AssistanceFixture(generating, input, c)
	} else {
		p, e = (assistance.HTTPAdapter{Secrets: a.secrets}).Propose(generating, input, c)
	}
	canceled := generating.Err() != nil
	cancel()
	if e != nil {
		return nil, e
	}
	if canceled || ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	// Injected fixtures must satisfy exactly the same untrusted response contract.
	raw, _ := json.Marshal(p)
	if len(raw) > c.Limits.MaxResponseBytes {
		return nil, contracts.Fail("output_limit")
	}
	p, e = assistance.Parse(raw)
	if e != nil {
		return nil, e
	}
	_, now, e := a.assistanceConfiguration(ctx)
	if e != nil {
		return nil, e
	}
	currentProfile, _ := json.Marshal(a.Workspace.Config.Profiles.Graph)
	if now != rev || documentHash(currentProfile) != profile {
		return nil, contracts.Fail("conflict")
	}
	validating, finish := context.WithTimeout(ctx, time.Duration(c.Limits.QueryTimeoutMS)*time.Millisecond)
	defer finish()
	state, e := a.validateProposal(validating, p.Query, c, selectedAdapter)
	if e != nil {
		return nil, e
	}
	out := assistanceResult{Proposal: p.Query, Explanation: p.Explanation, Validation: map[string]any{"status": state, "access_mode": "read-only"}, Provenance: map[string]any{"adapter": c.Adapter, "model": c.Model, "route": c.Route, "mode": mode, "settings_revision": rev, "profile_digest": profile, "capability_digest": digestJSON(caps), "context_included": len(input.Context) > 0, "context_digest": documentHash(input.Context)}}
	if e := a.assistanceFence(ctx, rev, profile); e != nil {
		return nil, e
	}
	if mode == "auto-run" {
		if state != "validated" {
			return nil, contracts.Fail("unavailable")
		}
		result, e := a.runQuery(validating, p.Query, "query.run")
		if e != nil {
			return nil, e
		}
		out.Execution, e = boundedExecution(result, c.Limits.MaxRows, c.Limits.MaxResultBytes)
		if e != nil {
			return nil, e
		}
	}
	if ctx.Err() != nil || validating.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	if e := a.assistanceFence(ctx, rev, profile); e != nil {
		return nil, e
	}
	return out, nil
}
func digestJSON(value any) string { raw, _ := json.Marshal(value); return documentHash(raw) }
func assistanceDialects(adapter string) []string {
	if adapter == "ladybugdb" {
		return []string{"ladybug-cypher"}
	}
	return []string{"arcade-opencypher", "arcade-sql"}
}

func (a *App) assistanceFence(ctx context.Context, rev int64, profile string) error {
	_, now, e := a.assistanceConfiguration(ctx)
	if e != nil {
		return e
	}
	raw, _ := json.Marshal(a.Workspace.Config.Profiles.Graph)
	if now != rev || documentHash(raw) != profile {
		return contracts.Fail("conflict")
	}
	return nil
}
