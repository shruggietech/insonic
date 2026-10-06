// SPDX-License-Identifier: Apache-2.0
package artifact

import "github.com/shruggietech/insonic/internal/catalog"

// Receipt is the bounded runtime view of a publication. Complete multipart and
// lifecycle journals remain durable in the catalog and its portable snapshot.
type Receipt struct {
	ID               string `json:"id"`
	ArtifactID       string `json:"artifact_id"`
	LocationID       string `json:"location_id"`
	ProfileID        string `json:"profile_id"`
	ProfileRevision  int64  `json:"profile_revision"`
	Digest           string `json:"digest"`
	Size             int64  `json:"size"`
	Kind             string `json:"kind"`
	Key              string `json:"key"`
	Version          string `json:"version"`
	Verification     string `json:"verification"`
	State            string `json:"state"`
	Generation       int64  `json:"generation"`
	AvailableAt      int64  `json:"available_at"`
	AdmissionID      string `json:"admission_id"`
	JournalReceiptID string `json:"journal_receipt_id"`
	PartCount        int    `json:"part_count"`
	ReferenceCount   int    `json:"reference_count"`
	LeaseCount       int    `json:"lease_count"`
	EventCount       int    `json:"event_count"`
}

func ReceiptOf(p catalog.Publication) Receipt {
	return Receipt{ID: p.ID, ArtifactID: p.ArtifactID, LocationID: p.LocationID,
		ProfileID: p.ProfileID, ProfileRevision: p.ProfileRevision, Digest: p.Digest,
		Size: p.Size, Kind: p.Kind, Key: p.Key, Version: p.Version,
		Verification: p.Verification, State: p.State, Generation: p.Generation,
		AvailableAt: p.AvailableAt, AdmissionID: p.AdmissionID,
		JournalReceiptID: p.JournalReceiptID, PartCount: len(p.Parts),
		ReferenceCount: len(p.References), LeaseCount: len(p.Leases), EventCount: len(p.Events)}
}
