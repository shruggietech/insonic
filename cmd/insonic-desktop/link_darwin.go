//go:build desktop

package main

// Wails uses UTType in its native dialogs; link its defining system framework.
// #cgo LDFLAGS: -framework UniformTypeIdentifiers
import "C"
