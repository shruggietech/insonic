// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/graph"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
)

// Provision is fresh-target initialization. It never replaces a populated workspace.
// The generated configuration itself is the retry identity.
func Provision(ctx context.Context, root string, c workspace.Config, input *credentialcmd.Input) (*workspace.Workspace, error) {
	return ProvisionMode(ctx, root, c, input, "native")
}

type Provisioner struct{ GraphFactory graph.Factory }

func ProvisionMode(ctx context.Context, root string, c workspace.Config, input *credentialcmd.Input, mode string) (*workspace.Workspace, error) {
	return (Provisioner{}).ProvisionMode(ctx, root, c, input, mode)
}
func (svc Provisioner) ProvisionMode(ctx context.Context, root string, c workspace.Config, input *credentialcmd.Input, mode string) (*workspace.Workspace, error) {
	if mode != "native" && mode != "vault" && mode != "session" {
		return nil, contracts.Fail("invalid_request")
	}
	raw, _ := json.Marshal(c)
	var e error
	c, e = Build(raw)
	if e != nil {
		return nil, e
	}
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	existing, e := workspace.Open(root)
	if e == nil {
		a, _ := json.Marshal(existing.Config)
		b, _ := json.Marshal(c)
		if bytes.Equal(a, b) {
			return existing, nil
		}
		return nil, contracts.Fail("conflict")
	}
	w, e := Draft(root, c)
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(w.Root, 0700); e != nil {
		return nil, contracts.Fail("unavailable")
	}
	canonical, e := filepath.EvalSymlinks(w.Root)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	w.Root = canonical
	w.Control = filepath.Join(canonical, ".insonic")
	created := false
	if e = os.Mkdir(w.Control, 0700); e == nil {
		created = true
	} else if !os.IsExist(e) {
		return nil, contracts.Fail("unavailable")
	}
	if e = workspace.SecureDirectory(w.Control, created); e != nil {
		return nil, e
	}
	lock, e := w.Lock()
	if e != nil {
		return nil, e
	}
	defer lock.Close()
	defer lock.Unlock()
	if _, e = os.Lstat(filepath.Join(w.Control, "workspace.json")); !os.IsNotExist(e) {
		return nil, contracts.Fail("conflict")
	}
	if e = retainIntent(w, mode); e != nil {
		return nil, e
	}
	var provider *secrets.Manager
	needs := false
	references := map[string]bool{}
	for _, p := range []workspace.Profile{c.Profiles.Storage, c.Profiles.Catalog, c.Profiles.Graph} {
		if id, _ := p.Configuration["credential_id"].(string); id != "" {
			needs = true
			references[id] = true
		}
	}
	if needs || input != nil {
		filtered := &credentialcmd.Input{}
		if input != nil {
			filtered.Passphrase = input.Passphrase
			if mode == "session" {
				filtered.Session = input.Session
			}
		}
		provider, e = credentialcmd.ProviderMode(w, mode, filtered)
		if e != nil {
			return nil, e
		}
		defer provider.Close()
		if input != nil {
			for id, value := range input.Session {
				if !references[id] {
					return nil, contracts.Fail("invalid_request")
				}
				if e = provider.Add(ctx, id, value); e != nil {
					old, err := provider.Resolve(ctx, id)
					same := err == nil && bytes.Equal(old, value)
					clear(old)
					if !same {
						if err != nil {
							return nil, e
						}
						if e = provider.Replace(ctx, id, value); e != nil {
							return nil, e
						}
					}
				}
			}
		}
	}
	store, e := catalog.OpenWorkspace(ctx, w, provider, true)
	if e != nil {
		return nil, e
	}
	defer store.Close()
	if e = store.RegisterWorkspace(ctx, w); e != nil {
		return nil, e
	}
	artifacts, e := artifact.OpenStore(ctx, c.Profiles.Storage, w.Control, provider)
	if e != nil {
		return nil, e
	}
	defer artifacts.Close()
	g, e := graph.Initialize(ctx, w, provider, svc.GraphFactory)
	if e != nil {
		return nil, e
	}
	defer g.Close()
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	if mode != "native" {
		if e = secrets.Select(w, mode); e != nil {
			return nil, e
		}
	}
	return workspace.InitConfig(w.Root, c)
}
