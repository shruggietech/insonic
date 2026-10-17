//go:build ignore

// SPDX-License-Identifier: Apache-2.0
// Disposable-host cancellation barrier. No product behavior or native result is
// replaced: the driver must click the real MSI Cancel and confirmation controls.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

func main() { os.Exit(run()) }
func run() int {
	if len(os.Args) != 2 {
		return 1
	}
	stage := filepath.Clean(os.Args[1])
	if !filepath.IsAbs(stage) || !strings.EqualFold(filepath.Dir(stage), os.Getenv("ProgramData")) || !strings.HasPrefix(filepath.Base(stage), "insonic-ci-account-") {
		return 2
	}
	data, err := os.ReadFile(filepath.Join(stage, "owned-ci-account.json"))
	if err != nil {
		return 3
	}
	var marker struct {
		SID string `json:"sid_sha256"`
	}
	if json.Unmarshal(data, &marker) != nil {
		return 4
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil || windows.GetCurrentProcessToken().IsElevated() {
		return 5
	}
	sum := sha256.Sum256([]byte(user.User.Sid.String()))
	if hex.EncodeToString(sum[:]) != marker.SID {
		return 6
	}
	if err := os.WriteFile(filepath.Join(stage, "native/setup-cancel-ready.json"), []byte("{\"barrier\":\"after-old-removal\",\"feature_qualification\":false}\n"), 0600); err != nil {
		return 7
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(stage, "native/setup-cancel-release")); err == nil {
			return 0
		}
		time.Sleep(20 * time.Millisecond)
	}
	return 8
}
