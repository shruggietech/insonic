// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func audioElection(o Options) bool { return o.ReplaceAudio != nil && *o.ReplaceAudio }
func targetedMedia(i Item, o Options) bool {
	return i.Record != "" && i.Source != "" && (i.Kind == "media" || audioElection(o))
}
func validateReplacementItem(i Item, o Options) error {
	if i.AcceptedReceipt != "" {
		return nil
	}
	media := targetedMedia(i, o)
	if audioElection(o) && !media || (o.ExistingTranscript != "" || o.ExistingRoster != "" || o.TranscriptApplies != nil) && !media || media && i.NewEntry || i.Record != "" && o.KnownSpeakers != nil {
		return contracts.Fail("invalid_request")
	}
	if media && audioElection(o) {
		supplied := i.Transcript != "" || i.Subtitle != ""
		if (o.ExistingTranscript == "keep" || o.ExistingTranscript == "clear") && (supplied || o.SubtitleStreamIndex != nil || o.SubtitleLanguage != "") {
			return contracts.Fail("invalid_request")
		}
		if o.TranscriptApplies != nil && *o.TranscriptApplies && o.ExistingTranscript != "keep" {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
func (s *Service) replacementTarget(ctx context.Context, i Item, o Options) (catalog.LibraryEntry, *catalog.Recording, error) {
	if i.TargetError != "" {
		return catalog.LibraryEntry{}, nil, contracts.Fail(i.TargetError)
	}
	entry, e := s.target(ctx, i.Record)
	if e != nil {
		return entry, nil, e
	}
	current, e := s.Catalog.Recording(ctx, entry.ID)
	rev := int64(0)
	var doc *catalog.Recording
	if e == nil {
		rev = current.Revision
		doc = &current
	} else if code(e) != "not_found" {
		return entry, nil, e
	}
	if i.ExpectedRevision != nil && *i.ExpectedRevision != entry.Revision || i.ExpectedRecordingRevision != nil && *i.ExpectedRecordingRevision != rev {
		return entry, nil, contracts.Fail("conflict")
	}
	if audioElection(o) {
		hasTranscript := doc != nil && doc.DocumentDigest != "" || entry.SubtitlePublicationID != nil
		if hasTranscript && o.ExistingTranscript == "" {
			return entry, nil, contracts.Fail("invalid_request")
		}
		if o.ExistingTranscript == "keep" && (!hasTranscript || o.TranscriptApplies == nil || !*o.TranscriptApplies) {
			return entry, nil, contracts.Fail("invalid_request")
		}
		roster, e := s.Catalog.Roster(ctx, entry.ID)
		if e != nil {
			return entry, nil, e
		}
		if i.ExpectedRosterRevision != nil && *i.ExpectedRosterRevision != roster.Revision {
			return entry, nil, contracts.Fail("conflict")
		}
		if roster.Declared && o.ExistingRoster == "" {
			return entry, nil, contracts.Fail("invalid_request")
		}
	}
	return entry, doc, nil
}
func validateKeptSelection(old, candidate catalog.LibraryEntry, previous *catalog.Recording) error {
	var mapping struct {
		StreamIndex *int `json:"stream_index"`
		Channel     *int `json:"channel"`
	}
	if json.Unmarshal(previous.SourceMap, &mapping) != nil {
		return contracts.Fail("invalid_request")
	}
	if mapping.StreamIndex == nil {
		return validateKeptDocument(candidate, previous.Document)
	}
	var before, after Facts
	if json.Unmarshal(old.Facts, &before) != nil || json.Unmarshal(candidate.Facts, &after) != nil {
		return contracts.Fail("invalid_request")
	}
	var selected *Stream
	for i := range before.Streams {
		if before.Streams[i].Kind == "audio" && before.Streams[i].Index == *mapping.StreamIndex {
			selected = &before.Streams[i]
		}
	}
	if selected == nil {
		return contracts.Fail("invalid_request")
	}
	for _, stream := range after.Streams {
		if stream.Kind != "audio" || stream.Index != *mapping.StreamIndex {
			continue
		}
		if selected.Channels != nil && stream.Channels != nil && *selected.Channels != *stream.Channels || mapping.Channel != nil && (*mapping.Channel < 0 || stream.Channels != nil && int64(*mapping.Channel) >= *stream.Channels) {
			return contracts.Fail("invalid_request")
		}
		after.Streams = []Stream{stream}
		if after.Canonical != nil {
			copy := *after.Canonical
			copy.Tracks = nil
			for _, track := range after.Canonical.Tracks {
				if track.Index == stream.Index {
					copy.Tracks = append(copy.Tracks, track)
				}
			}
			after.Canonical = &copy
		}
		candidate.Facts = marshal(after)
		return validateKeptDocument(candidate, previous.Document)
	}
	return contracts.Fail("invalid_request")
}
func validateKeptDocument(entry catalog.LibraryEntry, document json.RawMessage) error {
	var doc struct {
		Cues []struct {
			Timing struct {
				Start *int64 `json:"start_milliseconds"`
				End   *int64 `json:"end_milliseconds"`
			} `json:"timing"`
			Assignments []json.RawMessage `json:"speaker_attributions"`
		} `json:"cues"`
	}
	if json.Unmarshal(document, &doc) != nil {
		return contracts.Fail("invalid_request")
	}
	// Validate native intervals through the same exact source-clock bounds as
	// consumer intervals, without rewriting the owner-approved document.
	for i := range doc.Cues {
		cue := &doc.Cues[i]
		if cue.Timing.Start != nil && cue.Timing.End != nil {
			b, _ := json.Marshal(map[string]any{"start_milliseconds": *cue.Timing.Start, "end_milliseconds": *cue.Timing.End})
			cue.Assignments = append(cue.Assignments, b)
		}
	}
	raw, _ := json.Marshal(doc)
	return validateAudioAttribution(entry, raw)
}
