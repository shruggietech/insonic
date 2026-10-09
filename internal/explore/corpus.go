// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
	"github.com/shruggietech/insonic/internal/graph"
	"math"
	"sort"
	"strconv"
	"strings"
)

type SourceSpan struct {
	CueID   string `json:"cue_id"`
	StartUS string `json:"source_start_us"`
	EndUS   string `json:"source_end_us"`
}
type Row struct {
	SourceSpans       []SourceSpan        `json:"source_spans,omitempty"`
	ID                string              `json:"id"`
	Kind              string              `json:"kind"`
	Label             string              `json:"label"`
	MediaID           string              `json:"media_id,omitempty"`
	MediaRevision     int64               `json:"media_revision,omitempty"`
	RecordingRevision int64               `json:"recording_revision,omitempty"`
	DocumentDigest    string              `json:"document_digest,omitempty"`
	CueID             string              `json:"cue_id,omitempty"`
	Text              string              `json:"text,omitempty"`
	SpeakerIDs        []string            `json:"speaker_ids,omitempty"`
	LocalSpeakerIDs   []string            `json:"local_speaker_ids,omitempty"`
	StartUS           string              `json:"source_start_us,omitempty"`
	EndUS             string              `json:"source_end_us,omitempty"`
	SeekSeconds       *float64            `json:"seek_seconds,omitempty"`
	RecordingDate     string              `json:"recording_date,omitempty"`
	ModelKind         string              `json:"model_kind,omitempty"`
	Provenance        any                 `json:"provenance,omitempty"`
	Statement         *evidence.Assertion `json:"statement,omitempty"`
}
type Corpus struct {
	Refs     graph.References
	Rows     map[string]Row
	Records  catalog.Records
	Revision int64
}

