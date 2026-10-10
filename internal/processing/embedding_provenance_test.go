// SPDX-License-Identifier: Apache-2.0
package processing

import "testing"

func TestEmbeddingWorkerInputCountProvenance(t *testing.T) {
	for _, count := range []float64{1, 64} {
		if !validProvenance(map[string]any{"input_count": count, "offline": true}) {
			t.Fatal("valid worker batch provenance rejected", count)
		}
	}
	for _, count := range []any{float64(0), float64(65), float64(1.5), "2"} {
		if validProvenance(map[string]any{"input_count": count}) {
			t.Fatal("invalid batch count accepted", count)
		}
	}
}
