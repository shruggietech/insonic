//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"github.com/wailsapp/go-webview2/pkg/edge"
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
