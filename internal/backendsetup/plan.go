// SPDX-License-Identifier: Apache-2.0
// Package backendsetup shares catalog-independent configuration between CLI and desktop.
package backendsetup

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func Build(raw []byte) (workspace.Config, error) {
	var c workspace.Config
	if len(raw) > 1<<20 || catalog.ValidateJSON(raw) != nil {
		return c, contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	if d.Decode(&c) != nil {
		return c, contracts.Fail("invalid_request")
	}
	if c.Version == "" {
		c.Version = contracts.Version
	}
	if !contracts.SupportedDocumentVersion(c.Version) {
		return c, contracts.Fail("incompatible_version")
	}
	if c.Kind != "" && c.Kind != "backend-plan" && c.Kind != "workspace-config" {
		return c, contracts.Fail("invalid_request")
	}
	c.Kind = "workspace-config"
	if strings.TrimSpace(c.DisplayName) == "" || !contracts.ValidTextLength(c.DisplayName, 1, 256) {
		return c, contracts.Fail("invalid_request")
	}
	if c.WorkspaceID == "" {
		c.WorkspaceID = contracts.ID()
	}
	if c.ControlDirectory == "" {
		c.ControlDirectory = ".insonic"
	}
	if c.ControlDirectory != ".insonic" {
		return c, contracts.Fail("invalid_request")
	}
	ps := []*workspace.Profile{&c.Profiles.Storage, &c.Profiles.Catalog, &c.Profiles.Graph}
	defaults := []string{"filesystem", "sqlite", "ladybugdb"}
	opts := []map[string]any{{"root": "artifacts"}, {"path": "catalog.sqlite"}, {"path": "graph/library.lbug"}}
	for i, p := range ps {
		if p.Adapter == "" {
			p.Adapter = defaults[i]
			p.Configuration = opts[i]
		}
		if p.ID == "" {
			p.ID = contracts.ID()
		}
		if p.Revision == 0 {
			p.Revision = 1
		}
		if p.Version == "" {
			p.Version = contracts.Version
		}
		if p.Adapter == "postgresql" {
			if p.Configuration == nil {
				p.Configuration = map[string]any{}
			}
			if _, ok := p.Configuration["tls_mode"]; !ok {
				p.Configuration["tls_mode"] = "verify-full"
			}
		}
	}
	raw, e := json.Marshal(c)
	if e != nil {
		return c, contracts.Condition("invalid_request", "backend_plan")
	}
	if e = schemas.ValidateWorkspace(raw); e != nil {
		return c, schemaPlanError(e)
	}
	if c.Profiles.Catalog.Adapter == "postgresql" {
		var pg catalog.PostgreSQLConfig
		if Decode(c.Profiles.Catalog.Configuration, &pg) != nil {
			return c, contracts.Fail("invalid_request")
		}
		if _, e = catalog.PostgreSQLBaseConfig(pg); e != nil {
			return c, e
		}
	}
	for _, p := range ps {
		if p.Adapter == "sqlite" {
			path, _ := p.Configuration["path"].(string)
			if strings.HasPrefix(path, "\\\\") || strings.HasPrefix(path, "//") {
				return c, contracts.Fail("invalid_request")
			}
		}
	}
	if c.Profiles.Graph.Adapter == "arcadedb" {
		u, _ := url.Parse(c.Profiles.Graph.Configuration["endpoint"].(string))
		id, _ := c.Profiles.Graph.Configuration["credential_id"].(string)
		ip := net.ParseIP(u.Hostname())
		if id != "" && u.Scheme != "https" && !strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
			return c, contracts.Condition("invalid_request", "backend_endpoint_tls")
		}
	}
	return c, nil
}
func Decode(in any, out any) error {
	raw, e := json.Marshal(in)
	if e != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func Draft(root string, c workspace.Config) (*workspace.Workspace, error) {
	p, e := filepath.Abs(root)
	if e != nil || root == "" {
		return nil, contracts.Fail("invalid_request")
	}
	if actual, e := filepath.EvalSymlinks(p); e == nil {
		p = actual
	} else if !os.IsNotExist(e) {
		return nil, contracts.Fail("unavailable")
	}
	return &workspace.Workspace{Root: p, Control: filepath.Join(p, ".insonic"), Config: c}, nil
}
