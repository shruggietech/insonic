// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"reflect"
	"strings"
	"time"
)

const artifactDDL = `CREATE TABLE IF NOT EXISTS artifact_publication (workspace_id TEXT NOT NULL, id TEXT NOT NULL, artifact_id TEXT NOT NULL, data TEXT NOT NULL, PRIMARY KEY(workspace_id,id), FOREIGN KEY(workspace_id) REFERENCES workspace(id))`
const artifactIndexDDL = `CREATE INDEX IF NOT EXISTS artifact_publication_content ON artifact_publication(workspace_id,artifact_id)`

// Publication separates operational availability from immutable location evidence.
// Paths and credential values never belong in this portable journal.
type Publication struct {
	ID                  string                          `json:"id"`
	ArtifactID          string                          `json:"artifact_id"`
	LocationID          string                          `json:"location_id"`
	ProfileID           string                          `json:"profile_id"`
	ProfileRevision     int64                           `json:"profile_revision"`
	Digest              string                          `json:"digest"`
	Size                int64                           `json:"size"`
	Kind                string                          `json:"kind"`
	Key                 string                          `json:"key"`
	Version             string                          `json:"version"`
	Verification        string                          `json:"verification"`
	UploadID            string                          `json:"upload_id"`
	CompletionRequested bool                            `json:"completion_requested"`
	PartSize            int64                           `json:"part_size"`
	Parts               []UploadPart                    `json:"parts"`
	State               string                          `json:"state"`
	Owner               string                          `json:"owner"`
	Generation          int64                           `json:"generation"`
	LeaseUntil          int64                           `json:"lease_until"`
	AvailableAt         int64                           `json:"available_at"`
	AdmissionID         string                          `json:"admission_id"`
	JournalReceiptID    string                          `json:"journal_receipt_id"`
	References          map[string]bool                 `json:"references"`
	Leases              map[string]MaterializationLease `json:"leases"`
	Events              []string                        `json:"events"`
}
type UploadPart struct {
	Number int    `json:"number"`
	ETag   string `json:"etag"`
	Size   int64  `json:"size"`
}
type MaterializationLease struct {
	ID         string `json:"id"`
	Owner      string `json:"owner"`
	Generation int64  `json:"generation"`
	Until      int64  `json:"until"`
	Released   bool   `json:"released"`
}

