// SPDX-License-Identifier: Apache-2.0
package pipeline

import (
	"github.com/shruggietech/insonic/internal/processing"
	"testing"
)

func TestRepairPolicyElectionCopiesWholePolicies(t *testing.T) {
	d := localDefinition()
	d.Repair = &processing.RepairPolicy{MaxAttempts: 3, MaxWindows: 9}
	d.Coverage = &processing.CoverageOptions{MinimumGapUS: 900000}
	p := processing.RepairPolicy{MaxWindows: 2}
	c := processing.CoverageOptions{GapToleranceUS: 1000}
	elected, e := Elect("local", d, Override{Repair: &p, Coverage: &c})
	if e != nil {
		t.Fatal(e)
	}
	p.MaxWindows = 99
	c.GapToleranceUS = 88
	if elected.Repair.MaxAttempts != 1 || elected.Repair.MaxWindows != 2 || elected.Coverage.MinimumGapUS != 250000 || elected.Coverage.GapToleranceUS != 1000 {
		t.Fatal("partial merge or mutable election", elected)
	}
	for _, p := range []processing.RepairPolicy{{MaxAttempts: 4}, {MaxWindows: 16385}, {MaxDurationUS: 604800000001}} {
		d.Repair = &p
		if Validate("local", d) == nil {
			t.Fatal("unbounded repair policy")
		}
	}
	d.Repair = nil
	d.Coverage = &processing.CoverageOptions{After: 1}
	if Validate("local", d) == nil {
		t.Fatal("saved comparison skipped regions")
	}
}
