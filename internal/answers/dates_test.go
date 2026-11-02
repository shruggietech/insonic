// SPDX-License-Identifier: Apache-2.0
package answers_test

import (
	"encoding/json"
	"testing"

	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
)

func TestPersistedApproximateDatesRetainBoundedAndOpenRanges(t *testing.T) {
	lower, _ := library.BoundInstant("2026-03-08T00:30:00Z")
	upper, _ := library.BoundInstant("2026-03-10T03:00:00Z")
	for _, tc := range []struct {
		name                                  string
		date                                  library.Date
		lo, hi, excludedFrom, excludedThrough string
	}{
		{"bounded", library.Date{Precision: "range", State: "bounded", Lower: lower, Upper: upper}, "2026-03-07", "2026-03-09", "2026-03-10", ""},
		{"lower-only", library.Date{Precision: "range", State: "bounded", Lower: lower}, "2026-03-07", "", "", "2026-03-06"},
		{"upper-only", library.Date{Precision: "range", State: "bounded", Upper: upper}, "", "2026-03-09", "2026-03-10", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(library.Dates{Selected: &tc.date})
			if err != nil {
				t.Fatal(err)
			}
			lo, hi, known, err := answers.DateSpan(raw, "America/New_York")
			if err != nil || !known || lo != tc.lo || hi != tc.hi {
				t.Fatal("persisted range lost", lo, hi, known, err, string(raw))
			}
			item := answers.Item{RecordingDate: lo, RecordingDateThrough: hi, DateKnown: known}
			for _, includeUndated := range []bool{false, true} {
				r := answers.Request{Filters: answers.Filters{RecordingDates: &contracts.CalendarFilter{From: "2026-03-08", Through: "2026-03-08", IncludeUndated: includeUndated}}}
				if !answers.Matches(item, r, false) {
					t.Fatal("overlapping approximate date excluded")
				}
				r.Filters.RecordingDates.From = tc.excludedFrom
				r.Filters.RecordingDates.Through = tc.excludedThrough
				if answers.Matches(item, r, false) {
					t.Fatal("nonoverlapping approximate date treated as undated")
				}
			}
		})
	}
}
