// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

type transferContextKey struct{}
type portableActivationKey struct{}
type transferAuthority struct {
	Lease          Lease
	administrative bool
	maintenance    bool
}

// WithTransfer carries expiring transfer authority, never a global bypass.
func WithTransfer(ctx context.Context, lease Lease) context.Context {
	return context.WithValue(ctx, transferContextKey{}, transferAuthority{Lease: lease})
}

func WithoutTransfer(ctx context.Context) context.Context {
	return context.WithValue(ctx, transferContextKey{}, nil)
}

func (s *Store) transferBarrier(ctx context.Context, tx *sql.Tx) error {
	authority, has := ctx.Value(transferContextKey{}).(transferAuthority)
	pending, e := s.pendingPortableActivation(ctx, tx)
	if e != nil {
		return e
	}
	if pending != "" && !authority.administrative && !authority.maintenance && ctx.Value(portableActivationKey{}) != pending {
		return contracts.Fail("conflict")
	}
	var owner string
	var generation, until int64
	e = s.row(ctx, tx, "SELECT owner_id,generation,lease_until FROM workspace_lease WHERE workspace_id=? AND role='portable-transfer'", s.workspace).Scan(&owner, &generation, &until)
	if e == sql.ErrNoRows {
		if has && !authority.administrative {
			return contracts.Fail("conflict")
		}
		return nil
	}
	if e != nil {
		return e
	}
	if authority.administrative {
		return nil
	}
	now, e := s.now(ctx, tx)
	if e != nil {
		return e
	}
	if has {
		if authority.Lease.Role != "portable-transfer" || authority.Lease.OwnerID != owner || authority.Lease.Generation != generation || until <= now {
			return contracts.Fail("conflict")
		}
	} else if until > now {
		return contracts.Fail("conflict")
	}
	return nil
}

// BeginTransfer atomically excludes live processing and freezes catalog writes.
// It does not revive stale generations after lease expiry.
func (s *Store) BeginTransfer(ctx context.Context, owner string, ttl time.Duration) (Lease, error) {
	if !contracts.ValidID(owner) || !validTTL(ttl) {
		return Lease{}, contracts.Fail("invalid_request")
	}
	var lease Lease
	ctx = context.WithValue(ctx, transferContextKey{}, transferAuthority{administrative: true})
	e := s.write(ctx, func(tx *sql.Tx, _ int64) error {
		pending, e := s.pendingPortableActivation(ctx, tx)
		if e != nil {
			return e
		}
		if pending != "" && ctx.Value(portableActivationKey{}) != pending {
			return contracts.Fail("conflict")
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		var current string
		var generation, until int64
		e = s.row(ctx, tx, "SELECT owner_id,generation,lease_until FROM workspace_lease WHERE workspace_id=? AND role='portable-transfer'", s.workspace).Scan(&current, &generation, &until)
		if e != nil && e != sql.ErrNoRows {
			return e
		}
		if until > now {
			return contracts.Fail("conflict")
		}
		if generation == math.MaxInt64 {
			return contracts.Fail("conflict")
		}
		var active int
		if e = s.row(ctx, tx, "SELECT (SELECT count(*) FROM work_operation WHERE workspace_id=? AND state='running' AND lease_until>?)+(SELECT count(*) FROM job_attempt WHERE workspace_id=? AND state='running' AND lease_until>?)+(SELECT count(*) FROM graph_target WHERE workspace_id=? AND lease_until>?)", s.workspace, now, s.workspace, now, s.workspace, now).Scan(&active); e != nil {
			return e
		}
		if active != 0 {
			return contracts.Fail("conflict")
		}
		lease = Lease{Role: "portable-transfer", OwnerID: owner, Generation: generation + 1, Until: now + int64(ttl)}
		_, e = s.exec(ctx, tx, "INSERT INTO workspace_lease(workspace_id,role,owner_id,generation,lease_until) VALUES(?,'portable-transfer',?,?,?) ON CONFLICT(workspace_id,role) DO UPDATE SET owner_id=excluded.owner_id,generation=excluded.generation,lease_until=excluded.lease_until", s.workspace, owner, lease.Generation, lease.Until)
		return e
	})
	return lease, e
}
func (s *Store) RenewTransfer(ctx context.Context, lease Lease, ttl time.Duration) error {
	if !validTTL(ttl) {
		return contracts.Fail("invalid_request")
	}
	ctx = context.WithValue(ctx, transferContextKey{}, transferAuthority{Lease: lease, maintenance: true})
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		_, e = s.exec(ctx, tx, "UPDATE workspace_lease SET lease_until=? WHERE workspace_id=? AND role='portable-transfer'", now+int64(ttl), s.workspace)
		return e
	})
}
func (s *Store) EndTransfer(ctx context.Context, lease Lease) error {
	ctx = context.WithValue(ctx, transferContextKey{}, transferAuthority{Lease: lease, maintenance: true})
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		_, e := s.exec(ctx, tx, "UPDATE workspace_lease SET lease_until=0 WHERE workspace_id=? AND role='portable-transfer'", s.workspace)
		return e
	})
}

