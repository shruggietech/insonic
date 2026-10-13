// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/graph"
	"github.com/shruggietech/insonic/internal/workspace"
)

type fixtureGraph struct {
	graph.Adapter
	initialized bool
	bound       string
	schemaErr   error
	bindErr     error
	closed      bool
	version     string
}

func (g *fixtureGraph) Capabilities() map[string]any {
	return map[string]any{"backend_version": g.version}
}
func (g *fixtureGraph) EnsureSchema(context.Context) error { g.initialized = true; return g.schemaErr }
func (g *fixtureGraph) BindWorkspace(_ context.Context, id string) error {
	g.bound = id
	return g.bindErr
}
func (g *fixtureGraph) Close() { g.closed = true }
func fixtureFactory(context.Context, *workspace.Workspace, contracts.SecretProvider) (graph.Adapter, error) {
	return &fixtureGraph{version: "fixture"}, nil
}
func provisionFixture(ctx context.Context, root string, c workspace.Config, input *credentialcmd.Input) (*workspace.Workspace, error) {
	return (Provisioner{GraphFactory: fixtureFactory}).ProvisionMode(ctx, root, c, input, "native")
}
