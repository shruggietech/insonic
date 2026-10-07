// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"encoding/json"
)

// LibraryEntry stores only current computed facts. Source artifacts are immutable.
// DurationUS is authoritative; nil means unavailable, zero is measured zero.
type LibraryEntry struct {
	ID                    string          `json:"id"`
	AssetID               string          `json:"asset_id"`
	Title                 string          `json:"title"`
	Class                 string          `json:"class"`
	Mode                  string          `json:"mode"`
	SourceLocator         string          `json:"source_locator"`
	Digest                string          `json:"digest"`
	Size                  int64           `json:"size"`
	OriginalPublicationID *string         `json:"original_publication_id"`
	SubtitlePublicationID *string         `json:"subtitle_publication_id"`
	DurationUS            *int64          `json:"duration_us"`
	Facts                 json.RawMessage `json:"facts"`
	Metadata              json.RawMessage `json:"metadata"`
	Dates                 json.RawMessage `json:"dates"`
	ReportPublicationIDs  json.RawMessage `json:"report_publication_ids"`
	Revision              int64           `json:"revision"`
}
type BaseModelInstall struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Digest         string          `json:"digest"`
	Manifest       json.RawMessage `json:"manifest"`
	PublicationIDs json.RawMessage `json:"publication_ids"`
	State          string          `json:"state"`
	Revision       int64           `json:"revision"`
}
type Work struct {
	JournalReceiptID string          `json:"journal_receipt_id"`
	ID               string          `json:"id"`
	Kind             string          `json:"kind"`
	Payload          json.RawMessage `json:"payload"`
	State            string          `json:"state"`
	Owner            string          `json:"owner"`
	Generation       int64           `json:"generation"`
	LeaseUntil       int64           `json:"lease_until"`
	Phase            string          `json:"phase"`
	Result           json.RawMessage `json:"result"`
	Error            string          `json:"error"`
}

// Cleanup retains only publication identities, never superseded report bytes.
type Cleanup struct {
	Revision int64  `json:"revision"`
	ID       string `json:"id"`
	EntryID  string `json:"entry_id"`
	State    string `json:"state"`
}