// Empty proves the selected workspace has no accepted authority.
func (s *Store) Empty(ctx context.Context) error {
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if rev != 0 {
			return contracts.Fail("conflict")
		}
		for _, d := range domains {
			var n int
			if e := s.row(ctx, tx, "SELECT count(*) FROM "+d.name+" WHERE workspace_id=?", s.workspace).Scan(&n); e != nil {
				return e
			}
			if n != 0 {
				return contracts.Fail("conflict")
			}
		}
		return nil
	})
}

// ValidateSnapshot exercises the same relational and admission constraints as
// real restore against a private disposable local catalog before any target I/O.
func ValidateSnapshot(ctx context.Context, snap Snapshot) error {
	ctx = context.WithValue(ctx, transferContextKey{}, nil)
	dir, e := os.MkdirTemp("", "insonic-backup-validation-*")
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer os.RemoveAll(dir)
	s, e := OpenSQLite(ctx, filepath.Join(dir, "catalog.sqlite"), snap.WorkspaceID)
	if e != nil {
		return e
	}
	defer s.Close()
	return s.Restore(ctx, snap)
}

// BackupPublications selects live current authority and durable model bytes.
// Historical receipt identities remain in the logical snapshot; obsolete bytes
// and transient preparation never acquire a hidden backup copy.
func BackupPublications(snap Snapshot) ([]Publication, error) {
	byID := map[string]Publication{}
	for _, table := range snap.State {
		if table.Name != "artifact_publication" {
			continue
		}
		for _, row := range table.Rows {
			if len(row) != 3 {
				return nil, contracts.Fail("invalid_request")
			}
			var raw string
			if strict(row[2], &raw) != nil {
				return nil, contracts.Fail("invalid_request")
			}
			p, e := decodePublication(raw)
			if e != nil {
				return nil, e
			}
			if _, ok := byID[p.ID]; ok {
				return nil, contracts.Fail("invalid_request")
			}
			byID[p.ID] = p
		}
	}
	selected := map[string]bool{}
	requiredArtifacts := map[string]bool{}
	excluded := map[string]bool{}
	add := func(raw json.RawMessage) error {
		var ids []string
		if strict(raw, &ids) != nil {
			return contracts.Fail("invalid_request")
		}
		for _, id := range ids {
			selected[id] = true
		}
		return nil
	}
	for _, c := range snap.Records.Cleanups {
		excluded[c.ID] = true
	}
	for _, entry := range snap.Records.Library {
		if entry.Mode != "copy" || entry.SourceLocator != "" {
			return nil, contracts.Fail("unsupported_capability")
		}
		if entry.OriginalPublicationID != nil {
			selected[*entry.OriginalPublicationID] = true
		}
		if e := add(entry.ReportPublicationIDs); e != nil {
			return nil, e
		}
	}
	for _, r := range snap.Records.Recordings {
		if r.MappedAudioPublicationID != nil {
			selected[*r.MappedAudioPublicationID] = true
		}
	}
	for _, m := range snap.Records.BaseModels {
		if m.State == "available" {
			if e := add(m.PublicationIDs); e != nil {
				return nil, e
			}
		}
	}
	for _, a := range snap.Records.ModelArtifacts {
		requiredArtifacts[a.ArtifactID] = true
	}
	for _, d := range snap.Records.Datasets {
		if d.State == "current" && d.ManifestArtifactID != nil {
			requiredArtifacts[*d.ManifestArtifactID] = true
		}
	}
	for _, v := range snap.Records.Versions {
		if v.ManifestArtifactID != nil {
			requiredArtifacts[*v.ManifestArtifactID] = true
		}
	}
	for _, run := range snap.Records.Runs {
		if run.State == "current" && run.PreparationArtifactID != nil {
			requiredArtifacts[*run.PreparationArtifactID] = true
		}
	}
	for _, c := range snap.Records.SpeakerCheckpoints {
		selected[c.PublicationID] = true
	}
	for _, p := range byID {
		if requiredArtifacts[p.ArtifactID] {
			selected[p.ID] = true
		}
		if p.State == "available" && !excluded[p.ID] && p.Kind != "mapped-audio" && p.Kind != "speaker-clip" && p.Kind != "derived-manifest" && p.Kind != "subtitle-source" && p.Kind != "inference-preparation" {
			selected[p.ID] = true
		}
	}
	out := []Publication{}
	for id := range selected {
		p, ok := byID[id]
		if !ok || p.State != "available" {
			return nil, contracts.Fail("conflict")
		}
		if excluded[id] && !requiredArtifacts[p.ArtifactID] {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// PinBackup binds all exact manifest publications in one catalog transaction.
func (s *Store) PinBackup(ctx context.Context, id string, publications []Publication) error {
	if !contracts.ValidID(id) {
		return contracts.Fail("invalid_request")
	}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		identities := []any{}
		for _, p := range publications {
			identities = append(identities, []any{publicationIntent(p), p.Version})
		}
		d, e := intent([]any{"backup-pin", id, identities})
		if e != nil {
			return e
		}
		op := operationID("backup-pin", id)
		_, replayed, e := s.replay(ctx, tx, op, d)
		if e != nil {
			return e
		}
		var released int
		if e = s.row(ctx, tx, "SELECT count(*) FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, operationID("backup-release", id)).Scan(&released); e != nil {
			return e
		}
		if released != 0 {
			return contracts.Fail("conflict")
		}
		if !replayed {
			rows, e := tx.QueryContext(ctx, s.query("SELECT data FROM artifact_publication WHERE workspace_id=?"), s.workspace)
			if e != nil {
				return e
			}
			for rows.Next() {
				var raw string
				if e = rows.Scan(&raw); e != nil {
					rows.Close()
					return e
				}
				p, e := decodePublication(raw)
				if e != nil {
					rows.Close()
					return e
				}
				if p.References[id] {
					rows.Close()
					return contracts.Fail("conflict")
				}
			}
			e = rows.Err()
			rows.Close()
			if e != nil {
				return e
			}
		}
		for _, want := range publications {
			p, e := s.publicationTx(ctx, tx, want.ID)
			if e != nil {
				return e
			}
			a, _ := intent(publicationIntent(p))
			b, _ := intent(publicationIntent(want))
			if p.State != "available" || a != b || p.Version != want.Version {
				return contracts.Fail("conflict")
			}
			if p.References[id] {
				if !replayed {
					return contracts.Fail("conflict")
				}
				continue
			}
			p.References[id] = true
			if e = s.putPublication(ctx, tx, &p, rev); e != nil {
				return e
			}
			rev++
		}
		if replayed {
			return nil
		}
		_, e = s.accept(ctx, tx, op, d, rev, map[string]any{"backup_reference_id": id, "backup_state": "pinned"})
		return e
	})
}

func (s *Store) backupReferenceReserved(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	var raw string
	e := s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, operationID("backup-pin", id)).Scan(&raw)
	if e == sql.ErrNoRows {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var proof struct {
		ID    string `json:"backup_reference_id"`
		State string `json:"backup_state"`
	}
	if json.Unmarshal([]byte(raw), &proof) != nil {
		return false, contracts.Fail("invalid_request")
	}
	return proof.ID == id && proof.State == "pinned", nil
}

// ReleaseBackupID reconciles retained pins after interrupted bundle creation,
// even when no completed manifest exists. Only a recorded backup lifecycle may
// authorize this operation; ordinary explicit references are never guessed.
func (s *Store) ReleaseBackupID(ctx context.Context, id string) error {
	if !contracts.ValidID(id) {
		return contracts.Fail("invalid_request")
	}
	var result string
	e := s.db.QueryRowContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?"), s.workspace, operationID("backup-pin", id)).Scan(&result)
	if e != nil {
		return sanitize(e)
	}
	var lifecycle struct {
		ID string `json:"backup_reference_id"`
	}
	if json.Unmarshal([]byte(result), &lifecycle) != nil || lifecycle.ID != id {
		return contracts.Fail("invalid_request")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		return e
	}
	publications := []Publication{}
	for _, table := range snap.State {
		if table.Name != "artifact_publication" {
			continue
		}
		for _, row := range table.Rows {
			var raw string
			if strict(row[2], &raw) != nil {
				return contracts.Fail("invalid_request")
			}
			p, e := decodePublication(raw)
			if e != nil {
				return e
			}
			if p.References[id] {
				publications = append(publications, p)
			}
		}
	}
	return s.ReleaseBackup(ctx, id, publications)
}
func (s *Store) ReleaseBackup(ctx context.Context, id string, publications []Publication) error {
	if !contracts.ValidID(id) {
		return contracts.Fail("invalid_request")
	}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		var raw string
		if e := s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, operationID("backup-pin", id)).Scan(&raw); e != nil {
			return e
		}
		var proof struct {
			ID    string `json:"backup_reference_id"`
			State string `json:"backup_state"`
		}
		if json.Unmarshal([]byte(raw), &proof) != nil || proof.ID != id || proof.State != "pinned" {
			return contracts.Fail("invalid_request")
		}
		provided := map[string]bool{}
		for _, p := range publications {
			if !contracts.ValidID(p.ID) || provided[p.ID] {
				return contracts.Fail("invalid_request")
			}
			provided[p.ID] = true
		}
		rows, e := tx.QueryContext(ctx, s.query("SELECT data FROM artifact_publication WHERE workspace_id=?"), s.workspace)
		if e != nil {
			return e
		}
		for rows.Next() {
			var raw string
			if e = rows.Scan(&raw); e != nil {
				rows.Close()
				return e
			}
			p, e := decodePublication(raw)
			if e != nil {
				rows.Close()
				return e
			}
			if p.References[id] && !provided[p.ID] {
				rows.Close()
				return contracts.Fail("conflict")
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, want := range publications {
			p, e := s.publicationTx(ctx, tx, want.ID)
			if e != nil {
				return e
			}
			if p.Digest != want.Digest || p.Key != want.Key || p.Version != want.Version {
				return contracts.Fail("conflict")
			}
			if !p.References[id] {
				continue
			}
			delete(p.References, id)
			if e = s.putPublication(ctx, tx, &p, rev); e != nil {
				return e
			}
			rev++
		}
		d, e := intent([]any{"backup-release", id})
		if e != nil {
			return e
		}
		op := operationID("backup-release", id)
		if _, ok, e := s.replay(ctx, tx, op, d); e != nil {
			return e
		} else if ok {
			return nil
		}
		_, e = s.accept(ctx, tx, op, d, rev, map[string]any{"backup_reference_id": id, "backup_state": "released"})
		return e
	})
}