func StableID(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	h[6] = (h[6] & 15) | 80
	h[8] = (h[8] & 63) | 128
	s := hex.EncodeToString(h[:16])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
func ref(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
func Build(s catalog.Snapshot) (Corpus, error) {
	c := Corpus{Refs: graph.References{Nodes: []graph.Node{}, Edges: []graph.Edge{}}, Rows: map[string]Row{}, Records: s.Records, Revision: s.Revision}
	node := func(id, kind string, reference any, row Row) {
		if _, ok := c.Rows[id]; ok {
			return
		}
		row.ID = id
		row.Kind = kind
		c.Rows[id] = row
		c.Refs.Nodes = append(c.Refs.Nodes, graph.Node{ID: id, Kind: kind, Reference: ref(reference)})
	}
	edgeIDs := map[string]bool{}
	edge := func(from, to, kind string) {
		eid := StableID(from, to, kind)
		if edgeIDs[eid] {
			return
		}
		edgeIDs[eid] = true
		if _, ok := c.Rows[from]; !ok {
			return
		}
		if _, ok := c.Rows[to]; !ok {
			return
		}
		c.Refs.Edges = append(c.Refs.Edges, graph.Edge{ID: StableID(from, to, kind), From: from, To: to, Kind: kind})
	}
	for _, v := range s.Records.Speakers {
		node("speaker:"+v.ID, "speaker", map[string]any{"speaker_id": v.ID, "revision": v.Revision}, Row{Label: v.Name, SpeakerIDs: []string{v.ID}})
	}
	for _, v := range s.Records.Aliases {
		node("alias:"+v.ID, "speaker-alias", map[string]any{"alias_id": v.ID}, Row{Label: v.Text, SpeakerIDs: []string{v.SpeakerID}})
		edge("speaker:"+v.SpeakerID, "alias:"+v.ID, "has-alias")
	}
	for _, v := range s.Records.Terms {
		node("term:"+v.ID, "term", map[string]any{"term_id": v.ID, "revision": v.Revision}, Row{Label: v.Canonical})
		if v.SpeakerID != nil {
			edge("speaker:"+*v.SpeakerID, "term:"+v.ID, "uses-term")
		}
	}
	for _, v := range s.Records.Artifacts {
		node("artifact:"+v.ID, "artifact", map[string]any{"artifact_id": v.ID}, Row{Label: v.Kind + " " + v.Digest})
	}
	for _, v := range s.Records.Assets {
		node("asset:"+v.ID, "asset", map[string]any{"asset_id": v.ID}, Row{Label: v.Role})
		edge("asset:"+v.ID, "artifact:"+v.ArtifactID, "stored-as")
	}
	for _, v := range s.Records.Library {
		row := Row{Label: v.Title, MediaID: v.ID, MediaRevision: v.Revision, Provenance: map[string]any{"class": v.Class, "selected_dates": wireDates(v.Dates)}}
		node("media:"+v.ID, "media", map[string]any{"media_id": v.ID, "revision": v.Revision}, row)
		edge("media:"+v.ID, "asset:"+v.AssetID, "has-asset")
		node("metadata:"+v.ID, "metadata", map[string]any{"media_id": v.ID, "revision": v.Revision}, Row{Label: "Raw metadata", MediaID: v.ID, MediaRevision: v.Revision})
		edge("media:"+v.ID, "metadata:"+v.ID, "has-metadata")
		node("date:"+v.ID, "date-selection", map[string]any{"media_id": v.ID, "revision": v.Revision}, Row{Label: "Selected origination date", MediaID: v.ID, MediaRevision: v.Revision, Provenance: wireDates(v.Dates)})
		edge("media:"+v.ID, "date:"+v.ID, "originated")
	}
	for _, v := range s.Records.Media {
		node("media:"+v.ID, "media", map[string]any{"media_id": v.ID}, Row{Label: v.Title, MediaID: v.ID, Provenance: map[string]any{"class": v.Class}})
	}
	for _, v := range s.Records.MediaAssets {
		edge("media:"+v.MediaID, "asset:"+v.AssetID, "has-asset")
	}
	for _, h := range s.Records.Rosters {
		node("roster:"+h.ID, "declared-roster", map[string]any{"recording_id": h.ID, "revision": h.Revision}, Row{Label: "Declared speaker roster", MediaID: h.ID, Provenance: map[string]any{"declared": true, "revision": h.Revision}})
		edge("media:"+h.ID, "roster:"+h.ID, "has-roster")
	}
	for _, m := range s.Records.RosterMembers {
		edge("media:"+m.RecordingID, "speaker:"+m.SpeakerID, "declared-speaker")
		edge("roster:"+m.RecordingID, "speaker:"+m.SpeakerID, "declares")
	}
	for _, v := range s.Records.Metadata {
		node("snapshot:"+v.ID, "metadata-snapshot", map[string]any{"snapshot_id": v.ID}, Row{Label: v.Extractor + " " + v.State})
		edge("asset:"+v.AssetID, "snapshot:"+v.ID, "has-metadata-snapshot")
		if v.ReportArtifactID != nil {
			edge("snapshot:"+v.ID, "artifact:"+*v.ReportArtifactID, "raw-report")
		}
	}
	for _, v := range s.Records.Observations {
		node("observation:"+v.ID, "metadata-observation", map[string]any{"observation_id": v.ID}, Row{Label: v.Family + " " + v.Tag})
		edge("snapshot:"+v.SnapshotID, "observation:"+v.ID, "has-observation")
	}
	for _, v := range s.Records.Dates {
		node("date-observation:"+v.ID, "date-observation", map[string]any{"date_id": v.ID}, Row{Label: v.Literal, MediaID: v.MediaID, Provenance: map[string]any{"precision": v.Precision, "basis": v.Basis, "zone": v.Zone}})
		edge("media:"+v.MediaID, "date-observation:"+v.ID, "has-date-observation")
		if v.ObservationID != nil {
			edge("date-observation:"+v.ID, "observation:"+*v.ObservationID, "derived-from")
		}
	}
	for _, v := range s.Records.DateSelections {
		node("date-selection:"+v.ID, "date-selection", map[string]any{"selection_id": v.ID, "revision": v.Revision}, Row{Label: v.Policy, MediaID: v.MediaID})
		edge("media:"+v.MediaID, "date-selection:"+v.ID, "selected-date")
		if v.DateID != nil {
			edge("date-selection:"+v.ID, "date-observation:"+*v.DateID, "selects")
		}
	}
	entries := map[string]catalog.LibraryEntry{}
	for _, v := range s.Records.Library {
		entries[v.ID] = v
	}
	recordings := map[string]catalog.Recording{}
	for _, v := range s.Records.Recordings {
		recordings[v.ID] = v
		if v.DocumentDigest == "" || string(v.Document) == "null" {
			continue
		}
		cues, e := evidence.ParseCues(v.Document)
		if e != nil {
			return c, e
		}
		entry := entries[v.ID]
		base := Row{Label: entry.Title, MediaID: v.ID, MediaRevision: entry.Revision, RecordingRevision: v.Revision, DocumentDigest: v.DocumentDigest}
		docID := "document:" + v.ID + ":" + v.DocumentDigest
		node(docID, "document", map[string]any{"media_id": v.ID, "recording_revision": v.Revision, "document_digest": v.DocumentDigest}, base)
		edge("media:"+v.ID, docID, "has-current-document")
		for _, cue := range cues {
			row := base
			row.CueID = cue.ID
			row.Text = cue.Text
			row.Label = cue.Text
			row.LocalSpeakerIDs = cue.Voices
			if cue.Timed {
				if cue.StartMS > math.MaxInt64/1000 || cue.EndMS > math.MaxInt64/1000 {
					return c, contracts.Fail("invalid_request")
				}
				row.StartUS = strconv.FormatInt(cue.StartMS*1000, 10)
				row.EndUS = strconv.FormatInt(cue.EndMS*1000, 10)
				sec := float64(cue.StartMS) / 1000
				row.SeekSeconds = &sec
			}
			cueID := "cue:" + v.ID + ":" + v.DocumentDigest + ":" + cue.ID
			for _, m := range s.Records.SpeakerMappings {
				if m.RecordingID == v.ID && m.DocumentDigest == v.DocumentDigest && contains(cue.Voices, m.LocalSpeakerID) {
					row.SpeakerIDs = append(row.SpeakerIDs, m.SpeakerID)
				}
			}
			sort.Strings(row.SpeakerIDs)
			node(cueID, "cue", map[string]any{"media_id": v.ID, "document_digest": v.DocumentDigest, "cue_id": cue.ID}, row)
			edge(docID, cueID, "contains-cue")
			for _, voice := range cue.Voices {
				voiceID := localVoiceID(v.ID, v.DocumentDigest, voice)
				node(voiceID, "local-voice", map[string]any{"media_id": v.ID, "document_digest": v.DocumentDigest, "local_speaker_id": voice}, Row{Label: voice, MediaID: v.ID, RecordingRevision: v.Revision, DocumentDigest: v.DocumentDigest})
				edge(cueID, voiceID, "attributes-to")
			}
			for _, term := range s.Records.Terms {
				if term.State != "active" {
					continue
				}
				variants := []string{term.Canonical}
				var synonyms []string
				_ = json.Unmarshal(term.Variants, &synonyms)
				variants = append(variants, synonyms...)
				for _, word := range variants {
					if strings.Contains(strings.ToLower(cue.Text), strings.ToLower(word)) {
						edge(cueID, "term:"+term.ID, "mentions-term")
						break
					}
				}
			}
		}
	}
	for _, m := range s.Records.SpeakerMappings {
		r, ok := recordings[m.RecordingID]
		if !ok || r.DocumentDigest != m.DocumentDigest {
			continue
		}
		edge(localVoiceID(m.RecordingID, m.DocumentDigest, m.LocalSpeakerID), "speaker:"+m.SpeakerID, "maps-to")
	}
	for _, v := range s.Records.Extractions {
		rec, ok := recordings[v.ID]
		if !ok || rec.Revision != v.RecordingRevision || rec.DocumentDigest != v.DocumentDigest {
			continue
		}
		var assertions []evidence.Assertion
		if json.Unmarshal(v.Assertions, &assertions) != nil {
			continue
		}
		for _, a := range assertions {
			aid := "assertion:" + v.ID + ":" + v.DocumentDigest + ":" + a.ID
			row := Row{Label: a.Subject + " " + a.Relation + " " + a.Object, MediaID: v.ID, MediaRevision: entries[v.ID].Revision, RecordingRevision: rec.Revision, DocumentDigest: rec.DocumentDigest, Statement: &a, Provenance: map[string]any{"extraction_revision": v.Revision, "adapter": v.Provenance}}
			for _, cid := range a.CueIDs {
				cue := c.Rows["cue:"+v.ID+":"+v.DocumentDigest+":"+cid]
				if cue.StartUS != "" {
					row.SourceSpans = append(row.SourceSpans, SourceSpan{cid, cue.StartUS, cue.EndUS})
					start, _ := strconv.ParseInt(cue.StartUS, 10, 64)
					old, _ := strconv.ParseInt(row.StartUS, 10, 64)
					if row.StartUS == "" || start < old {
						row.StartUS = cue.StartUS
						row.EndUS = cue.EndUS
						row.CueID = cid
						row.SeekSeconds = cue.SeekSeconds
					}
				}
			}
			node(aid, "assertion", map[string]any{"media_id": v.ID, "document_digest": v.DocumentDigest, "extraction_revision": v.Revision, "assertion_id": a.ID}, row)
			for _, cid := range a.CueIDs {
				edge(aid, "cue:"+v.ID+":"+v.DocumentDigest+":"+cid, "supported-by")
			}
			for _, p := range []struct{ value, kind string }{{a.Subject, "subject"}, {a.Relation, "relation"}, {a.Object, "object"}} {
				id := StableID(p.kind, p.value)
				node("concept:"+id, "concept", map[string]any{"concept_id": id}, Row{Label: p.value})
				edge(aid, "concept:"+id, p.kind)
			}
		}
	}
	for _, v := range s.Records.Segments {
		rec, ok := recordings[v.RecordingID]
		if !ok || rec.DocumentDigest != v.DocumentDigest {
			continue
		}
		cue := "cue:" + v.RecordingID + ":" + v.DocumentDigest + ":" + v.CueID
		if _, ok := c.Rows[cue]; !ok {
			continue
		}
		row := c.Rows[cue]
		row.Label = "Speaker segment " + v.ID
		node("segment:"+v.ID, "segment", map[string]any{"segment_id": v.ID, "revision": v.Revision, "document_digest": v.DocumentDigest}, row)
		edge("segment:"+v.ID, cue, "segment-of")
	}
	for _, v := range s.Records.Datasets {
		node("dataset:"+v.ID, "dataset", map[string]any{"dataset_id": v.ID}, Row{Label: v.ID, SpeakerIDs: []string{v.SpeakerID}})
		edge("dataset:"+v.ID, "speaker:"+v.SpeakerID, "for-speaker")
	}
	for _, v := range s.Records.Members {
		edge("dataset:"+v.DatasetID, "segment:"+v.SegmentID, "contains-segment")
	}
	for _, v := range s.Records.Runs {
		node("run:"+v.ID, "training-run", map[string]any{"run_id": v.ID}, Row{Label: v.Adapter, SpeakerIDs: []string{v.SpeakerID}})
		edge("run:"+v.ID, "dataset:"+v.DatasetID, "trained-on")
	}
	for _, v := range s.Records.Models {
		node("model:"+v.ID, "model", map[string]any{"model_id": v.ID}, Row{Label: v.Name, SpeakerIDs: []string{v.SpeakerID}})
		edge("model:"+v.ID, "speaker:"+v.SpeakerID, "for-speaker")
	}
	for _, v := range s.Records.Versions {
		node("version:"+v.ID, "model-version", map[string]any{"version_id": v.ID}, Row{Label: v.Kind, ModelKind: v.Kind})
		edge("model:"+v.ModelID, "version:"+v.ID, "has-version")
		edge("version:"+v.ID, "run:"+v.RunID, "produced-by")
	}
	for _, v := range s.Records.ModelAssociations {
		edge("model:"+v.ModelID, "speaker:"+v.SpeakerID, "associated-with")
	}
	for _, v := range s.Records.BaseModels {
		node("base-model:"+v.ID, "base-model", map[string]any{"base_model_id": v.ID, "revision": v.Revision}, Row{Label: v.Name, ModelKind: "base"})
	}
	// Normalize order so identical catalog facts yield identical events on either backend.
	sort.Slice(c.Refs.Nodes, func(i, j int) bool { return c.Refs.Nodes[i].ID < c.Refs.Nodes[j].ID })
	sort.Slice(c.Refs.Edges, func(i, j int) bool { return c.Refs.Edges[i].ID < c.Refs.Edges[j].ID })
	return c, nil
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func localVoiceID(recording, document, token string) string {
	sum := sha256.Sum256([]byte(recording + "\x00" + document + "\x00" + token))
	return "voice:" + hex.EncodeToString(sum[:])
}
