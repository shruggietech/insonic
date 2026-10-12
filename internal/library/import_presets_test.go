// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func selectedFixtureImport(items ...Item) ImportRequest {
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	options := Options{Timezone: "UTC", Attribution: "off"}
	request := ImportRequest{Items: items, Selections: map[int]string{}, Elections: map[string]ImportSelection{key: {Options: options}}}
	for i := range items {
		request.Selections[i] = key
	}
	return request
}
func TestPresetProcessingOnlyForAcceptedItems(t *testing.T) {
	t.Run("new-admission", func(t *testing.T) {
		s, db := libraryFixture(t)
		calls := 0
		s.AfterAdmission = func(context.Context, catalog.Work, int, string, ImportSelection) ([]string, error) {
			calls++
			return []string{contracts.ID()}, nil
		}
		request := selectedFixtureImport(Item{Source: wavFixture(t)})
		result, err := s.Import(context.Background(), claimLibraryWork(t, db, "media.import", request), request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Items[0].State != "admitted" || calls != 1 || len(result.Items[0].QueuedJobIDs) != 1 {
			t.Fatalf("accepted admission lost processing: %+v calls=%d", result.Items, calls)
		}
	})
	t.Run("duplicate-content", func(t *testing.T) {
		s, db := libraryFixture(t)
		source := wavFixture(t)
		entry := seedLegacyAdmission(t, s, db, "")
		digest, size, err := identity(context.Background(), source)
		if err != nil {
			t.Fatal(err)
		}
		var facts Facts
		if err = json.Unmarshal(entry.Facts, &facts); err != nil {
			t.Fatal(err)
		}
		// Seed canonical source identity so deduplication is exercised without invoking media tools.
		facts.Canonical = &Canonical{SourceDigest: digest, SourceSize: size}
		entry.Facts = marshal(facts)
		if _, err = db.CommitLibrary(context.Background(), claimLibraryWork(t, db, "media.import", ImportRequest{}), entry); err != nil {
			t.Fatal(err)
		}
		s.legacyFixture = false
		calls := 0
		s.AfterAdmission = func(context.Context, catalog.Work, int, string, ImportSelection) ([]string, error) {
			calls++
			return []string{contracts.ID()}, nil
		}
		request := selectedFixtureImport(Item{Source: source}, Item{Source: source})
		result, err := s.Import(context.Background(), claimLibraryWork(t, db, "media.import", request), request)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 2 || result.Items[0].State != "skipped" || result.Items[1].State != "skipped" || calls != 0 || len(result.Items[0].QueuedJobIDs) != 0 || len(result.Items[1].QueuedJobIDs) != 0 || result.Items[0].Selection == nil || result.Items[1].Selection == nil {
			t.Fatalf("duplicate launched processing: calls=%d items=%+v", calls, result.Items)
		}
	})
	t.Run("existing-transcript", func(t *testing.T) {
		s, db := libraryFixture(t)
		entry := seedLegacyAdmission(t, s, db, "")
		path, _ := transcriptFixture(t)
		if first := executeTranscript(t, s, db, Item{Record: entry.ID, Transcript: path}); first.State != "admitted" {
			t.Fatal(first)
		}
		calls := 0
		s.AfterAdmission = func(context.Context, catalog.Work, int, string, ImportSelection) ([]string, error) {
			calls++
			return []string{contracts.ID()}, nil
		}
		request := selectedFixtureImport(Item{Record: entry.ID, Transcript: path})
		result, err := s.Import(context.Background(), claimLibraryWork(t, db, "media.import", request), request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Items[0].State != "skipped" || calls != 0 || len(result.Items[0].QueuedJobIDs) != 0 {
			t.Fatalf("no-replacement launched processing: %+v calls=%d", result.Items, calls)
		}
		replace := true
		request = selectedFixtureImport(Item{Record: entry.ID, Transcript: path})
		for key, selected := range request.Elections {
			selected.Options.ReplaceTranscript = &replace
			request.Elections[key] = selected
		}
		result, err = s.Import(context.Background(), claimLibraryWork(t, db, "media.import", request), request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Items[0].State != "replaced" || calls != 1 || len(result.Items[0].QueuedJobIDs) != 1 {
			t.Fatalf("accepted replacement lost processing: %+v calls=%d", result.Items, calls)
		}
	})
}