// RestorePortable validates untouched source receipts before admitting verified
// destination object authority. IDs used by recordings and models stay stable.
func (s *Store) RestorePortable(ctx context.Context, snap Snapshot, w *workspace.Workspace, relocations []Publication, referenceOnly bool) error {
	return s.restorePortable(ctx, snap, w, relocations, referenceOnly, nil)
}

// RestorePortableFenced keeps the passive imported scope unwritable until the
// configuration activation marker has been durably installed by the caller.
func (s *Store) RestorePortableFenced(ctx context.Context, snap Snapshot, w *workspace.Workspace, relocations []Publication) (Lease, error) {
	var lease Lease
	e := s.restorePortable(ctx, snap, w, relocations, false, &lease)
	return lease, e
}

func (s *Store) restorePortable(ctx context.Context, snap Snapshot, w *workspace.Workspace, relocations []Publication, referenceOnly bool, activation *Lease) error {
	return s.restoreSnapshot(ctx, snap, func(tx *sql.Tx) error {
		backupReferences := map[string]bool{}
		for _, table := range snap.State {
			if table.Name != "operation_receipt" {
				continue
			}
			for _, row := range table.Rows {
				if len(row) != 4 {
					return contracts.Fail("invalid_request")
				}
				var receiptID, raw string
				if strict(row[0], &receiptID) != nil || strict(row[3], &raw) != nil {
					return contracts.Fail("invalid_request")
				}
				var result struct {
					ID    string `json:"backup_reference_id"`
					State string `json:"backup_state"`
				}
				if json.Unmarshal([]byte(raw), &result) != nil {
					return contracts.Fail("invalid_request")
				}
				if result.ID != "" {
					if !contracts.ValidID(result.ID) {
						continue
					}
					pinned := receiptID == operationID("backup-pin", result.ID)
					released := receiptID == operationID("backup-release", result.ID)
					if !pinned && !released {
						continue
					}
					if pinned && result.State != "pinned" || released && result.State != "released" {
						return contracts.Fail("invalid_request")
					}
					backupReferences[result.ID] = true
				}
			}
		}
		var rev int64
		if e := s.row(ctx, tx, "SELECT revision FROM workspace WHERE id=?", s.workspace).Scan(&rev); e != nil {
			return e
		}
		for _, selected := range []struct {
			role    string
			profile workspace.Profile
		}{{"storage", w.Config.Profiles.Storage}, {"catalog", w.Config.Profiles.Catalog}, {"graph", w.Config.Profiles.Graph}} {
			p := selected.profile
			raw, e := json.Marshal(p.Configuration)
			if e != nil {
				return e
			}
			raw, e = canonical(raw)
			if e != nil {
				return e
			}
			var credential *string
			if id, ok := p.Configuration["credential_id"].(string); ok {
				credential = &id
			}
			record := Profile{ID: p.ID, Revision: p.Revision, Role: selected.role, Adapter: p.Adapter, Version: p.Version, Configuration: raw, CredentialID: credential}
			if p.ExpectedBackendVersion != "" {
				record.ExpectedBackendVersion = &p.ExpectedBackendVersion
			}
			if e = validateRecord(record); e != nil {
				return e
			}
			var existing string
			e = s.row(ctx, tx, "SELECT configuration FROM profile_revision WHERE workspace_id=? AND id=? AND revision=?", s.workspace, p.ID, p.Revision).Scan(&existing)
			if e == nil {
				if existing != string(raw) {
					return contracts.Fail("conflict")
				}
				continue
			}
			if e != sql.ErrNoRows {
				return e
			}
			if e = s.insertRecords(ctx, tx, Records{Profiles: []Profile{record}}); e != nil {
				return e
			}
		}
		retained := map[string]bool{}
		for _, destination := range relocations {
			old, e := s.publicationTx(ctx, tx, destination.ID)
			if e != nil {
				return e
			}
			if old.State != "available" || old.ArtifactID != destination.ArtifactID || old.Digest != destination.Digest || old.Size != destination.Size {
				return contracts.Fail("conflict")
			}
			retained[old.ID] = true
			if referenceOnly {
				if old.ProfileID != destination.ProfileID || old.ProfileRevision != destination.ProfileRevision || old.Key != destination.Key || old.Version != destination.Version {
					return contracts.Fail("conflict")
				}
				continue
			}
			p := old
			p.References = map[string]bool{}
			for id, v := range destination.References {
				if !backupReferences[id] {
					p.References[id] = v
				}
			}
			p.ProfileID = destination.ProfileID
			p.ProfileRevision = destination.ProfileRevision
			p.Key = destination.Key
			p.Version = destination.Version
			p.LocationID = contracts.ID()
			p.UploadID = ""
			p.Parts = nil
			p.PartSize = 0
			p.CompletionRequested = false
			p.Leases = map[string]MaterializationLease{}
			p.LeaseUntil = 0
			p.AdmissionID = contracts.ID()
			p.JournalReceiptID = p.AdmissionID
			p.Events = append(p.Events, "portable-restore")
			var version *string
			if p.Version != "" {
				version = &p.Version
			}
			if e = s.insertRecords(ctx, tx, Records{Locations: []ArtifactLocation{{ID: p.LocationID, ArtifactID: p.ArtifactID, ProfileID: p.ProfileID, ProfileRevision: p.ProfileRevision, Key: p.Key, Version: version, State: "available"}}}); e != nil {
				return e
			}
			d, e := intent(publicationIntent(p))
			if e != nil {
				return e
			}
			if _, e = s.accept(ctx, tx, p.AdmissionID, d, rev, p); e != nil {
				return e
			}
			rev++
			if e = s.putPublication(ctx, tx, &p, rev); e != nil {
				return e
			}
			rev++
		}
		// Omitted obsolete bytes must not appear available in restored authority.
		for _, table := range snap.State {
			if table.Name != "artifact_publication" {
				continue
			}
			for _, row := range table.Rows {
				var raw string
				strict(row[2], &raw)
				p, _ := decodePublication(raw)
				if retained[p.ID] || p.State != "available" {
					continue
				}
				p.State = "missing"
				p.Leases = map[string]MaterializationLease{}
				p.LeaseUntil = 0
				p.Events = append(p.Events, "backup-byte-excluded")
				if _, e := s.exec(ctx, tx, "UPDATE artifact_location SET state='missing' WHERE workspace_id=? AND id=?", s.workspace, p.LocationID); e != nil {
					return e
				}
				if e := s.putPublication(ctx, tx, &p, rev); e != nil {
					return e
				}
				rev++
			}
		}
		// Preserve old acknowledgement history under historical IDs while resetting
		// live graph authority. Replayed events can then rebuild either backend.
		for _, table := range snap.State {
			if table.Name != "graph_event" {
				continue
			}
			for _, row := range table.Rows {
				var target, op string
				var sequence int64
				if strict(row[0], &target) != nil || strict(row[1], &sequence) != nil || strict(row[4], &op) != nil {
					return contracts.Fail("invalid_request")
				}
				id := operationID("ack", target, op, strconv.FormatInt(sequence, 10))
				if _, e := s.exec(ctx, tx, "UPDATE operation_receipt SET id=? WHERE workspace_id=? AND id=?", operationID("restored-ack-history", snap.Digest, id), s.workspace, id); e != nil {
					return e
				}
			}
		}
		if _, e := s.exec(ctx, tx, "UPDATE graph_target SET checkpoint=0,owner_id=NULL,lease_until=0 WHERE workspace_id=?", s.workspace); e != nil {
			return e
		}
		if _, e := s.exec(ctx, tx, "UPDATE workspace_lease SET lease_until=0 WHERE workspace_id=?", s.workspace); e != nil {
			return e
		}
		profiles, e := portableProfilesDigest(w)
		if e != nil {
			return e
		}
		if _, e := s.accept(ctx, tx, operationID("portable-restore", snap.Digest), hash([]byte("portable-restore:"+snap.Digest)), rev, map[string]any{"portable_restore_digest": snap.Digest, "portable_restore_profiles": profiles, "graph_dirty": true}); e != nil {
			return e
		}
		rev++
		if activation != nil {
			for _, role := range []string{"portable-activation-pending", "portable-activation-complete"} {
				id := operationID(role)
				if _, e := s.exec(ctx, tx, "UPDATE operation_receipt SET id=? WHERE workspace_id=? AND id=?", operationID("restored-activation-history", snap.Digest, id), s.workspace, id); e != nil {
					return e
				}
			}
			if _, e := s.accept(ctx, tx, operationID("portable-activation-pending"), hash([]byte("portable-activation-pending:"+snap.Digest)), rev, map[string]any{"portable_activation_pending": snap.Digest}); e != nil {
				return e
			}
			rev++
		}
		if e := s.validateArtifactState(ctx, tx, false); e != nil {
			return e
		}
		if e := s.validateRestoredState(ctx, tx, rev); e != nil {
			return e
		}
		if activation != nil {
			now, e := s.now(ctx, tx)
			if e != nil {
				return e
			}
			var generation int64
			e = s.row(ctx, tx, "SELECT generation FROM workspace_lease WHERE workspace_id=? AND role='portable-transfer'", s.workspace).Scan(&generation)
			if e != nil && e != sql.ErrNoRows {
				return e
			}
			if generation == math.MaxInt64 {
				return contracts.Fail("conflict")
			}
			*activation = Lease{Role: "portable-transfer", OwnerID: contracts.ID(), Generation: generation + 1, Until: now + int64(30*time.Second)}
			if _, e = s.exec(ctx, tx, "INSERT INTO workspace_lease(workspace_id,role,owner_id,generation,lease_until) VALUES(?,'portable-transfer',?,?,?) ON CONFLICT(workspace_id,role) DO UPDATE SET owner_id=excluded.owner_id,generation=excluded.generation,lease_until=excluded.lease_until", s.workspace, activation.OwnerID, activation.Generation, activation.Until); e != nil {
				return e
			}
		}
		return s.validateLibraryState(ctx, tx, false)
	})
}

