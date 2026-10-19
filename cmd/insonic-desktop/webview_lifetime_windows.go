//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"github.com/wailsapp/go-webview2/pkg/edge"
	"github.com/wailsapp/go-webview2/webviewloader"
	"os"
)

func nativeWebviewCloseError() error {
	cleanup := nativeCOMCleanup()
	if err := edge.WindowCloseError(); err != nil {
		return err
	}
	if cleanup != nil {
		return cleanup
	}
	if edge.ClosedControllerCount() != 1 {
		return fmt.Errorf("owned WebView controller close was not observed")
	}
	return nil
}

func nativeCOMCleanup() error { return edge.FinalizeCOMApartment() }

func nativeWebviewBootstrapSnapshot() any { return edge.OwnedWindowBootstrapSnapshots() }

func logNativeBrowserRuntimeVersion() {
	version, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString("")
	if err != nil {
		fmt.Fprintln(os.Stderr, "native-browser-runtime=unavailable")
		return
	}
	fmt.Fprintf(os.Stderr, "native-browser-runtime=%q\n", version[:min(len(version), 64)])
}
