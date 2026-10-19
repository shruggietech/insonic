//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"github.com/wailsapp/go-webview2/pkg/edge"
)

func nativeWebviewCloseError() error {
	if err := edge.WindowCloseError(); err != nil {
		return err
	}
	if edge.ClosedControllerCount() != 1 {
		return fmt.Errorf("owned WebView controller close was not observed")
	}
	return nil
}
