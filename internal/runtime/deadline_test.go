// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"testing"
	"time"
)

func TestOperationTimeoutPreservesCallerDefaultsAndLongFamilies(t *testing.T) {
	for _, ordinary := range []time.Duration{5 * time.Second, 10 * time.Second} {
		for _, operation := range []string{"media.import", "media.refresh", "models.acquire", "models.materialize", "artifacts.publish"} {
			if timeout := OperationTimeout(operation, ordinary); timeout != 10*time.Minute {
				t.Fatal(operation, timeout)
			}
		}
		for _, operation := range []string{"workspace.show", "credentials.status", "work.results", "jobs.cancel", "medial.import", "models", ""} {
			if timeout := OperationTimeout(operation, ordinary); timeout != ordinary {
				t.Fatal("ordinary deadline changed", operation, timeout)
			}
		}
	}
}
