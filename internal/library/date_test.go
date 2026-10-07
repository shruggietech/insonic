// SPDX-License-Identifier: Apache-2.0
package library

import (
	"testing"
)

func TestHistoricalDatesAndDST(t *testing.T) {
	cases := []struct{ literal, zone, fold, gap, state, iso string }{
		{"2026-01-05T14:30:00", "America/New_York", "", "", "resolved", "2026-01-05T19:30:00Z"},
		{"2026-07-05T14:30:00", "America/New_York", "", "", "resolved", "2026-07-05T18:30:00Z"},
		{"2026-11-01T01:30:00", "America/New_York", "", "", "ambiguous", ""},
		{"2026-11-01T01:30:00", "America/New_York", "earlier", "", "resolved", "2026-11-01T05:30:00Z"},
		{"2026-11-01T01:30:00", "America/New_York", "later", "", "resolved", "2026-11-01T06:30:00Z"},
		{"2026-03-08T02:30:00", "America/New_York", "", "", "nonexistent", ""},
		{"2026-03-08T02:30:00", "America/New_York", "", "shift-forward", "resolved", "2026-03-08T07:30:00Z"},
		{"2026-10-04T02:15:00", "Australia/Lord_Howe", "", "shift-forward", "resolved", "2026-10-03T15:45:00Z"},
		{"2011-12-30T12:00:00", "Pacific/Apia", "", "shift-forward", "resolved", "2011-12-30T22:00:00Z"},
		{"2026-01-05T14:30:00+00:00", "America/New_York", "", "", "invalid", ""},
	}
	for _, c := range cases {
		t.Run(c.literal+c.fold+c.gap+c.zone, func(t *testing.T) {
			d := ResolveDate(c.literal, "instant", c.zone, c.fold, c.gap, "owner")
			if d.State != c.state {
				t.Fatalf("state %+v", d)
			}
			if c.iso != "" && (d.Resolved == nil || d.Resolved.ISO != c.iso) {
				t.Fatalf("instant %+v", d)
			}
			if c.iso == "" && d.Resolved != nil {
				t.Fatal("invented instant")
			}
		})
	}
	d := ResolveDate("2026-10-07", "day", "America/New_York", "", "", "owner")
	if d.State != "date-only" || d.Resolved != nil || d.Literal != "2026-10-07" {
		t.Fatalf("date-only %+v", d)
	}
	d = ResolveDate("2026-02-30", "day", "UTC", "", "", "owner")
	if d.State != "invalid" {
		t.Fatal("invalid calendar admitted")
	}
	d = ResolveDate("2026-10-07T14:30:00", "instant", "+05:30", "", "", "owner")
	if d.Resolved == nil || d.Resolved.ISO != "2026-10-07T09:00:00Z" {
		t.Fatalf("offset %+v", d)
	}
}

func TestDatePrecedencePreservesOwnerAndConflicts(t *testing.T) {
	owner := ResolveDate("2026-10-07", "day", "UTC", "", "", "owner")
	embedded := ResolveDate("2026-10-06T12:00:00Z", "instant", "UTC", "", "", "embedded-own-zone")
	state := SelectDates([]Date{embedded, owner}, "owner-first")
	if len(state.Observations) != 2 || state.Selected == nil || state.Selected.Literal != owner.Literal || !state.Conflict {
		t.Fatalf("precedence %+v", state)
	}
	if SelectDates(nil, "owner-first").Selected != nil {
		t.Fatal("unknown invented")
	}
}
func TestApproximateBoundsPreservedWithoutAnInstant(t *testing.T) {
	lower, _ := BoundInstant("2026-10-01T00:00:00Z")
	upper, _ := BoundInstant("2026-10-07T23:59:59.123456789Z")
	state := SelectDates([]Date{ownerDate(Options{OriginatedEarliest: lower, OriginatedLatest: upper})}, "owner-first")
	if state.Selected == nil || state.Selected.Resolved != nil || state.Selected.Precision != "range" || state.Selected.Upper.UnixNS != upper.UnixNS {
		t.Fatal("bounded precision lost")
	}
	if validOptions(Options{OriginatedEarliest: upper, OriginatedLatest: lower}) {
		t.Fatal("reversed bounds accepted")
	}
	damaged := *lower
	damaged.UnixNS++
	if validOptions(Options{OriginatedEarliest: &damaged}) {
		t.Fatal("mismatched instant admitted")
	}
}

func TestDateConflictComparesRecordingValuesAndRetainsProvenance(t *testing.T) {
	earlier := ResolveDate("2026-11-01T01:30:00", "instant", "America/New_York", "earlier", "", "owner")
	later := ResolveDate("2026-11-01T01:30:00", "instant", "America/New_York", "later", "", "embedded-import-zone")
	otherZone := ResolveDate("2026-11-01T01:30:00", "instant", "America/Chicago", "earlier", "", "embedded-import-zone")
	for _, other := range []Date{later, otherZone} {
		state := SelectDates([]Date{earlier, other}, "owner-first")
		if !state.Conflict || len(state.Observations) != 2 {
			t.Fatal("distinct instants with equal literals lost conflict/provenance")
		}
	}
	utc := ResolveDate("2026-11-01T05:30:00Z", "instant", "", "", "", "embedded-own-zone")
	state := SelectDates([]Date{earlier, utc}, "owner-first")
	if state.Conflict || len(state.Observations) != 2 || state.Observations[0].Literal == state.Observations[1].Literal {
		t.Fatal("same exact instant falsely conflicts or provenance collapsed")
	}
}

func TestOffsetRequiresASCIIDigits(t *testing.T) {
	for _, zone := range []string{"+-1:00", "+01:-1", "++1:00", "+24:00", "+01:60"} {
		if _, ok := parseOffset(zone); ok {
			t.Fatal("malformed offset accepted")
		}
		if d := ResolveDate("2026-10-07T12:00:00", "instant", zone, "", "", "owner"); d.Resolved != nil {
			t.Fatal("malformed zone manufactured instant")
		}
	}
	for _, zone := range []string{"+14:00", "-05:30", "UTC", "Z"} {
		if d := ResolveDate("2026-10-07T12:00:00", "instant", zone, "", "", "owner"); d.Resolved == nil {
			t.Fatal("standard zone rejected")
		}
	}
}
