// SPDX-License-Identifier: Apache-2.0
package catalog

type RosterHeader struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
}
type RosterMember struct {
	ID          string `json:"id"`
	RecordingID string `json:"recording_id"`
	SpeakerID   string `json:"speaker_id"`
	Revision    int64  `json:"revision"`
}
type Roster struct {
	RecordingID string    `json:"recording_id"`
	Declared    bool      `json:"declared"`
	Revision    int64     `json:"revision"`
	Members     []Speaker `json:"members"`
}
type AdmissionElection struct {
	ReplaceAudio           bool     `json:"replace_audio"`
	ExpectedRosterRevision int64    `json:"expected_roster_revision"`
	RosterPolicy           string   `json:"roster_policy,omitempty"`
	SpeakerIDs             []string `json:"speaker_ids"`
}
