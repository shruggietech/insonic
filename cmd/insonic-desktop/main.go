//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"time"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--qualification" {
		result, err := desktop.Qualify()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		json.NewEncoder(os.Stdout).Encode(result)
		return
	}
	bridge := &desktop.Bridge{CLIExecutable: desktop.SiblingCLI()}
	smoke := len(os.Args) == 2 && os.Args[1] == "--webview-qualification"
	var startup func(context.Context)
	var domReady func(context.Context)
	var smokeContext context.Context
	smokePassed := false
	if smoke {
		directory, err := os.MkdirTemp("", "insonic-webview-*")
		if err != nil {
			os.Exit(1)
		}
		defer os.RemoveAll(directory)
		w, err := workspace.Init(directory, "Webview qualification")
		if err != nil {
			os.Exit(1)
		}
		bridge.Workspace = w
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ready := make(chan struct{})
		done := make(chan error, 1)
		go func() { done <- local.Serve(ctx, w, local.Options{Ready: ready}) }()
		select {
		case <-ready:
		case <-time.After(5 * time.Second):
			os.Exit(1)
		}
		defer func() { cancel(); <-done }()
		startup = func(ctx context.Context) {
			smokeContext = ctx
			go func() {
				select {
				case <-ctx.Done():
				case <-time.After(15 * time.Second):
					wailsruntime.Quit(ctx)
				}
			}()
		}
		domReady = func(ctx context.Context) { wailsruntime.EventsEmit(ctx, "qualification") }
		bridge.SmokeResult = func(response contracts.Response) {
			smokePassed = response.Error == nil && response.WorkspaceID == w.Config.WorkspaceID && contracts.ValidID(response.SessionID)
			wailsruntime.Quit(smokeContext)
		}
	}
	if len(os.Args) == 3 && os.Args[1] == "--workspace" {
		w, err := workspace.Open(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		bridge.Workspace = w
	}
	if err := wails.Run(&options.App{Title: "insonic", Width: 900, Height: 720, StartHidden: smoke, OnStartup: startup, OnDomReady: domReady, AssetServer: &assetserver.Options{Assets: desktop.Assets}, Bind: []any{bridge}}); err != nil {
		fmt.Fprintln(os.Stderr, "Desktop startup failed.")
		os.Exit(1)
	}
	if smoke {
		if !smokePassed {
			fmt.Fprintln(os.Stderr, "Native webview qualification failed.")
			os.Exit(1)
		}
		json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": contracts.Version, "native_webview": "passed", "frontend_bridge_ipc": "passed"})
	}
}
