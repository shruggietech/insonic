// SPDX-License-Identifier: Apache-2.0
package library

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/timezone"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
)

// Exact Go 1.27.1 IANA data and CLDR release-48 world mappings are bundled, so
// historical inputs have the same interpretation on every supported platform.
var RulesVersion = timezone.RulesVersion()

const MappingVersion = timezone.MappingVersion

type Date struct {
	ID             string           `json:"id"`
	Literal        string           `json:"literal"`
	Basis          string           `json:"basis"`
	Precision      string           `json:"precision"`
	Zone           string           `json:"zone"`
	ZoneSource     string           `json:"zone_source"`
	RulesVersion   string           `json:"rules_version"`
	MappingVersion string           `json:"mapping_version"`
	OffsetSeconds  *int64           `json:"offset_seconds"`
	Resolved       *catalog.Instant `json:"resolved"`
	Lower          *catalog.Instant `json:"earliest,omitempty"`
	Upper          *catalog.Instant `json:"latest,omitempty"`
	State          string           `json:"interpretation_state"`
	Assumptions    []string         `json:"assumptions"`
	Diagnostic     string           `json:"diagnostic,omitempty"`
}
type Dates struct {
	Observations []Date `json:"observations"`
	Selected     *Date  `json:"selected"`
	Policy       string `json:"policy"`
	Revision     string `json:"policy_revision"`
	Reason       string `json:"reason"`
	Conflict     bool   `json:"conflict"`
}

// CalendarLocation shares pinned rules with query validation and metadata dates.
func CalendarLocation(name string) (*time.Location, error) { return timezone.Location(name) }
func loadZone(name string) (*time.Location, error)         { return timezone.Location(name) }
func LocalZone() (string, error)                           { return timezone.LocalName() }

type zoneError struct{}

func (*zoneError) Error() string { return "unavailable timezone" }
func ResolveDate(literal, precision, zone, fold, gap, basis string) Date {
	d := Date{Literal: literal, Basis: basis, Precision: precision, Zone: zone, ZoneSource: "import-policy", RulesVersion: RulesVersion, MappingVersion: MappingVersion, State: "invalid", Assumptions: []string{}}
	if precision == "day" {
		if _, e := time.Parse("2006-01-02", literal); e == nil {
			d.State = "date-only"
		}
		return d
	}
	if precision != "instant" || (fold != "" && fold != "earlier" && fold != "later") || (gap != "" && gap != "shift-forward") {
		return d
	}
	explicit, e := time.Parse(time.RFC3339Nano, literal)
	hasOffset := e == nil
	wall := explicit
	if !hasOffset {
		wall, e = time.Parse("2006-01-02T15:04:05.999999999", literal)
		if e != nil {
			d.Diagnostic = "invalid recording timestamp"
			return d
		}
	}
	if zone == "" && hasOffset {
		_, offset := explicit.Zone()
		zone = explicit.Format("-07:00")
		if offset == 0 {
			zone = "UTC"
		}
		d.ZoneSource = "entered-offset"
	}
	if zone == "" || strings.EqualFold(zone, "local") {
		zone, e = LocalZone()
		d.ZoneSource = "captured-local"
		if e != nil {
			d.State = "unknown"
			d.Diagnostic = "local timezone mapping unavailable"
			return d
		}
	}
	d.Zone = zone
	location, e := loadZone(zone)
	if e != nil {
		d.State = "unknown"
		d.Diagnostic = "timezone rules unavailable"
		return d
	}
	if hasOffset {
		_, enteredOffset := explicit.Zone()
		_, actualOffset := explicit.In(location).Zone()
		if enteredOffset != actualOffset {
			d.Diagnostic = "entered offset disagrees with timezone"
			return d
		}
		return resolvedDate(d, explicit, int64(actualOffset))
	}
	wall = time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), time.UTC)
	offsets := map[int]bool{}
	for hours := -72; hours <= 72; hours++ {
		_, offset := wall.Add(time.Duration(hours) * time.Hour).In(location).Zone()
		offsets[offset] = true
	}
	candidates := []time.Time{}
	var shifted *time.Time
	var shiftSize time.Duration
	for offset := range offsets {
		candidate := wall.Add(-time.Duration(offset) * time.Second)
		local := candidate.In(location)
		reflected := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), local.Second(), local.Nanosecond(), time.UTC)
		delta := reflected.Sub(wall)
		if delta == 0 {
			candidates = append(candidates, candidate)
		} else if delta > 0 && (shifted == nil || delta < shiftSize) {
			value := candidate
			shifted = &value
			shiftSize = delta
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Before(candidates[j]) })
	if len(candidates) == 0 {
		d.State = "nonexistent"
		d.Diagnostic = "recording time falls in a timezone gap"
		if gap != "shift-forward" || shifted == nil {
			return d
		}
		d.Assumptions = append(d.Assumptions, "dst-gap-shift-forward")
		_, offset := shifted.In(location).Zone()
		return resolvedDate(d, *shifted, int64(offset))
	}
	if len(candidates) > 1 && fold == "" {
		d.State = "ambiguous"
		d.Diagnostic = "recording time repeats; select earlier/later or an offset"
		return d
	}
	chosen := candidates[0]
	if fold == "later" {
		chosen = candidates[len(candidates)-1]
	}
	if len(candidates) > 1 {
		d.Assumptions = append(d.Assumptions, "dst-fold-"+fold)
	}
	d.Assumptions = append(d.Assumptions, "import-timezone")
	_, offset := chosen.In(location).Zone()
	return resolvedDate(d, chosen, int64(offset))
}
func resolvedDate(d Date, t time.Time, offset int64) Date {
	if t.Before(time.Unix(0, math.MinInt64)) || t.After(time.Unix(0, math.MaxInt64)) {
		d.State = "invalid"
		d.Diagnostic = "recording instant exceeds timestamp range"
		return d
	}
	d.State = "resolved"
	d.OffsetSeconds = &offset
	d.Resolved = &catalog.Instant{ISO: t.UTC().Format(time.RFC3339Nano), UnixNS: t.UnixNano()}
	d.Diagnostic = ""
	return d
}
func SelectDates(observations []Date, policy string) Dates {
	if policy == "" {
		policy = "owner-first"
	}
	state := Dates{Observations: observations, Policy: policy, Revision: "1", Reason: "unknown"}
	rank := func(d Date) int {
		switch d.Basis {
		case "owner":
			if policy == "embedded-first" {
				return 3
			}
			return 0
		case "embedded-own-zone":
			return 1
		case "embedded-import-zone":
			return 2
		case "filesystem-modified":
			if policy == "filesystem-fallback" {
				return 4
			}
		}
		return 9
	}
	chosenRank := 10
	for i := range observations {
		d := observations[i]
		if d.State != "resolved" && d.State != "date-only" && !validBoundedDate(d) {
			continue
		}
		r := rank(d)
		if r < chosenRank {
			value := d
			state.Selected = &value
			state.Reason = d.Basis
			chosenRank = r
		}
	}
	if chosenRank == 9 {
		state.Selected = nil
		state.Reason = "unknown"
	}
	if state.Selected != nil {
		for _, d := range observations {
			if rank(d) < 9 && (d.State == "resolved" || d.State == "date-only" || validBoundedDate(d)) && !sameDateValue(d, *state.Selected) {
				state.Conflict = true
			}
		}
	}
	return state
}

