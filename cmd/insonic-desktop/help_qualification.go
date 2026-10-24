//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	"os"
	"sync/atomic"
	"time"

	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// This fixture has no workspace, runtime, provider or model configuration. Its
// help document is subject to the production local-assets-only response policy.
func qualifyHelpOnly() (exitCode int) {
	profile, err := qualificationWebviewProfile()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	bridge := &desktop.Bridge{}
	desktop.ConfigureHelpQualification(bridge)
	defer func() { desktop.RecordQualificationTerminal(bridge, exitCode == 0) }()
	var passed atomic.Bool
	var timedOut atomic.Bool
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
	app := &options.App{Title: "insonic", Width: 1200, Height: 800, MinWidth: 720, MinHeight: 600, StartHidden: true,
		OnStartup: func(ctx context.Context) {
			current = ctx
			go func() {
				select {
				case <-ctx.Done():
				case <-time.After(qualificationtiming.Duration("help_watchdog_seconds")):
					timedOut.Store(true)
					bridge.QualificationStep("watchdog-timeout")
					wailsruntime.Quit(ctx)
				}
			}()
		},
		OnDomReady:  func(ctx context.Context) { wailsruntime.EventsEmit(ctx, "qualification") },
		AssetServer: &assetserver.Options{Assets: desktop.Assets, Middleware: desktop.HelpAssetPolicy}, Bind: []any{bridge}}
	if profile != "" {
		app.Windows = &windows.Options{WebviewUserDataPath: profile}
	}
	err = wails.Run(app)
	if err == nil {
		err = nativeWebviewCloseError()
	}
	if err != nil || timedOut.Load() || !passed.Load() {
		fmt.Fprintln(os.Stderr, "Installed help qualification failed.")
		return 1
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": contracts.Version, "offline_help_activation": "passed", "workspace": "not-selected", "network_policy": "local-assets-only"})
	return 0
}
