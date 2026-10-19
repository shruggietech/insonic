//go:build desktop && linux

// SPDX-License-Identifier: Apache-2.0
package main

func nativeWebviewCloseError() error { return nil }

func nativeCOMCleanup() error { return nil }

func nativeWebviewBootstrapSnapshot() any { return nil }

func nativeWebviewBrowserSnapshot() any { return nil }

func logNativeBrowserRuntimeVersion() {}

func nativeStartupWaitSnapshot() any { return nil }

func qualifyNativeWait() int { return 1 }
