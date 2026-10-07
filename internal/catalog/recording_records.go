// SPDX-License-Identifier: Apache-2.0
package catalog

import "encoding/json"

// Recording owns the only persisted current subtitle/speaker-assignment document.
// A null document distinguishes no-speech from speech without usable cue timing.
type Recording struct {
	ID                       string          `json:"id"`
	Revision                 int64           `json:"revision"`
	SourceDigest             string          `json:"source_digest"`
	SourceRevision           int64           `json:"source_revision"`
	State                    string          `json:"state"`
	Document                 json.RawMessage `json:"document"`
	DocumentDigest           string          `json:"document_digest"`
	MappedAudioPublicationID *string         `json:"mapped_audio_publication_id"`
	SourceMap                json.RawMessage `json:"source_map"`
	Provenance               json.RawMessage `json:"provenance"`
	Diagnostics              json.RawMessage `json:"diagnostics"`
}

// SpeakerMapping is external identity correlation, never another assignment list.
type SpeakerMapping struct {
	ID             string `json:"id"`
	RecordingID    string `json:"recording_id"`
	LocalSpeakerID string `json:"local_speaker_id"`
	SpeakerID      string `json:"speaker_id"`
	DocumentDigest string `json:"document_digest"`
	Revision       int64  `json:"revision"`
}