// Value equivalence drives conflicts; separate observations still retain each
// literal, timezone and interpretation choice that produced the value.
func sameDateValue(a, b Date) bool {
	if a.Precision != b.Precision || a.State != b.State {
		return false
	}
	switch a.State {
	case "resolved":
		return a.Resolved != nil && b.Resolved != nil && a.Resolved.UnixNS == b.Resolved.UnixNS
	case "date-only":
		return a.Literal == b.Literal
	case "bounded":
		return validBoundedDate(a) && validBoundedDate(b) && sameBound(a.Lower, b.Lower) && sameBound(a.Upper, b.Upper)
	}
	return false
}
func sameBound(a, b *catalog.Instant) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.UnixNS == b.UnixNS
}
func validBound(i *catalog.Instant) bool {
	if i == nil {
		return true
	}
	t, e := time.Parse(time.RFC3339Nano, i.ISO)
	return e == nil && !t.Before(time.Unix(0, math.MinInt64)) && !t.After(time.Unix(0, math.MaxInt64)) && t.UnixNano() == i.UnixNS
}
func BoundInstant(literal string) (*catalog.Instant, error) {
	t, e := time.Parse(time.RFC3339Nano, literal)
	if e != nil || t.Before(time.Unix(0, math.MinInt64)) || t.After(time.Unix(0, math.MaxInt64)) {
		return nil, &zoneError{}
	}
	return &catalog.Instant{ISO: t.UTC().Format(time.RFC3339Nano), UnixNS: t.UnixNano()}, nil
}
func validBoundedDate(d Date) bool {
	return d.State == "bounded" && d.Precision == "range" && d.Resolved == nil && (d.Lower != nil || d.Upper != nil) && validBound(d.Lower) && validBound(d.Upper) && (d.Lower == nil || d.Upper == nil || d.Lower.UnixNS <= d.Upper.UnixNS)
}
func ownerDate(options Options) Date {
	if options.OriginatedEarliest != nil || options.OriginatedLatest != nil {
		d := Date{Basis: "owner", Precision: "range", State: "bounded", Lower: options.OriginatedEarliest, Upper: options.OriginatedLatest, Zone: options.Timezone, RulesVersion: RulesVersion, MappingVersion: MappingVersion, Assumptions: []string{"approximate-bounds"}}
		if !validBoundedDate(d) {
			d.State = "invalid"
		}
		return d
	}
	literal := options.OriginatedAt
	precision := "instant"
	if options.OriginatedOn != "" {
		literal = options.OriginatedOn
		precision = "day"
	}
	zone := options.Timezone
	if options.TimezoneImplicit {
		if _, e := time.Parse(time.RFC3339Nano, literal); e == nil {
			zone = ""
		}
	}
	d := ResolveDate(literal, precision, zone, options.DSTFold, options.DSTGap, "owner")
	if options.TimezoneImplicit && d.ZoneSource == "import-policy" {
		d.ZoneSource = "captured-local"
	}
	return d
}
func decodeDates(data json.RawMessage) Dates {
	var state Dates
	json.Unmarshal(data, &state)
	return state
}
