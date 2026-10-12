// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
)

var startupReported bool

func emitStartup(err error) {
	startupReported = true
	report := contracts.StartupReport{Kind: "runtime-startup", Version: contracts.Version, StartupID: operatorContext.StartupID, WorkspaceID: responseContext.WorkspaceID, State: "ready", Stage: "ready"}
	if err != nil {
		e := contracts.SafeError(err)
		if e.Reason == "" {
			e = contracts.Condition("unavailable", "startup_exit")
		}
		report.State = "failed"
		report.Stage = "startup"
		report.Code = e.Code
		report.Reason = e.Reason
	}
	_ = json.NewEncoder(os.Stdout).Encode(report)
}