func (s *Store) RestoredPortable(ctx context.Context, digest string) (bool, error) {
	var found string
	e := s.db.QueryRowContext(ctx, s.query("SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?"), s.workspace, operationID("portable-restore", digest)).Scan(&found)
	if e == sql.ErrNoRows {
		return false, nil
	}
	if e != nil {
		return false, sanitize(e)
	}
	return found == hash([]byte("portable-restore:"+digest)), nil
}

func portableProfilesDigest(w *workspace.Workspace) (string, error) {
	profiles := []workspace.Profile{w.Config.Profiles.Storage, w.Config.Profiles.Catalog, w.Config.Profiles.Graph}
	for i, p := range profiles {
		configuration := map[string]any{}
		for k, v := range p.Configuration {
			configuration[k] = v
		}
		key := ""
		switch p.Adapter {
		case "filesystem":
			key = "root"
		case "sqlite", "ladybugdb":
			key = "path"
		}
		if key != "" {
			if value, ok := configuration[key].(string); ok && !filepath.IsAbs(value) {
				configuration[key] = filepath.Join(w.Control, value)
			}
		}
		profiles[i].Configuration = configuration
	}
	raw, e := json.Marshal(profiles)
	if e != nil {
		return "", contracts.Fail("invalid_request")
	}
	return hash(raw), nil
}

