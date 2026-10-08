// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"strconv"
	"strings"
	"time"
)

// Exact catalog timestamps travel as decimal strings to JavaScript clients.
func wireDates(raw json.RawMessage) any {
	var v any
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	if d.Decode(&v) != nil {
		return nil
	}
	return exactWire(v)
}
func exactWire(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, a := range x {
			if k == "unix_ns" {
				if n, ok := a.(json.Number); ok {
					x[k] = string(n)
					continue
				}
			}
			x[k] = exactWire(a)
		}
	case []any:
		for i, a := range x {
			x[i] = exactWire(a)
		}
	}
	return v
}
func dateSpan(raw json.RawMessage, zone string) (string, string, bool, error) {
	var ds library.Dates
	if json.Unmarshal(raw, &ds) != nil || ds.Selected == nil {
		return "", "", false, nil
	}
	loc, e := library.CalendarLocation(zone)
	if e != nil {
		return "", "", false, contracts.Fail("invalid_request")
	}
	d := ds.Selected
	if d.Precision == "day" && len(d.Literal) >= 10 {
		day := d.Literal[:10]
		if _, e := time.Parse("2006-01-02", day); e == nil {
			return day, day, true, nil
		}
	}
	day := func(iso string) string {
		t, e := time.Parse(time.RFC3339Nano, iso)
		if e != nil {
			return ""
		}
		return t.In(loc).Format("2006-01-02")
	}
	if d.Resolved != nil {
		v := day(d.Resolved.ISO)
		return v, v, v != "", nil
	}
	lo, hi := "", ""
	if d.Lower != nil {
		lo = day(d.Lower.ISO)
	}
	if d.Upper != nil {
		hi = day(d.Upper.ISO)
	}
	return lo, hi, lo != "" || hi != "", nil
}
func Calendar(c Corpus, f contracts.CalendarFilter, p *contracts.QueryPagination) (Result, error) {
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "media-list", Filters: contracts.QueryFilter{RecordingDates: &f}, OrderBy: []contracts.QueryOrder{{Field: "recording_date", Direction: "asc"}}, Pagination: p}}
	return Query(c, c.Refs, q)
}
func applyDates(c *Corpus, f *contracts.CalendarFilter) error {
	zone := "UTC"
	if f != nil && f.Timezone != "" {
		zone = f.Timezone
	}
	byMedia := map[string][]string{}
	for id, row := range c.Rows {
		byMedia[row.MediaID] = append(byMedia[row.MediaID], id)
	}
	for _, entry := range c.Records.Library {
		lo, hi, known, e := dateSpan(entry.Dates, zone)
		if e != nil {
			return e
		}
		for _, id := range byMedia[entry.ID] {
			row := c.Rows[id]
			row.RecordingDate = lo
			if lo == "" {
				row.RecordingDate = hi
			}
			c.Rows[id] = row
		}
		if f == nil {
			continue
		}
		matches := known
		if !known {
			matches = f.IncludeUndated
		} else {
			if f.From != "" && hi != "" && hi < f.From {
				matches = false
			}
			if f.Through != "" && lo != "" && lo > f.Through {
				matches = false
			}
		}
		if !matches {
			for _, id := range byMedia[entry.ID] {
				delete(c.Rows, id)
			}
		}
	}
	return nil
}
func sourceTime(row Row) (int64, int64, bool) {
	start, e := strconv.ParseInt(row.StartUS, 10, 64)
	end, ee := strconv.ParseInt(row.EndUS, 10, 64)
	return start, end, e == nil && ee == nil
}