func publicationIntent(p Publication) any {
	return struct {
		ID, Artifact, Location, Profile, Digest, Kind, Key string
		Revision, Size                                     int64
	}{p.ID, p.ArtifactID, p.LocationID, p.ProfileID, p.Digest, p.Kind, p.Key, p.ProfileRevision, p.Size}
}
func (p Publication) valid() bool {
	if !contracts.ValidID(p.ID) || !contracts.ValidID(p.ArtifactID) || !contracts.ValidID(p.LocationID) || !contracts.ValidID(p.ProfileID) || !contracts.ValidID(p.Owner) || p.Generation < 1 || p.ProfileRevision < 1 || !digestPattern.MatchString(p.Digest) || p.Size < 0 || p.Kind == "" || len(p.Kind) > 128 || len(p.Version) > 4096 || len(p.UploadID) > 4096 || !validObjectKey(p.Key) {
		return false
	}
	if !strings.Contains("|pending|available|missing|retiring|retired|aborted|", "|"+p.State+"|") || p.References == nil || p.Leases == nil {
		return false
	}
	if p.State != "pending" && p.State != "aborted" && (p.Verification != "sha256-readback" || !contracts.ValidID(p.AdmissionID)) {
		return false
	}
	for id, v := range p.References {
		if !contracts.ValidID(id) || !v {
			return false
		}
	}
	for id, l := range p.Leases {
		if id != l.ID || !contracts.ValidID(id) || !contracts.ValidID(l.Owner) || l.Generation < 1 {
			return false
		}
	}
	if p.PartSize < 0 || len(p.Parts) > 10000 || (len(p.Parts) > 0 && (p.UploadID == "" || p.PartSize < 5<<20 || p.PartSize > 5<<30)) || (p.UploadID != "" && (p.PartSize < 5<<20 || p.PartSize > 5<<30 || p.Size < 1 || (p.Size-1)/p.PartSize+1 > 10000)) {
		return false
	}
	for i, part := range p.Parts {
		offset := int64(i) * p.PartSize
		if offset >= p.Size || part.Number != i+1 || part.Size != min(p.PartSize, p.Size-offset) || part.ETag == "" || len(part.ETag) > 256 {
			return false
		}
	}
	return true
}
func validObjectKey(k string) bool {
	if len(k) == 0 || len(k) > 1024 || strings.ContainsAny(k, "\\\x00\r\n:") || strings.HasPrefix(k, "/") {
		return false
	}
	for _, v := range strings.Split(k, "/") {
		if v == "" || v == "." || v == ".." || strings.TrimSpace(v) != v {
			return false
		}
	}
	return true
}
func decodePublication(data string) (Publication, error) {
	var p Publication
	if strict([]byte(data), &p) != nil || !p.valid() {
		return p, contracts.Fail("invalid_request")
	}
	return p, nil
}
func (s *Store) publicationTx(ctx context.Context, tx *sql.Tx, id string) (Publication, error) {
	var data string
	e := s.row(ctx, tx, "SELECT data FROM artifact_publication WHERE workspace_id=? AND id=?", s.workspace, id).Scan(&data)
	if e != nil {
		return Publication{}, sanitize(e)
	}
	return decodePublication(data)
}
func (s *Store) Publication(ctx context.Context, id string) (Publication, error) {
	if !contracts.ValidID(id) {
		return Publication{}, contracts.Fail("invalid_request")
	}
	var data string
	e := s.db.QueryRowContext(ctx, s.query("SELECT data FROM artifact_publication WHERE workspace_id=? AND id=?"), s.workspace, id).Scan(&data)
	if e != nil {
		return Publication{}, sanitize(e)
	}
	return decodePublication(data)
}
func journalDigest(p Publication) (string, error) {
	p.LeaseUntil = 0
	p.Leases = copyLeases(p.Leases)
	for id, l := range p.Leases {
		l.Until = 0
		p.Leases[id] = l
	}
	return intent(p)
}
func copyLeases(in map[string]MaterializationLease) map[string]MaterializationLease {
	out := map[string]MaterializationLease{}
	for id, l := range in {
		out[id] = l
	}
	return out
}
func (s *Store) putPublication(ctx context.Context, tx *sql.Tx, p *Publication, rev int64) error {
	p.JournalReceiptID = contracts.ID()
	if !p.valid() {
		return contracts.Fail("invalid_request")
	}
	data, e := json.Marshal(p)
	if e != nil {
		return e
	}
	_, e = s.exec(ctx, tx, "INSERT INTO artifact_publication(workspace_id,id,artifact_id,data) VALUES(?,?,?,?) ON CONFLICT(workspace_id,id) DO UPDATE SET data=excluded.data", s.workspace, p.ID, p.ArtifactID, string(data))
	if e != nil {
		return e
	}
	digest, e := journalDigest(*p)
	if e != nil {
		return e
	}
	_, e = s.accept(ctx, tx, p.JournalReceiptID, digest, rev, map[string]any{"publication": p.ID, "state": p.State, "generation": p.Generation})
	return e
}
func validTTL(ttl time.Duration) bool { return ttl > 0 && ttl <= 24*time.Hour }
func (s *Store) BeginPublication(ctx context.Context, p Publication, ttl time.Duration) (Publication, error) {
	if !validTTL(ttl) || !contracts.ValidID(p.Owner) {
		return p, contracts.Fail("invalid_request")
	}
	var out Publication
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		var data string
		e := s.row(ctx, tx, "SELECT data FROM artifact_publication WHERE workspace_id=? AND id=?", s.workspace, p.ID).Scan(&data)
		now, e2 := s.now(ctx, tx)
		if e2 != nil {
			return e2
		}
		if e == nil {
			old, e := decodePublication(data)
			if e != nil {
				return e
			}
			a, _ := intent(publicationIntent(old))
			b, _ := intent(publicationIntent(p))
			if a != b {
				return contracts.Fail("conflict")
			}
			if old.State != "pending" {
				out = old
				return nil
			}
			if old.LeaseUntil > now && old.Owner != p.Owner {
				return contracts.Fail("conflict")
			}
			if old.Generation == math.MaxInt64 {
				return contracts.Fail("conflict")
			}
			owner := p.Owner
			p = old
			p.Owner = owner
			p.Generation++
		} else if e == sql.ErrNoRows {
			p.State = "pending"
			p.Generation = 1
			p.References = map[string]bool{}
			p.Leases = map[string]MaterializationLease{}
			p.Events = []string{"pending"}
			var n int
			if e = s.row(ctx, tx, "SELECT count(*) FROM profile_revision WHERE workspace_id=? AND id=? AND revision=? AND role='storage'", s.workspace, p.ProfileID, p.ProfileRevision).Scan(&n); e != nil {
				return e
			}
			if n != 1 {
				return contracts.Fail("invalid_request")
			}
		} else {
			return e
		}
		p.LeaseUntil = now + int64(ttl)
		e = s.putPublication(ctx, tx, &p, rev)
		out = p
		return e
	})
	return out, e
}
func sameClaim(a, b Publication, now int64) bool {
	return a.ID == b.ID && a.Owner == b.Owner && a.Generation == b.Generation && a.LeaseUntil > now
}
func (s *Store) SavePublication(ctx context.Context, p Publication, ttl time.Duration) (Publication, error) {
	if !validTTL(ttl) {
		return p, contracts.Fail("invalid_request")
	}
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		old, e := s.publicationTx(ctx, tx, p.ID)
		if e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		if old.State != "pending" || p.State != "pending" || !sameClaim(old, p, now) {
			return contracts.Fail("conflict")
		}
		a, _ := intent(publicationIntent(old))
		b, _ := intent(publicationIntent(p))
		if a != b {
			return contracts.Fail("conflict")
		}
		p.LeaseUntil = now + int64(ttl)
		p.References = old.References
		p.Leases = old.Leases
		p.Events = old.Events
		return s.putPublication(ctx, tx, &p, rev)
	})
	return p, e
}
func (s *Store) AdmitPublication(ctx context.Context, p Publication) (Publication, error) {
	var out Publication
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		old, e := s.publicationTx(ctx, tx, p.ID)
		if e != nil {
			return e
		}
		if old.AdmissionID != "" {
			a, _ := intent(publicationIntent(old))
			b, _ := intent(publicationIntent(p))
			if a != b {
				return contracts.Fail("conflict")
			}
			out = old
			return nil
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		if old.State != "pending" || !sameClaim(old, p, now) || p.Verification != "sha256-readback" {
			return contracts.Fail("conflict")
		}
		a, _ := intent(publicationIntent(old))
		b, _ := intent(publicationIntent(p))
		if a != b {
			return contracts.Fail("conflict")
		}
		p.State = "available"
		p.AvailableAt = now
		p.AdmissionID = operationID("artifact-admit", p.ID)
		p.JournalReceiptID = p.AdmissionID
		p.LeaseUntil = 0
		p.Events = append(old.Events, "available")
		p.References = old.References
		p.Leases = old.Leases
		var version *string
		if p.Version != "" {
			version = &p.Version
		}
		if e = s.insertRecords(ctx, tx, Records{Artifacts: []Artifact{{ID: p.ArtifactID, Digest: p.Digest, Size: p.Size, Kind: p.Kind}}, Locations: []ArtifactLocation{{ID: p.LocationID, ArtifactID: p.ArtifactID, ProfileID: p.ProfileID, ProfileRevision: p.ProfileRevision, Key: p.Key, Version: version, State: "available"}}}); e != nil {
			return e
		}
		data, _ := json.Marshal(p)
		if _, e = s.exec(ctx, tx, "UPDATE artifact_publication SET data=? WHERE workspace_id=? AND id=?", string(data), s.workspace, p.ID); e != nil {
			return e
		}
		digest, _ := intent(publicationIntent(p))
		_, e = s.accept(ctx, tx, p.AdmissionID, digest, rev, p)
		out = p
		return e
	})
	return out, e
}
func (s *Store) AbortPublication(ctx context.Context, p Publication) (Publication, error) {
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		old, e := s.publicationTx(ctx, tx, p.ID)
		if e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		if old.State == "aborted" {
			p = old
			return nil
		}
		if old.State != "pending" || !sameClaim(old, p, now) {
			return contracts.Fail("conflict")
		}
		p = old
		p.State = "aborted"
		p.LeaseUntil = 0
		p.Events = append(p.Events, "aborted")
		return s.putPublication(ctx, tx, &p, rev)
	})
	return p, e
}
func (s *Store) ArtifactReference(ctx context.Context, id, ref string, release bool) error {
	if !contracts.ValidID(ref) {
		return contracts.Fail("invalid_request")
	}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		p, e := s.publicationTx(ctx, tx, id)
		if e != nil {
			return e
		}
		if !release && p.State != "available" {
			return contracts.Fail("conflict")
		}
		if release {
			delete(p.References, ref)
		} else {
			p.References[ref] = true
		}
		return s.putPublication(ctx, tx, &p, rev)
	})
}
func (s *Store) ArtifactLease(ctx context.Context, id, lease, owner string, ttl time.Duration, release bool) (MaterializationLease, error) {
	return s.artifactLease(ctx, id, lease, owner, ttl, release, true)
}
func (s *Store) RenewArtifactLease(ctx context.Context, id, lease, owner string, ttl time.Duration) (MaterializationLease, error) {
	return s.artifactLease(ctx, id, lease, owner, ttl, false, false)
}
func (s *Store) artifactLease(ctx context.Context, id, lease, owner string, ttl time.Duration, release, create bool) (MaterializationLease, error) {
	var out MaterializationLease
	if !contracts.ValidID(lease) || !contracts.ValidID(owner) || (!release && !validTTL(ttl)) {
		return out, contracts.Fail("invalid_request")
	}
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		p, e := s.publicationTx(ctx, tx, id)
		if e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		old, exists := p.Leases[lease]
		if !exists && !create {
			return contracts.Fail("conflict")
		}
		if !create && old.Until <= now {
			return contracts.Fail("conflict")
		}
		if exists && old.Owner != owner {
			return contracts.Fail("conflict")
		}
		if release {
			if exists {
				old.Released = true
				old.Until = 0
				p.Leases[lease] = old
			}
			out = old
		} else {
			if exists && old.Released {
				return contracts.Fail("conflict")
			}
			if p.State != "available" {
				return contracts.Fail("conflict")
			}
			generation := int64(1)
			if exists {
				if old.Generation == math.MaxInt64 {
					return contracts.Fail("conflict")
				}
				generation = old.Generation
				if old.Until <= now {
					generation++
				}
			}
			out = MaterializationLease{ID: lease, Owner: owner, Generation: generation, Until: now + int64(ttl)}
			p.Leases[lease] = out
		}
		return s.putPublication(ctx, tx, &p, rev)
	})
	return out, e
}
func (s *Store) ExpireMaterializations(ctx context.Context, id string) ([]string, error) {
	out := []string{}
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		p, e := s.publicationTx(ctx, tx, id)
		if e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		for id, l := range p.Leases {
			if l.Until <= now {
				out = append(out, id)
				l.Released = true
				l.Until = 0
				p.Leases[id] = l
			}
		}
		if len(out) == 0 {
			return nil
		}
		return s.putPublication(ctx, tx, &p, rev)
	})
	return out, e
}
func (s *Store) ForgetMaterialization(ctx context.Context, id, lease string) error {
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		p, e := s.publicationTx(ctx, tx, id)
		if e != nil {
			return e
		}
		l, found := p.Leases[lease]
		if !found {
			return nil
		}
		if !l.Released {
			return contracts.Fail("conflict")
		}
		delete(p.Leases, lease)
		return s.putPublication(ctx, tx, &p, rev)
	})
}
func (s *Store) structuralReferences(ctx context.Context, tx *sql.Tx, artifact string) (bool, error) {
	for _, d := range domains {
		for _, fk := range d.foreign {
			v := strings.Split(fk, ":")
			if v[1] != "artifact" || d.field == "Locations" {
				continue
			}
			var n int
			e := s.row(ctx, tx, "SELECT count(*) FROM "+d.name+" WHERE workspace_id=? AND "+v[0]+"=?", s.workspace, artifact).Scan(&n)
			if e != nil {
				return false, e
			}
			if n > 0 {
				return true, nil
			}
		}
	}
	return false, nil
}
func (s *Store) ClaimRetirement(ctx context.Context, id, owner string, grace, ttl time.Duration) (Publication, error) {
	var out Publication
	if !contracts.ValidID(owner) || grace < 0 || !validTTL(ttl) {
		return out, contracts.Fail("invalid_request")
	}
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		p, e := s.publicationTx(ctx, tx, id)
		if e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		if p.State == "retired" {
			out = p
			return nil
		}
		if p.State == "retiring" && p.LeaseUntil > now && p.Owner != owner {
			return contracts.Fail("conflict")
		}
		if p.State != "available" && p.State != "missing" && p.State != "retiring" {
			return contracts.Fail("conflict")
		}
		if now-p.AvailableAt < int64(grace) || len(p.References) > 0 {
			return contracts.Fail("conflict")
		}
		for _, l := range p.Leases {
			if l.Until > now {
				return contracts.Fail("conflict")
			}
		}
		retained, e := s.structuralReferences(ctx, tx, p.ArtifactID)
		if e != nil {
			return e
		}
		if retained || p.Generation == math.MaxInt64 {
			return contracts.Fail("conflict")
		}
		p.Owner = owner
		p.Generation++
		p.State = "retiring"
		p.LeaseUntil = now + int64(ttl)
		p.Events = append(p.Events, "retiring")
		e = s.putPublication(ctx, tx, &p, rev)
		out = p
		return e
	})
	return out, e
}
func (s *Store) FinishRetirement(ctx context.Context, p Publication) (Publication, error) {
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		old, e := s.publicationTx(ctx, tx, p.ID)
		if e != nil {
			return e
		}
		if old.State == "retired" {
			p = old
			return nil
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		if old.State != "retiring" || !sameClaim(old, p, now) {
			return contracts.Fail("conflict")
		}
		p = old
		p.State = "retired"
		p.LeaseUntil = 0
		p.Events = append(p.Events, "retired")
		if _, e = s.exec(ctx, tx, "UPDATE artifact_location SET state='retired' WHERE workspace_id=? AND id=? AND artifact_id=?", s.workspace, p.LocationID, p.ArtifactID); e != nil {
			return e
		}
		return s.putPublication(ctx, tx, &p, rev)
	})
	return p, e
}

