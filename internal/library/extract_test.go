// SPDX-License-Identifier: Apache-2.0
package library

import (
	"encoding/json"
	"testing"
)

func TestProbeExactDurationAndSourceClocks(t *testing.T) {
	raw := []byte(`{"format":{"format_name":"wav","duration":"1.000001"},"streams":[{"index":0,"codec_type":"audio","codec_name":"pcm_s16le","sample_rate":"48000","channels":2,"channel_layout":"stereo","time_base":"1/48000","start_pts":0,"duration_ts":48001}]}`)
	facts, e := parseProbe(raw)
	if e != nil || facts.DurationUS == nil || *facts.DurationUS != 1000001 || len(facts.Streams) != 1 || facts.Streams[0].TimeBase != "1/48000" {
		t.Fatalf("facts %+v %v", facts, e)
	}
	facts, e = parseProbe([]byte(`{"format":{},"streams":[]}`))
	if e != nil || facts.DurationUS != nil {
		t.Fatal("unknown became zero")
	}
	facts, e = parseProbe([]byte(`{"format":{"duration":"0"},"streams":[]}`))
	if e != nil || facts.DurationUS == nil || *facts.DurationUS != 0 {
		t.Fatal("known zero lost")
	}
	if _, e = parseProbe([]byte(`{"format":{"duration":"NaN"},"streams":[]}`)); e == nil {
		t.Fatal("invalid duration accepted")
	}
}
func TestQualifiedRawMetadataAndDateCandidates(t *testing.T) {
	raw := []byte(`[{"SourceFile":"scratch","EXIF:ExifIFD:Copy1:DateTimeOriginal":"2026:10:07 14:30:00","XMP:XMP-exif:Copy2:DateTimeOriginal":"2026:10:06 14:30:00","XMP:XMP-dc:Copy1:Subject":["first","second"],"File:File:MIMEType":"audio/wav"}]`)
	observations, mime, e := parseExif(raw)
	if e != nil || len(observations) != 3 || mime != "audio/wav" {
		t.Fatalf("observations %+v %v", observations, e)
	}
	candidates := embeddedDates(observations, Options{Timezone: "UTC"})
	if len(candidates) != 2 || candidates[0].Resolved == nil {
		t.Fatalf("dates %+v", candidates)
	}
	var values []string
	if json.Unmarshal(observations[2].Raw, &values) != nil { // deterministic sorting places XMP-exif last
		found := false
		for _, o := range observations {
			if o.Tag == "Subject" && json.Unmarshal(o.Raw, &values) == nil && len(values) == 2 {
				found = true
			}
		}
		if !found {
			t.Fatal("structured value lost")
		}
	}
}
