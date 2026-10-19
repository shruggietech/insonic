//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Production and native qualification use one menu, startup and shutdown path.
func applicationOptions(bridge *desktop.Bridge, startup, domReady func(context.Context), hidden bool, profile string) *options.App {
	quit := desktop.NewQuitLifecycle(wailsruntime.Quit)
	requestQuit := quit.Request
	if profile != "" {
		requestQuit = func() { fmt.Fprintln(os.Stderr, "native-exit-stage=menu-request"); quit.Request() }
	}
	middleware := desktop.HelpAssetPolicy
	if profile != "" {
		middleware = observeQualificationAssets(middleware)
	}
	app := &options.App{
		Title: "insonic", Width: 1200, Height: 800, MinWidth: 720, MinHeight: 600, StartHidden: hidden,
		Menu: desktop.ApplicationMenu(runtime.GOOS, requestQuit),
		OnStartup: func(ctx context.Context) {
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
			if startup != nil {
				startup(ctx)
			}
			quit.Ready(ctx)
		},
		OnDomReady:  domReady,
		OnShutdown:  func(context.Context) { desktop.ClosePlaybackHandles(bridge) },
		AssetServer: &assetserver.Options{Assets: desktop.Assets, Middleware: middleware, Handler: desktop.PlaybackServer{Bridge: bridge}},
		Bind:        []any{bridge},
	}
	if profile != "" {
		app.Windows = &windows.Options{WebviewUserDataPath: profile}
	}
	return app
}

func runDesktopApplication(bridge *desktop.Bridge, startup, domReady func(context.Context), hidden bool) error {
	return runDesktopApplicationWithProfile(bridge, startup, domReady, hidden, "")
}

// Disposable native qualification supplies its owned profile explicitly. Normal
// application launches retain the persistent Wails default.
func runDesktopApplicationWithProfile(bridge *desktop.Bridge, startup, domReady func(context.Context), hidden bool, profile string) error {
	stopNativeMenu := configureNativeMenu(bridge, profile != "")
	defer stopNativeMenu()
	closePlayback, err := desktop.StartPlaybackTransport(bridge)
	if err != nil {
		return err
	}
	defer func() {
		if hidden {
			fmt.Fprintln(os.Stderr, "Desktop qualification cleanup: playback")
		}
		closePlayback()
		if hidden {
			fmt.Fprintln(os.Stderr, "Desktop qualification cleanup: playback complete")
		}
	}()
	if err := wails.Run(applicationOptions(bridge, startup, domReady, hidden, profile)); err != nil {
		return err
	}
	return nativeWebviewCloseError()
}