// Generic domain admission cannot bypass a claimed retirement barrier.
func (s *Store) guardArtifactReferences(ctx context.Context, tx *sql.Tx, records Records) error {
	v := reflect.ValueOf(records)
	for _, d := range domains {
		items := v.FieldByName(d.field)
		for _, fk := range d.foreign {
			f := strings.Split(fk, ":")
			if f[1] != "artifact" || d.field == "Locations" {
				continue
			}
			for _, c := range columns(d.typ()) {
				if c.name != f[0] {
					continue
				}
				for i := 0; i < items.Len(); i++ {
					id := fieldValue(items.Index(i), c.path)
					if id == nil {
						continue
					}
					rows, e := tx.QueryContext(ctx, s.query("SELECT data FROM artifact_publication WHERE workspace_id=? AND artifact_id=?"), s.workspace, id)
					if e != nil {
						return e
					}
					blocked := false
					for rows.Next() {
						var data string
						if e = rows.Scan(&data); e != nil {
							break
						}
						p, err := decodePublication(data)
						if err != nil {
							e = err
							break
						}
						if p.ArtifactID == id && p.State != "available" {
							blocked = true
						}
					}
					err := rows.Err()
					rows.Close()
					if e != nil {
						return e
					}
					if err != nil {
						return err
					}
					if blocked {
						return contracts.Fail("conflict")
					}
				}
			}
		}
	}
	return nil
}

