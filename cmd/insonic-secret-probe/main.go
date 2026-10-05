// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"runtime"
)

func main() {
	state := "unavailable"
	if available() {
		state = "available"
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": contracts.Version, "platform": runtime.GOOS, "native_secret_service": state, "credential_values": "not-returned"})
}
