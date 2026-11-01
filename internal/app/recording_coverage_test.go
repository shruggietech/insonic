// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/processing"
	"os"
	"strings"
	"testing"
)

func TestRecordingCoverageLegacyAndAmbiguousTracks(t *testing.T) {
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	tracks := []processing.TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 48000, StartNumerator: "0", StartDenominator: "1", EndSample: 48000}, {StreamIndex: 1, SourceSampleRate: 16000, SourceSampleCount: 48000, StartNumerator: "0", StartDenominator: "1", EndSample: 48000}}
	source, _ := json.Marshal(windowSourceMap{Policy: "windowed-canonical-source-clock-v1", SourceDigest: strings.Repeat("a", 64), Tracks: tracks})
	r := catalog.Recording{State: "ready", Document: doc, DocumentDigest: documentHash(doc), SourceDigest: strings.Repeat("a", 64), SourceRevision: 1, Revision: 2, SourceMap: source, Provenance: json.RawMessage("{}")}
	in, e := recordingCoverageInput(r)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range in.Cues {
		if c.TrackOrdinal != -1 {
			t.Fatal("guessed source lane")
		}
	}
	report, e := processing.EvaluateCoverage(in, processing.CoverageOptions{})
	if e != nil || report.Tracks[0].Metrics.UnknownActivityUS != 3000000 || report.Tracks[0].Metrics.CueCoverageUS != 0 {
		t.Fatalf("unknown coverage: %+v %v", report, e)
	}
}
