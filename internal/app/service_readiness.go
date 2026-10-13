// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
)

func (a *App) serviceReadiness(ctx context.Context, req contracts.Request) (any, error) {
	c, rev, e := a.assistanceConfiguration(ctx)
	if e != nil {
		return nil, e
	}
	adapter := assistance.Ollama{Secrets: a.secrets}
	if req.Operation == "query.services" {
		var in assistance.DiscoveryInput
		if len(req.Data) > 0 && strictPayload(req.Data, &in) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		out, e := adapter.Discover(ctx, in, c)
		if e != nil {
			return out, e
		}
		raw, testRev, err := a.Catalog.NamedSetting(ctx, "query-assistance-test")
		_ = testRev
		if err == nil {
			var receipt struct {
				Selection string `json:"selection"`
				Revision  int64  `json:"configuration_revision"`
			}
			if json.Unmarshal(raw, &receipt) == nil && receipt.Selection == digestJSON(c) && receipt.Revision == rev {
				for i := range out.Candidates {
					out.Candidates[i].Tested = out.Candidates[i].Configured
				}
			}
		}
		return out, nil
	}
	var in struct {
		Expected int64 `json:"settings_revision"`
	}
	if strictPayload(req.Data, &in) != nil || in.Expected != rev {
		return nil, contracts.Condition("conflict", "revision_stale")
	}
	if c.Adapter != "ollama" || !c.Enabled {
		return nil, contracts.Fail("unsupported_capability")
	}
	request := assistance.Request{ContractVersion: "1", Operation: "query-assistance", Model: c.Model, Prompt: "List five fictional media entries. Use the normalized media-list operation with pagination limit 5.", Schema: json.RawMessage("{\"query_contract\":" + string(schemas.QuerySchemaBytes()) + "}"), Limits: c.Limits}
	proposal, e := adapter.Propose(ctx, request, c)
	if e != nil {
		return nil, e
	}
	_, now, e := a.assistanceConfiguration(ctx)
	if e != nil {
		return nil, e
	}
	if now != rev {
		return nil, contracts.Condition("conflict", "revision_stale")
	}
	raw, current, e := a.Catalog.NamedSetting(ctx, "query-assistance-test")
	_ = raw
	if e != nil {
		return nil, e
	}
	receipt := map[string]any{"selection": digestJSON(c), "configuration_revision": rev, "endpoint": c.Endpoint, "model": c.Model, "inference_route": proposal.ObservedRoute, "tested": true, "task": "query-assistance", "library_context_sent": false}
	bytes, _ := json.Marshal(receipt)
	if _, e = a.Catalog.PutNamedSetting(ctx, req.RequestID, "query-assistance-test", current, bytes); e != nil {
		return nil, e
	}
	return receipt, nil
}
