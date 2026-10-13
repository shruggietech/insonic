// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"path/filepath"
)

// Factory is non-nil only in explicit deterministic test construction.
type Factory func(context.Context, *workspace.Workspace, contracts.SecretProvider) (Adapter, error)

// Initialize applies the same selected path/version/schema/owner contract at bootstrap and runtime.
func Initialize(ctx context.Context, w *workspace.Workspace, p contracts.SecretProvider, factory Factory) (Adapter, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	profile := w.Config.Profiles.Graph
	var g Adapter
	var e error
	if factory != nil {
		g, e = factory(ctx, w, p)
	} else {
		switch profile.Adapter {
		case "ladybugdb":
			path, _ := profile.Configuration["path"].(string)
			if path == "" {
				return nil, contracts.Fail("invalid_request")
			}
			if !filepath.IsAbs(path) {
				path = filepath.Join(w.Root, path)
			}
			g, e = OpenLadybug(path)
		case "arcadedb":
			raw, _ := json.Marshal(profile.Configuration)
			var c ArcadeConfig
			if contracts.DecodeExplore(raw, &c) != nil {
				return nil, contracts.Fail("invalid_request")
			}
			g, e = OpenArcadeContext(ctx, c, p)
		default:
			return nil, contracts.Fail("unsupported_capability")
		}
	}
	if e != nil {
		return nil, e
	}
	if g == nil {
		return nil, contracts.Fail("unavailable")
	}
	if profile.ExpectedBackendVersion != "" && g.Capabilities()["backend_version"] != profile.ExpectedBackendVersion {
		g.Close()
		return nil, contracts.Fail("incompatible_version")
	}
	if e = g.EnsureSchema(ctx); e == nil {
		e = g.BindWorkspace(ctx, w.Config.WorkspaceID)
	}
	if e == nil && ctx.Err() != nil {
		e = contracts.Fail("cancelled")
	}
	if e != nil {
		g.Close()
		return nil, e
	}
	return g, nil
}
