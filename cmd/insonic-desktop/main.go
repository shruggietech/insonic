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
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"sync/atomic"
	"time"
)

func main() {
	code := run()
	if err := nativeCOMCleanup(); err != nil {
		fmt.Fprintln(os.Stderr, "Desktop COM cleanup failed:", err)
		if code == 0 {
			code = 1
		}
	}
	if len(os.Args) == 2 && os.Args[1] == "--webview-qualification" {
		fmt.Fprintln(os.Stderr, "Desktop qualification cleanup: process exit")
	}
	os.Exit(code)
}

func run() (exitCode int) {
	if len(os.Args) == 2 && os.Args[1] == "--native-wait-query" {
		return qualifyNativeWait()
	}
	if len(os.Args) == 2 && os.Args[1] == "--exit-qualification" {
		return qualifyNativeExit()
	}
	if len(os.Args) == 3 && os.Args[1] == "--exit-qualification-child" {
		return qualifyNativeExitChild(os.Args[2])
	}
	if len(os.Args) == 2 && os.Args[1] == "--help-qualification" {
		return qualifyHelpOnly()
	}
	smoke := len(os.Args) == 2 && os.Args[1] == "--webview-qualification"
	profile := ""
	if smoke {
		var err error
		profile, err = qualificationWebviewProfile()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	var smokePassed atomic.Bool
	if smoke {
		// Register first: success is emitted only after every later cleanup has
		// returned, including runtime cancellation and fixture removal.
		defer func() {
			if exitCode == 0 && smokePassed.Load() {
				json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": contracts.Version, "native_webview": "passed", "frontend_bridge_ipc": "passed", "desktop_journeys": "passed", "ui_model_references": "passed", "audio_video_playback": "passed", "model_execution": "not-run", "ui_compact_density": "passed"})
			}
		}()
	}
	if err := workspace.RequireSupportedHost(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
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
	var startup func(context.Context)
	var domReady func(context.Context)
	var smokeContext context.Context
	if smoke {
		directory, err := os.MkdirTemp("", "insonic-webview-*")
		if err != nil {
			return 1
		}
		defer func() {
			fmt.Fprintln(os.Stderr, "Desktop qualification cleanup: fixtures")
			if err := os.RemoveAll(directory); err != nil {
				fmt.Fprintln(os.Stderr, "Desktop qualification fixture cleanup failed.")
				exitCode = 1
			}
			fmt.Fprintln(os.Stderr, "Desktop qualification cleanup: fixtures complete")
		}()
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
			done <- local.Serve(ctx, w, local.Options{Idle: 90 * time.Second, Ready: ready, AssistanceFixture: func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
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
		defer func() {
			fmt.Fprintln(os.Stderr, "Desktop qualification cleanup: runtime")
			cancel()
			if err := <-done; err != nil {
				fmt.Fprintln(os.Stderr, "Desktop qualification runtime cleanup failed.")
				exitCode = 1
			}
			fmt.Fprintln(os.Stderr, "Desktop qualification cleanup: runtime complete")
		}()
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
			for _, name := range []string{"ui_library_import", "ui_metadata_date", "ui_current_assembly", "ui_audio_playback", "ui_video_source_audio_playback", "ui_cue_seek", "ui_terms", "ui_speakers", "ui_pipelines", "ui_jobs", "ui_settings", "ui_model_references", "ui_keyboard_help", "ui_offline_help_activation", "ui_explore_calendar", "ui_explore_query", "ui_explore_graph", "ui_query_assistance", "ui_compact_density"} {
				passed = passed && result[name] == "passed"
			}
			smokePassed.Store(passed)
			wailsruntime.Quit(smokeContext)
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
	if err := runDesktopApplicationWithProfile(bridge, startup, domReady, desktop.QualificationStartsHidden(smoke), profile); err != nil {
		fmt.Fprintln(os.Stderr, "Desktop lifecycle failed.")
		return 1
	}
	if smoke {
		if !smokePassed.Load() {
			fmt.Fprintln(os.Stderr, "Native webview qualification failed.")
			return 1
		}
	}
	return 0
}
