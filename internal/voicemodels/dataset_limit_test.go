// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"strconv"
	"strings"
	"testing"
)

type oversizedCorpusCatalog struct {
	catalog.Catalog
	seen  int
	calls int
}

func (s *oversizedCorpusCatalog) ReplaySpeakerDataset(context.Context, string, string, json.RawMessage) (catalog.SpeakerDataset, bool, error) {
	return catalog.SpeakerDataset{}, false, nil
}
func (s *oversizedCorpusCatalog) CurrentSpeakerReferences(_ context.Context, selection catalog.SpeakerSelection) (catalog.SpeakerSelectionPage, error) {
	count := selection.Limit
	remaining := catalog.SpeakerDatasetReferenceLimit + 1 - s.seen
	if count > remaining {
		count = remaining
	}
	s.seen += count
	s.calls++
	next := ""
	if s.seen < catalog.SpeakerDatasetReferenceLimit+1 {
		next = strconv.Itoa(s.seen)
	}
	return catalog.SpeakerSelectionPage{References: make([]catalog.CurrentReference, count), Epoch: strings.Repeat("a", 64), Revision: 1, Next: next}, nil
}

func TestDatasetReferenceLimitRejectsCompleteOversizedSelectionBeforePublication(t *testing.T) {
	selected := &oversizedCorpusCatalog{}
	// Deliberately absent artifact and evidence services prove the resource bound
	// is checked before decoding, manifest creation, or any managed publication.
	service := &Service{Catalog: selected}
	_, err := service.CreateDataset(context.Background(), contracts.ID(), DatasetOptions{SpeakerID: contracts.ID()})
	if errorCode(err) != "input_limit" || selected.seen != catalog.SpeakerDatasetReferenceLimit+1 || selected.calls != catalog.SpeakerDatasetReferenceLimit/100+1 {
		t.Fatal("selection truncated or reached publication before limit", selected.seen, selected.calls, err)
	}
}
