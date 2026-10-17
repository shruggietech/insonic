// SPDX-License-Identifier: Apache-2.0
package timezone

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed zoneinfo.zip windows-zones.json
var zoneFiles embed.FS
var zoneOnce sync.Once
var zones map[string][]byte
var rulesVersion string

const MappingVersion = "CLDR-release-48"

func RulesVersion() string { zoneOnce.Do(loadZones); return rulesVersion }
func loadZones() {
	zones = map[string][]byte{}
	data, _ := zoneFiles.ReadFile("zoneinfo.zip")
	sum := sha256.Sum256(data)
	rulesVersion = "Go-1.27.1-zoneinfo-sha256:" + hex.EncodeToString(sum[:])
	archive, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return
	}
	for _, f := range archive.File {
		r, e := f.Open()
		if e != nil {
			continue
		}
		b, e := io.ReadAll(r)
		r.Close()
		if e == nil {
			zones[f.Name] = b
		}
	}
}

// Location resolves calendar, query and metadata dates from the same pinned rules.
func Location(name string) (*time.Location, error) {
	if name == "" {
		name = "UTC"
	}
	if strings.EqualFold(name, "local") {
		var err error
		name, err = LocalName()
		if err != nil {
			return nil, err
		}
	}
	return loadZone(name)
}
func loadZone(name string) (*time.Location, error) {
	zoneOnce.Do(loadZones)
	if name == "UTC" || name == "Z" {
		return time.UTC, nil
	}
	if offset, ok := ParseOffset(name); ok {
		return time.FixedZone(name, int(offset)), nil
	}
	if data, ok := zones[name]; ok {
		return time.LoadLocationFromTZData(name, data)
	}
	return nil, &zoneError{}
}

type zoneError struct{}

func (*zoneError) Error() string { return "unavailable timezone" }
func ParseOffset(s string) (int64, bool) {
	if len(s) != 6 || (s[0] != '+' && s[0] != '-') || s[3] != ':' {
		return 0, false
	}
	for _, i := range []int{1, 2, 4, 5} {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	h, e1 := strconv.Atoi(s[1:3])
	m, e2 := strconv.Atoi(s[4:])
	if e1 != nil || e2 != nil || h > 23 || m > 59 {
		return 0, false
	}
	offset := int64(h*3600 + m*60)
	if s[0] == '-' {
		offset = -offset
	}
	return offset, true
}
func LocalName() (string, error) { return platformLocalZone() }
func windowsMapping(key string) (string, bool) {
	data, _ := zoneFiles.ReadFile("windows-zones.json")
	var mappings map[string]string
	json.Unmarshal(data, &mappings)
	v, ok := mappings[strings.TrimSpace(key)]
	return v, ok
}