// PortableDestinationDigest binds physical destinations, resolving local
// profile paths against the configured control directory.
func PortableDestinationDigest(w *workspace.Workspace) (string, error) {
	return portableProfilesDigest(w)
}

// RestoredPortableWorkspace reconciles only the exact configured destination.
func (s *Store) RestoredPortableWorkspace(ctx context.Context, digest string, w *workspace.Workspace) (bool, error) {
	var proof, raw string
	e := s.db.QueryRowContext(ctx, s.query("SELECT digest,result FROM operation_receipt WHERE workspace_id=? AND id=?"), s.workspace, operationID("portable-restore", digest)).Scan(&proof, &raw)
	if e == sql.ErrNoRows {
		return false, nil
	}
	if e != nil {
		return false, sanitize(e)
	}
	profiles, e := portableProfilesDigest(w)
	if e != nil {
		return false, e
	}
	var result struct {
		Digest   string `json:"portable_restore_digest"`
		Profiles string `json:"portable_restore_profiles"`
	}
	if json.Unmarshal([]byte(raw), &result) != nil || proof != hash([]byte("portable-restore:"+digest)) || result.Digest != digest || result.Profiles != profiles {
		return false, contracts.Fail("conflict")
	}
	return true, nil
}

func (s *Store) pendingPortableActivation(ctx context.Context, tx *sql.Tx) (string, error) {
	var proof, raw string
	e := s.row(ctx, tx, "SELECT digest,result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, operationID("portable-activation-pending")).Scan(&proof, &raw)
	if e == sql.ErrNoRows {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	var pending struct {
		Digest string `json:"portable_activation_pending"`
	}
	if json.Unmarshal([]byte(raw), &pending) != nil || len(pending.Digest) != 64 || proof != hash([]byte("portable-activation-pending:"+pending.Digest)) {
		return "", contracts.Fail("invalid_request")
	}
	var complete string
	e = s.row(ctx, tx, "SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, operationID("portable-activation-complete")).Scan(&complete)
	if e == sql.ErrNoRows {
		return pending.Digest, nil
	}
	if e != nil {
		return "", e
	}
	if complete != hash([]byte("portable-activation-complete:"+pending.Digest)) {
		return "", contracts.Fail("conflict")
	}
	return "", nil
}

// BeginPortableActivation permits only recovery of the committed bundle and
// exact destination configuration. The durable pending proof blocks ordinary
// writes even when the operational transfer lease has expired.
func (s *Store) BeginPortableActivation(ctx context.Context, digest string, w *workspace.Workspace) (Lease, error) {
	ok, e := s.RestoredPortableWorkspace(ctx, digest, w)
	if e != nil {
		return Lease{}, e
	}
	if !ok {
		return Lease{}, contracts.Fail("conflict")
	}
	ctx = context.WithValue(WithoutTransfer(ctx), portableActivationKey{}, digest)
	return s.BeginTransfer(ctx, contracts.ID(), 30*time.Second)
}

// CompletePortableActivation follows the durable configuration replacement.
func (s *Store) CompletePortableActivation(ctx context.Context, lease Lease, digest string, w *workspace.Workspace) error {
	ok, e := s.RestoredPortableWorkspace(ctx, digest, w)
	if e != nil {
		return e
	}
	if !ok {
		return contracts.Fail("conflict")
	}
	ctx = context.WithValue(WithTransfer(ctx, lease), portableActivationKey{}, digest)
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		id := operationID("portable-activation-complete")
		d := hash([]byte("portable-activation-complete:" + digest))
		if _, ok, e := s.replay(ctx, tx, id, d); e != nil {
			return e
		} else if ok {
			return nil
		}
		_, e := s.accept(ctx, tx, id, d, rev, map[string]any{"portable_activation_complete": digest})
		return e
	})
}
