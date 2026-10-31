// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
	"testing"
)

func TestTranscriptDiarizationCarriesGeneratedDeviceNoticeThroughReplay(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	entry := seedLegacyAdmission(t, s, db, "")
	path, _ := transcriptFixture(t)
	calls := 0
	notice := "CUDA could not execute (cuda_library_missing); completed on CPU."
	s.DiarizeWithPolicy = func(_ context.Context, got catalog.LibraryEntry, doc json.RawMessage, model, digest, device string) (DiarizationResult, error) {
		calls++
		if got.ID != entry.ID || device != "auto" {
			t.Fatalf("changed import election: %s %s", got.ID, device)
		}
		return DiarizationResult{Admission: subtitles.Admission{Document: doc}, Provenance: json.RawMessage(`{"source_map":{"policy":"fixture"},"diarization":{"device":"cpu","requested_device":"auto","attempted_devices":["cuda","cpu"],"fallback_reason":"cuda_library_missing"}}`), Notices: []string{notice}}, nil
	}
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "diarize", DiarizationModelID: contracts.ID(), DiarizationDevice: "auto"}, Items: []Item{{Record: entry.ID, Transcript: path}}})
	for attempt := 0; attempt < 2; attempt++ {
		result, err := s.Execute(ctx, work)
		if err != nil {
			t.Fatal(err)
		}
		item := result.(ImportResult).Items[0]
		if item.State != "admitted" || len(item.Notices) != 1 || item.Notices[0] != notice || calls != 1 {
			t.Fatalf("attempt %d lost generated notice: %+v calls=%d", attempt, item, calls)
		}
	}
}
