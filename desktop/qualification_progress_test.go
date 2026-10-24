// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/qualificationtiming"
)

func TestHelpProgressUsesBoundedNativeMonotonicObservations(t *testing.T) {
	var out bytes.Buffer
	origin := time.Unix(100, 0)
	now := origin
	p := newQualificationProgress("help", &out, func() time.Time { return now })
	if p.record("private-path-or-token") {
		t.Fatal("unbounded stage accepted")
	}
	now = now.Add(36 * time.Second)
	if !p.record("help-open-returned") {
		t.Fatal("fixed stage rejected")
	}
	p.terminal(true)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || strings.Contains(out.String(), "private-path") {
		t.Fatal(out.String())
	}
	var stage, terminal map[string]any
	if json.Unmarshal([]byte(lines[0]), &stage) != nil || json.Unmarshal([]byte(lines[1]), &terminal) != nil {
		t.Fatal(out.String())
	}
	if stage["elapsed_ns"] != float64(36*time.Second) || stage["observation"] != "native-receipt" || terminal["classification"] != "passed-after-former-bound" || terminal["capture_bound_seconds"] != float64(qualificationtiming.Seconds("help_seconds")) || terminal["effective_bound_seconds"] != float64(qualificationtiming.Seconds("help_watchdog_seconds")) {
		t.Fatal(out.String())
	}
	p.terminal(false)
	if len(strings.Split(strings.TrimSpace(out.String()), "\n")) != 2 {
		t.Fatal("terminal emitted twice")
	}
}

func TestHelpProgressFailureAndFormerBoundRemainDistinct(t *testing.T) {
	for _, sample := range []struct {
		elapsed        time.Duration
		passed         bool
		classification string
	}{
		{35 * time.Second, true, "passed-within-former-bound"},
		{36 * time.Second, true, "passed-after-former-bound"},
		{180 * time.Second, false, "failed"},
	} {
		var out bytes.Buffer
		now := time.Unix(0, 0)
		p := newQualificationProgress("help", &out, func() time.Time { return now })
		now = now.Add(sample.elapsed)
		p.terminal(sample.passed)
		if !strings.Contains(out.String(), `"classification":"`+sample.classification+`"`) {
			t.Fatal(out.String())
		}
	}
}

func TestOrdinaryBridgeCannotActivateQualificationProgress(t *testing.T) {
	b := &Bridge{}
	if b.QualificationStep("help-open-returned") {
		t.Fatal("ordinary desktop enabled fixture progress")
	}
	RecordQualificationTerminal(b, true)
	if b.qualificationProgress != nil {
		t.Fatal("ordinary terminal call created an observer")
	}
}

func TestQualificationProgressNeverRegressesOrGrowsWithoutBound(t *testing.T) {
	var out bytes.Buffer
	now := time.Unix(100, 0)
	p := newQualificationProgress("help", &out, func() time.Time { return now })
	now = now.Add(time.Second)
	if !p.record("help-open-returned") {
		t.Fatal("fixed stage refused")
	}
	now = now.Add(-time.Second)
	if p.record("help-closed") {
		t.Fatal("regressing observation accepted")
	}
	now = now.Add(2 * time.Second)
	for range 200 {
		p.record("help-open-returned")
	}
	p.terminal(false)
	if got := len(strings.Split(strings.TrimSpace(out.String()), "\n")); got != 129 {
		t.Fatal("unbounded or missing terminal evidence", got)
	}
}

func TestFrontendQualificationClockIsSeparateMonotonicAndBounded(t *testing.T) {
	var out bytes.Buffer
	now := time.Unix(100, 0)
	p := newQualificationProgress("help", &out, func() time.Time { return now })
	for _, elapsed := range []float64{-1, math.NaN(), math.Inf(1), 86400001} {
		if p.recordFrontend("fixtures", elapsed) {
			t.Fatal("invalid frontend clock accepted", elapsed)
		}
	}
	now = now.Add(10 * time.Second)
	if !p.recordFrontend("fixtures", 2) {
		t.Fatal("frontend observation rejected")
	}
	if p.recordFrontend("help-open-returned", 1) {
		t.Fatal("frontend clock regressed")
	}
	now = now.Add(time.Second)
	if !p.recordFrontend("help-open-returned", 1000) {
		t.Fatal("later frontend observation rejected")
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatal(out.String())
	}
	var first map[string]any
	if json.Unmarshal([]byte(lines[0]), &first) != nil || first["elapsed_ns"] != float64(10*time.Second) || first["frontend_elapsed_ms"] != float64(2) || first["frontend_origin"] != "first-reported-stage" {
		t.Fatal("clock domains conflated", out.String())
	}
	b := &Bridge{}
	if b.QualificationProgress("fixtures", 1) || b.qualificationProgress != nil {
		t.Fatal("ordinary bridge activated frontend diagnostics")
	}
}