func (s *Store) validateArtifactState(ctx context.Context, tx *sql.Tx, expire bool) error {
	rows, e := tx.QueryContext(ctx, s.query("SELECT id,artifact_id,data FROM artifact_publication WHERE workspace_id=?"), s.workspace)
	if e != nil {
		return e
	}
	var all []Publication
	for rows.Next() {
		var id, artifactID, data string
		if e = rows.Scan(&id, &artifactID, &data); e != nil {
			break
		}
		p, err := decodePublication(data)
		if err != nil || id != p.ID || artifactID != p.ArtifactID {
			e = contracts.Fail("invalid_request")
			break
		}
		all = append(all, p)
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if err != nil {
		return err
	}
	// Portable SQL has no shared JSON operators. Inspect receipt envelopes once
	// and bind each journal to the newest receipt for that publication.
	receipts, e := tx.QueryContext(ctx, s.query("SELECT id,revision,result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	latest := map[string]string{}
	for receipts.Next() {
		var id, result string
		var revision int64
		if e = receipts.Scan(&id, &revision, &result); e != nil {
			break
		}
		var envelope struct {
			Publication string `json:"publication"`
			ID          string `json:"id"`
			ArtifactID  string `json:"artifact_id"`
		}
		if json.Unmarshal([]byte(result), &envelope) != nil {
			continue
		}
		if envelope.Publication != "" {
			latest[envelope.Publication] = id
		} else if envelope.ArtifactID != "" {
			latest[envelope.ID] = id
		}
	}
	err = receipts.Err()
	receipts.Close()
	if e != nil {
		return e
	}
	if err != nil {
		return err
	}
	for _, p := range all {
		var n int
		if e = s.row(ctx, tx, "SELECT count(*) FROM profile_revision WHERE workspace_id=? AND id=? AND revision=? AND role='storage'", s.workspace, p.ProfileID, p.ProfileRevision).Scan(&n); e != nil {
			return e
		}
		if n != 1 {
			return contracts.Fail("invalid_request")
		}
		if !contracts.ValidID(p.JournalReceiptID) || latest[p.ID] != p.JournalReceiptID {
			return contracts.Fail("invalid_request")
		}
		var latestDigest string
		if e = s.row(ctx, tx, "SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, p.JournalReceiptID).Scan(&latestDigest); e != nil {
			return contracts.Fail("invalid_request")
		}
		var expected string
		if p.JournalReceiptID == p.AdmissionID {
			expected, _ = intent(publicationIntent(p))
			if p.State != "available" || len(p.References) != 0 || len(p.Leases) != 0 {
				return contracts.Fail("invalid_request")
			}
		} else {
			expected, _ = journalDigest(p)
		}
		if latestDigest != expected {
			return contracts.Fail("invalid_request")
		}
		if p.AdmissionID != "" {
			var digest, result string
			if e = s.row(ctx, tx, "SELECT digest,result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, p.AdmissionID).Scan(&digest, &result); e != nil {
				return contracts.Fail("invalid_request")
			}
			var accepted Publication
			if strict([]byte(result), &accepted) != nil || !accepted.valid() || accepted.State != "available" {
				return contracts.Fail("invalid_request")
			}
			want, _ := intent(publicationIntent(p))
			acceptedIntent, _ := intent(publicationIntent(accepted))
			if want != digest || acceptedIntent != digest || accepted.AdmissionID != p.AdmissionID || accepted.Verification != p.Verification || accepted.Version != p.Version || accepted.AvailableAt != p.AvailableAt {
				return contracts.Fail("invalid_request")
			}
			locationState := "available"
			if p.State == "retired" || p.State == "missing" {
				locationState = p.State
			}
			if e = s.row(ctx, tx, "SELECT count(*) FROM artifact a JOIN artifact_location l ON l.workspace_id=a.workspace_id AND l.artifact_id=a.id WHERE a.workspace_id=? AND a.id=? AND a.digest=? AND a.size=? AND l.id=? AND l.key=? AND l.profile_id=? AND l.profile_revision=? AND coalesce(l.version,'')=? AND l.state=?", s.workspace, p.ArtifactID, p.Digest, p.Size, p.LocationID, p.Key, p.ProfileID, p.ProfileRevision, p.Version, locationState).Scan(&n); e != nil {
				return e
			}
			if n != 1 {
				return contracts.Fail("invalid_request")
			}
		}
		if expire {
			p.LeaseUntil = 0
			for id, l := range p.Leases {
				l.Until = 0
				p.Leases[id] = l
			}
			data, _ := json.Marshal(p)
			if _, e = s.exec(ctx, tx, "UPDATE artifact_publication SET data=? WHERE workspace_id=? AND id=?", string(data), s.workspace, p.ID); e != nil {
				return e
			}
		}
	}
	return nil
}
