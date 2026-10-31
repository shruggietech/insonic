// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
)

func (f fixtureSubtitles) AssembleScoped(_ context.Context, raw []byte, duration *int64, turns []subtitles.Turn, participation []subtitles.Participation, cues, voices map[string]int) (subtitles.Assembly, error) {
	return subtitles.AssembleDocumentScoped(raw, duration, turns, participation, cues, voices)
}

func TestWindowReuseRequiresSameNativeSourceSelection(t *testing.T) {
	track := processing.TrackPlan{StreamIndex: 2, SourceSampleRate: 48000, SourceSampleCount: 480000, StartNumerator: "1", StartDenominator: "3", EndSample: 480000}
	plan := processing.ProcessingPlan{Identity: processing.PlanIdentity{SourceDigest: strings.Repeat("a", 64)}, Tracks: []processing.TrackPlan{track}}
	raw, _ := json.Marshal(windowSourceMap{Policy: "windowed-canonical-source-clock-v1", SourceDigest: plan.Identity.SourceDigest, Tracks: plan.Tracks})
	previous := catalog.Recording{SourceMap: raw}
	if err := compatibleWindowReuse(previous, plan); err != nil {
		t.Fatal(err)
	}
	plan.Tracks[0].StartSample = 48000
	if compatibleWindowReuse(previous, plan) == nil {
		t.Fatal("changed selected range reused")
	}
	plan.Tracks[0] = track
	plan.Identity.SourceDigest = strings.Repeat("b", 64)
	if compatibleWindowReuse(previous, plan) == nil {
		t.Fatal("changed source reused")
	}
}
