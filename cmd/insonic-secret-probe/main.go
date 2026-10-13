// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"runtime"
)

func main() {
	if workspace.RequireSupportedHost() != nil {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"error_code": "unsupported_platform", "native_secret_service": "unavailable"})
		os.Exit(1)
	}
	state := "unavailable"
	if available() {
		state = "available"
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": contracts.Version, "platform": runtime.GOOS, "native_secret_service": state, "credential_values": "not-returned"})
}
