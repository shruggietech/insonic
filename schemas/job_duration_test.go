// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestJobDurationContractBoundaries(t *testing.T) {
	for _, duration := range []int{0, 1, 60000, 60001, 600000, 600001} {
		raw, _ := json.Marshal(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: "jobs.start", DurationMS: duration})
		err := ValidateRequest(raw)
		valid := duration >= 1 && duration <= 600000
		if (err == nil) != valid {
			t.Errorf("duration %d accepted=%v want=%v", duration, err == nil, valid)
		}
	}
}
