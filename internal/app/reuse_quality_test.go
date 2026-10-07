// SPDX-License-Identifier: Apache-2.0
package app

import (
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"testing"
)

func TestReuseRejectsDifferentChannelStreamAndClock(t *testing.T) {
	channel := 0
	old := processing.SourceMap{StartNumerator: "1", StartDenominator: "2", SampleRate: 16000, SampleCount: 16000, StreamIndex: 1, Channel: &channel, Policy: "selected-channel"}
	now := old
	now.StartNumerator = "2"
	now.StartDenominator = "4"
	if compatibleReuseMap(configurationBytes(old), now) != nil {
		t.Fatal("equivalent rational origin rejected")
	}
	other := 1
	now.Channel = &other
	if compatibleReuseMap(configurationBytes(old), now) == nil {
		t.Fatal("channel replacement accepted")
	}
	now = old
	now.StreamIndex = 2
	if compatibleReuseMap(configurationBytes(old), now) == nil {
		t.Fatal("stream replacement accepted")
	}
	now = old
	now.StartNumerator = "2"
	if compatibleReuseMap(configurationBytes(old), now) == nil {
		t.Fatal("changed origin accepted")
	}
	if compatibleReuseMap([]byte(`{"policy":"supplied-subtitle-source-clock"}`), old) == nil {
		t.Fatal("unbound source evidence accepted")
	}
}

func TestReusedQualityUsesCurrentSourceClockAndElection(t *testing.T) {
	m := processing.SourceMap{StartNumerator: "1", StartDenominator: "2", SampleRate: 16000, SampleCount: 16000}
	turns := []subtitles.Turn{{SpeakerID: "voice", StartNS: 500000000, EndNS: 1000000000}}
	minimum := 0.8
	ds, e := reusedQuality(turns, m, &processing.QualityConfig{MinSpeechCoverage: &minimum})
	if e != nil {
		t.Fatal(e)
	}
	coverage, threshold := false, false
	for _, d := range ds {
		if d.Code == "speech_coverage_fraction" && d.Value != nil && *d.Value == 0.5 {
			coverage = true
		}
		if d.Code == "speech_coverage_below_threshold" {
			threshold = true
		}
	}
	if !coverage || !threshold {
		t.Fatal("new quality election/source offset ignored")
	}
	disabled := false
	ds, e = reusedQuality(turns, m, &processing.QualityConfig{Enabled: &disabled})
	if e != nil || len(ds) != 0 {
		t.Fatal("quality disable ignored", e)
	}
}
