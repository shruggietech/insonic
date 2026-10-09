// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"github.com/shruggietech/insonic/internal/processing"
	"testing"
)

func TestSourceClockSampleProjection(t *testing.T) {
	mapping := processing.SourceMap{StartNumerator: "1", StartDenominator: "3", SampleRate: 16000, SampleCount: 32000}
	first, last, err := sampleBounds("334/1000", "1", mapping)
	if err != nil || first != 11 || last != 10666 {
		t.Fatalf("fractional source mapping: %d %d %v", first, last, err)
	}
	if _, _, err = sampleBounds("0", "1", mapping); err == nil {
		t.Fatal("outside source accepted")
	}
}
