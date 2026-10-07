// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
)

func TestCLIBoundedDateCorrectionAndRefresh(t *testing.T) {
	id := contracts.ID()
	lower := "2026-10-07T12:00:00.000000001+02:00"
	upper := "2026-10-08T00:00:00Z"
	for _, args := range [][]string{
		{"media", "import", "fixture.wav", "--originated-earliest", lower, "--originated-latest", upper},
		{"media", "refresh", id, "--originated-earliest", lower},
		{"media", "set-origin", id, "--revision", "1", "--originated-earliest", lower, "--originated-latest", upper},
	} {
		op, _, raw, e := parseDomain(args)
		if e != nil {
			t.Fatal(args, e)
		}
		var options library.Options
		switch op {
		case "media.import":
			var req library.ImportRequest
			json.Unmarshal(raw, &req)
			options = req.Defaults
		case "media.set-origin":
			var req struct {
				Options library.Options `json:"options"`
			}
			json.Unmarshal(raw, &req)
			options = req.Options
		default:
			json.Unmarshal(raw, &options)
		}
		expected, _ := time.Parse(time.RFC3339Nano, lower)
		if options.OriginatedEarliest == nil || options.OriginatedEarliest.ISO != "2026-10-07T10:00:00.000000001Z" || options.OriginatedEarliest.UnixNS != expected.UnixNano() {
			t.Fatal(string(raw))
		}
	}
	for _, flags := range [][]string{
		{"--originated-earliest", "not-a-time"},
		{"--originated-latest", "2500-01-01T00:00:00Z"},
		{"--originated-earliest", upper, "--originated-latest", lower},
		{"--originated-earliest", lower, "--originated-on", "2026-10-07"},
	} {
		if _, _, _, e := parseDomain(append([]string{"media", "import", "fixture.wav"}, flags...)); e == nil {
			t.Fatal("invalid bounds admitted", flags)
		}
	}
}
