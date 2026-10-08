// SPDX-License-Identifier: Apache-2.0
package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"time"
)

func (a *App) queryCompatibility(q contracts.QueryInput) json.RawMessage {
	p := a.Workspace.Config.Profiles.Graph
	check := map[string]any{"target_profile": map[string]any{"profile_id": p.ID, "profile_revision": p.Revision, "adapter_id": p.Adapter, "contract_version": p.Version}, "adapter_id": p.Adapter, "catalog_schema_version": fmt.Sprint(catalog.SchemaVersion), "projection_schema_version": "1", "status": "pending", "diagnostics": []string{"Definition validated; backend execution has not been performed."}}
	incompatible := q.Definition.Mode == "native" && ((p.Adapter == "ladybugdb" && q.Definition.Dialect != "ladybug-cypher") || (p.Adapter == "arcadedb" && q.Definition.Dialect == "ladybug-cypher"))
	if incompatible {
		check["status"] = "incompatible"
		check["diagnostics"] = []string{"The saved native dialect is unavailable on the selected graph adapter. Save a compatible revision to execute it."}
	} else {
		a.graphMu.Lock()
		g, e := a.openGraph(a.ctx)
		if e == nil {
			if q.Definition.Mode == "native" {
				_, e = g.Explain(a.ctx, q.Definition.Dialect, q.Definition.Text, q.Parameters)
			} else {
				_, e = g.ReadRefs(a.ctx, a.Workspace.Config.WorkspaceID)
			}
		}
		if e == nil && g.Capabilities()["backend_version"] != "unknown" {
			caps := g.Capabilities()
			raw, _ := json.Marshal(caps)
			digest := sha256.Sum256(raw)
			now := time.Now().UTC()
			check["status"] = "validated"
			check["backend_version"] = caps["backend_version"]
			check["capability_fingerprint"] = hex.EncodeToString(digest[:])
			check["validated_at"] = map[string]any{"iso": now.Format(time.RFC3339Nano), "unix_ns": now.UnixNano()}
			check["diagnostics"] = []string{}
		} else if e != nil {
			if typed, ok := e.(*contracts.Error); ok {
				check["diagnostics"] = []string{"Backend validation did not complete: " + typed.Code}
			}
		}
		a.graphMu.Unlock()
	}
	raw, _ := json.Marshal([]any{check})
	return raw
}
