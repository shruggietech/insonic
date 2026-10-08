// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed index.html all:assets
var Assets embed.FS

type Bridge struct {
	mu            sync.RWMutex
	tickets       map[string]playbackTicket
	pick          func(string) (string, error)
	playbackCall  func(*workspace.Workspace, contracts.Request) contracts.Response
	qualification map[string]any
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
	return b.Operate(contracts.Request{Operation: "workspace.show"})
}

func operationContext(operation string) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), local.OperationTimeout(operation, 10*time.Second))
}

// Operate uses the same versioned runtime contract as CLI domain commands.
func (b *Bridge) Operate(request contracts.Request) contracts.Response {
	b.mu.RLock()
	selected := &Bridge{Workspace: b.Workspace, CLIExecutable: b.CLIExecutable}
	b.mu.RUnlock()
	return selected.operate(request)
}

func (b *Bridge) operate(request contracts.Request) contracts.Response {
	if b.Workspace == nil {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Fail("not_found")}
	}
	if request.WorkspaceID != "" && request.WorkspaceID != b.Workspace.Config.WorkspaceID {
		return b.failed(contracts.Fail("workspace_mismatch"))
	}
	if request.Version != "" && request.Version != contracts.Version {
		return b.failed(contracts.Fail("incompatible_version"))
	}
	if request.Kind == "" {
		request.Kind = "runtime-request"
	}
	request.Version = contracts.Version
	request.WorkspaceID = b.Workspace.Config.WorkspaceID
	if request.RequestID == "" {
		request.RequestID = contracts.ID()
	}
	data, e := json.Marshal(request)
	if e != nil || schemas.ValidateRequest(data) != nil {
		return b.failed(contracts.Fail("invalid_request"))
	}
	ctx, cancel := operationContext(request.Operation)
	defer cancel()
	if err := local.Ensure(ctx, b.Workspace, b.CLIExecutable); err != nil {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Fail("unavailable")}
	}
	response, err := local.Call(ctx, b.Workspace, request)
	if err != nil {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Fail("unavailable")}
	}
	return response
}

func (b *Bridge) failed(e error) contracts.Response {
	typed, ok := e.(*contracts.Error)
	if !ok {
		typed = contracts.Fail("operation_failed")
	}
	response := contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: typed}
	if b.Workspace != nil {
		response.WorkspaceID = b.Workspace.Config.WorkspaceID
	}
	return response
}

// Credential accepts only newly entered input through the native bridge. Saved
// values never appear in the response. Bootstrap does not require the catalog.
func (b *Bridge) Credential(args []string, input string) contracts.Response {
	b.mu.RLock()
	selected := &Bridge{Workspace: b.Workspace, CLIExecutable: b.CLIExecutable}
	b.mu.RUnlock()
	return selected.credential(args, input)
}

func (b *Bridge) credential(args []string, input string) contracts.Response {
	if b.Workspace == nil {
		return b.failed(contracts.Fail("not_found"))
	}
	if len(args) == 0 {
		return b.failed(contracts.Fail("invalid_request"))
	}
	if args[0] == "select" {
		if len(args) != 2 || args[1] != "native" && args[1] != "vault" && args[1] != "session" {
			return b.failed(contracts.Fail("invalid_request"))
		}
	} else if args[0] == "unlock" || args[0] == "load" {
		if len(args) != 1 {
			return b.failed(contracts.Fail("invalid_request"))
		}
	} else if len(args) != 2 || !contracts.ValidID(args[1]) || (args[0] != "add" && args[0] != "replace" && args[0] != "delete" && args[0] != "status") {
		return b.failed(contracts.Fail("invalid_request"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	mode, e := secrets.ReadSelection(b.Workspace)
	if e != nil {
		return b.failed(e)
	}
	// Selection and status never consume supplied credential input. Deletion may
	// need a vault passphrase when no unlocked owner exists.
	raw := []byte("{}")
	if args[0] == "add" || args[0] == "replace" || args[0] == "unlock" || args[0] == "load" || args[0] == "delete" && mode == "vault" {
		if strings.TrimSpace(input) != "" {
			raw = []byte(input)
		}
	}
	defer clear(raw)
	protected, e := credentialcmd.ReadInput(bytes.NewReader(raw))
	if e != nil {
		return b.failed(e)
	}
	defer protected.Close()
	if args[0] == "unlock" && (mode != "vault" || len(protected.Passphrase) == 0) {
		return b.failed(contracts.Fail("invalid_request"))
	}
	if args[0] == "load" && (mode == "native" || mode == "vault" && len(protected.Passphrase) == 0 || mode == "session" && protected.Session == nil) {
		return b.failed(contracts.Fail("invalid_request"))
	}
	envelope, e := json.Marshal(struct {
		Arguments []string        `json:"arguments"`
		Input     json.RawMessage `json:"input,omitempty"`
	}{args, json.RawMessage(raw)})
	if e != nil {
		return b.failed(contracts.Fail("invalid_request"))
	}
	defer clear(envelope)
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: b.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "credentials." + args[0], Data: envelope}
	// Reuse a live manager first, including vault status and backend selection.
	// An application error is final; it must not fall back to a different owner.
	if out, e := local.Call(ctx, b.Workspace, request); e == nil {
		return out
	} else if typed, ok := e.(*contracts.Error); !ok || typed.Code != "unavailable" {
		return b.failed(e)
	}
	if args[0] == "unlock" || args[0] == "load" || mode == "session" && args[0] != "select" && args[0] != "status" {
		startup := []byte(nil)
		if args[0] == "unlock" || args[0] == "load" || protected.Session != nil {
			startup = raw
		}
		if e = local.EnsureInput(ctx, b.Workspace, b.CLIExecutable, startup); e != nil {
			return b.failed(e)
		}
		if args[0] == "unlock" || args[0] == "load" {
			return contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: b.Workspace.Config.WorkspaceID, Result: map[string]any{"backend": mode, "state": "configured"}}
		}
		out, e := local.Call(ctx, b.Workspace, request)
		if e != nil {
			return b.failed(e)
		}
		return out
	}
	result, e := credentialcmd.Execute(ctx, b.Workspace, args, bytes.NewReader(raw))
	if e != nil {
		return b.failed(e)
	}
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: b.Workspace.Config.WorkspaceID, Result: result}
}
func Qualify() (map[string]any, error) {
	if err := verifyHelp(); err != nil {
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
