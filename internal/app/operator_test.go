// SPDX-License-Identifier: Apache-2.0
package app

import (
	"github.com/shruggietech/insonic/internal/catalog"
	"testing"
)

func TestDatasetPreparationReadiness(t *testing.T) {
	for _, tc := range []struct {
		state string
		count int
		want  string
	}{{"current", 0, "empty"}, {"current", 1, "ready_for_preparation"}, {"invalidated", 1, "invalidated"}} {
		d := catalog.SpeakerDataset{}
		d.Dataset.State = tc.state
		for range tc.count {
			d.References = append(d.References, catalog.CurrentReference{})
		}
		view := speakerDatasetSummary(d).(map[string]any)
		if view["readiness"] != tc.want {
			t.Fatal(tc, view)
		}
	}
}
