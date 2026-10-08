// SPDX-License-Identifier: Apache-2.0
package app

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func graphLoops() int {
	b := make([]byte, 1<<20)
	return strings.Count(string(b[:runtime.Stack(b, true)]), "(*App).maintainGraph(")
}
func TestRecoveryKeepsSingleGraphMaintenanceLoop(t *testing.T) {
	before := graphLoops()
	a := configuredApp(t)
	// Three actual recovery ticks must leave exactly the one initialization loop.
	time.Sleep(3200 * time.Millisecond)
	if got := graphLoops(); got != before+1 {
		t.Fatalf("graph loops: %d, expected %d", got, before+1)
	}
	a.Close()
	if got := graphLoops(); got != before {
		t.Fatalf("shutdown left graph loops: %d", got)
	}
}
