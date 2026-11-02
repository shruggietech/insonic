// SPDX-License-Identifier: Apache-2.0
package answers

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/timezone"
	"time"
)

// DateSpan reads only the elected calendar interpretation. Original literals,
// unresolved alternatives and their identity remain in the catalog authority.
func DateSpan(raw []byte, zone string) (string, string, bool, error) {
	var value struct {
		Selected *struct {
			Precision              string `json:"precision"`
			Literal                string `json:"literal"`
			Resolved, Lower, Upper *struct {
				ISO string `json:"iso"`
			}
		} `json:"selected"`
	}
	if json.Unmarshal(raw, &value) != nil || value.Selected == nil {
		return "", "", false, nil
	}
	if zone == "" {
		zone = "UTC"
	}
	loc, err := timezone.Location(zone)
	if err != nil {
		return "", "", false, contracts.Fail("invalid_request")
	}
	d := value.Selected
	if d.Precision == "day" && len(d.Literal) >= 10 {
		day := d.Literal[:10]
		if _, err := time.Parse("2006-01-02", day); err == nil {
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
