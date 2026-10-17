//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// This fixture has no workspace, runtime, provider or model configuration. Its
// help document is subject to the production local-assets-only response policy.
func qualifyHelpOnly() int {
	bridge := &desktop.Bridge{}
	desktop.ConfigureHelpQualification(bridge)
	var passed atomic.Bool
	var current context.Context
	bridge.SmokeResult = func(response contracts.Response) {
		raw, _ := json.Marshal(response.Result)
		var result map[string]any
		json.Unmarshal(raw, &result)
		passed.Store(response.Error == nil && result["ui_offline_help_activation"] == "passed" && bridge.Workspace == nil)
		if response.Error != nil {
			fmt.Fprintln(os.Stderr, response.Error.Message)
		}
		wailsruntime.Quit(current)
	}
	err := wails.Run(&options.App{Title: "insonic", Width: 1200, Height: 800, MinWidth: 720, MinHeight: 600, StartHidden: true,
		OnStartup: func(ctx context.Context) {
			current = ctx
			go func() {
				select {
				case <-ctx.Done():
				case <-time.After(25 * time.Second):
					wailsruntime.Quit(ctx)
				}
			}()
		},
		OnDomReady:  func(ctx context.Context) { wailsruntime.EventsEmit(ctx, "qualification") },
		AssetServer: &assetserver.Options{Assets: desktop.Assets, Middleware: desktop.HelpAssetPolicy}, Bind: []any{bridge}})
	if err != nil || !passed.Load() {
		fmt.Fprintln(os.Stderr, "Installed help qualification failed.")
		return 1
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": contracts.Version, "offline_help_activation": "passed", "workspace": "not-selected", "network_policy": "local-assets-only"})
	return 0
}
