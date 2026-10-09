//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"sync/atomic"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) == 2 && os.Args[1] == "--qualification" {
		result, err := desktop.Qualify()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		json.NewEncoder(os.Stdout).Encode(result)
		return 0
	}
	bridge := &desktop.Bridge{CLIExecutable: desktop.SiblingCLI()}
	closePlayback, err := desktop.StartPlaybackTransport(bridge)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Desktop playback transport unavailable.")
		return 1
	}
	defer closePlayback()
	smoke := len(os.Args) == 2 && os.Args[1] == "--webview-qualification"
	var startup func(context.Context)
	var domReady func(context.Context)
	var smokeContext context.Context
	var smokePassed atomic.Bool
	if smoke {
		directory, err := os.MkdirTemp("", "insonic-webview-*")
		if err != nil {
			return 1
		}
		defer os.RemoveAll(directory)
		w, err := workspace.Init(directory, "Webview qualification")
		if err != nil {
			return 1
		}
		bridge.Workspace = w
		if err := desktop.ConfigureQualificationTools(w); err != nil {
			fmt.Fprintln(os.Stderr, "Qualification companions unavailable.")
			return 1
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ready := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- local.Serve(ctx, w, local.Options{Ready: ready, AssistanceFixture: func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
				return assistance.Proposal{ContractVersion: "1", Explanation: "Deterministic current media suggestion", Query: contracts.QueryInput{Title: "Qualification assistance", Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "media-list", Pagination: &contracts.QueryPagination{Limit: 5}}}}, nil
			}})
		}()
		select {
		case <-ready:
		case <-done:
			return 1
		case <-time.After(5 * time.Second):
			cancel()
			<-done
			return 1
		}
		defer func() { cancel(); <-done }()
		if err := desktop.PrepareWebviewQualification(bridge); err != nil {
			fmt.Fprintln(os.Stderr, "Desktop media qualification preparation failed:", err)
			return 1
		}
		startup = func(ctx context.Context) {
			smokeContext = ctx
			go func() {
				select {
				case <-ctx.Done():
				case <-time.After(80 * time.Second):
					wailsruntime.Quit(ctx)
				}
			}()
		}
		domReady = func(ctx context.Context) { wailsruntime.EventsEmit(ctx, "qualification") }
		bridge.SmokeResult = func(response contracts.Response) {
			if response.Error != nil {
				fmt.Fprintln(os.Stderr, response.Error.Message)
			}
			passed := response.Error == nil && response.WorkspaceID == w.Config.WorkspaceID && contracts.ValidID(response.SessionID)
			raw, _ := json.Marshal(response.Result)
			var result map[string]any
			json.Unmarshal(raw, &result)
			for _, name := range []string{"ui_library_import", "ui_metadata_date", "ui_current_assembly", "ui_audio_playback", "ui_video_source_audio_playback", "ui_cue_seek", "ui_terms", "ui_speakers", "ui_pipelines", "ui_jobs", "ui_settings", "ui_keyboard_help", "ui_explore_calendar", "ui_explore_query", "ui_explore_graph", "ui_query_assistance"} {
				passed = passed && result[name] == "passed"
			}
			smokePassed.Store(passed)
			wailsruntime.Quit(smokeContext)
		}
	}
	previousStartup := startup
	startup = func(ctx context.Context) {
		desktop.SetPicker(bridge, func(kind string) (string, error) {
			switch kind {
			case "workspace":
				return wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{Title: "Choose workspace"})
			case "media":
				return wailsruntime.OpenFileDialog(ctx, wailsruntime.OpenDialogOptions{Title: "Choose audio or video"})
			case "subtitle":
				return wailsruntime.OpenFileDialog(ctx, wailsruntime.OpenDialogOptions{Title: "Choose subtitles", Filters: []wailsruntime.FileFilter{{DisplayName: "Subtitles", Pattern: "*.srt;*.vtt;*.ass;*.ssa;*.cueson;*.json"}}})
			case "output":
				return wailsruntime.SaveFileDialog(ctx, wailsruntime.SaveDialogOptions{Title: "Export subtitles"})
			}
			return "", contracts.Fail("invalid_request")
		})
		if previousStartup != nil {
			previousStartup(ctx)
		}
	}
	if len(os.Args) == 3 && os.Args[1] == "--workspace" {
		w, err := workspace.Open(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		bridge.Workspace = w
	}
	if err := wails.Run(&options.App{Title: "insonic", Width: 1200, Height: 800, MinWidth: 720, MinHeight: 600, StartHidden: desktop.QualificationStartsHidden(smoke), OnStartup: startup, OnDomReady: domReady, OnShutdown: func(context.Context) { desktop.ClosePlaybackHandles(bridge) }, AssetServer: &assetserver.Options{Assets: desktop.Assets, Handler: desktop.PlaybackServer{Bridge: bridge}}, Bind: []any{bridge}}); err != nil {
		fmt.Fprintln(os.Stderr, "Desktop startup failed.")
		return 1
	}
	if smoke {
		if !smokePassed.Load() {
			fmt.Fprintln(os.Stderr, "Native webview qualification failed.")
			return 1
		}
		json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": contracts.Version, "native_webview": "passed", "frontend_bridge_ipc": "passed", "desktop_journeys": "passed", "audio_video_playback": "passed", "model_execution": "not-run"})
	}
	return 0
}
