// SPDX-License-Identifier: Apache-2.0
package main

import (
	"strings"
	"testing"
)

func TestWorkerProbeHumanSummaryPrecedesDetailsAndRejectsRawReasons(t *testing.T) {
	probe := map[string]any{"mode": "inspect", "stage": "library_import", "reason": "deadline", "elapsed_ms": 60000, "budget_ms": 60000}
	fields := map[string]any{"role": "diarization", "probe": probe}
	var b humanBuffer
	humanSummary(&b, fields, 0)
	if !strings.Contains(b.String(), "Worker diarization probe: deadline at library_import; native 60000 ms of 60000 ms.") {
		t.Fatal(b.String())
	}
	probe["reason"] = "secret-token private/path"
	b = humanBuffer{}
	humanSummary(&b, fields, 0)
	if strings.Contains(b.String(), "secret-token") {
		t.Fatal("unvalidated diagnostic reason escaped")
	}
}
