// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"embed"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"time"
)

//go:embed index.html assets
var Assets embed.FS

type Bridge struct {
	Workspace     *workspace.Workspace
	CLIExecutable string
	SmokeResult   func(contracts.Response)
}

func (b *Bridge) CompleteSmoke(response contracts.Response) {
	if b.SmokeResult != nil {
		b.SmokeResult(response)
	}
}
func (b *Bridge) Show() contracts.Response {
	if b.Workspace == nil {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Fail("not_found")}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := local.Ensure(ctx, b.Workspace, b.CLIExecutable); err != nil {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Fail("unavailable")}
	}
	response, err := local.Call(ctx, b.Workspace, contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: b.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "workspace.show"})
	if err != nil {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Fail("unavailable")}
	}
	return response
}

func Qualify() (map[string]any, error) {
	if _, err := Assets.ReadFile("assets/help/index.html"); err != nil {
		return nil, contracts.Fail("unavailable")
	}
	if _, err := Assets.ReadFile("assets/interface.css"); err != nil {
		return nil, contracts.Fail("unavailable")
	}
	directory, err := os.MkdirTemp("", "insonic-desktop-*")
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(directory)
	w, err := workspace.Init(directory, "Desktop qualification")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- local.Serve(ctx, w, local.Options{Ready: ready}) }()
	select {
	case <-ready:
	case err := <-done:
		return nil, err
	case <-time.After(5 * time.Second):
		return nil, contracts.Fail("unavailable")
	}
	bridge := &Bridge{Workspace: w}
	response := bridge.Show()
	cancel()
	if err := <-done; err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, response.Error
	}
	return map[string]any{"schema_version": contracts.Version, "desktop_bridge": "passed", "offline_help": "packaged", "workspace_id": w.Config.WorkspaceID}, nil
}

func SiblingCLI() string {
	executable, _ := os.Executable()
	name := "insonic"
	if filepath.Ext(executable) == ".exe" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(executable), name)
}
