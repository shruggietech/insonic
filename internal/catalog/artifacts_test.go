// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArtifactAuthority(t *testing.T) {
	ctx := context.Background()
	id := contracts.ID()
	s, e := OpenSQLite(ctx, filepath.Join(t.TempDir(), "catalog.sqlite"), id)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	artifactAuthoritySuite(t, s, func(id string) *Store {
		dst, e := OpenSQLite(ctx, filepath.Join(t.TempDir(), "restore.sqlite"), id)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { dst.Close() })
		return dst
	})
}
func artifactAuthoritySuite(t *testing.T, s *Store, newStore func(string) *Store) {
	ctx := context.Background()
	id := s.workspace
	var e error
	profile := Profile{ID: contracts.ID(), Revision: 1, Role: "storage", Adapter: "filesystem", Version: contracts.Version, Configuration: []byte(`{"root":"artifacts"}`)}
	_, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: 0, Records: Records{Profiles: []Profile{profile}}})
	if e != nil {
		t.Fatal(e)
	}
	p := Publication{ID: contracts.ID(), ArtifactID: contracts.ID(), LocationID: contracts.ID(), ProfileID: profile.ID, ProfileRevision: 1, Digest: strings.Repeat("a", 64), Size: 3, Kind: "other", Key: "objects/" + contracts.ID(), Owner: contracts.ID()}
	p, e = s.BeginPublication(ctx, p, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	stale := p
	stale.Generation++
	if _, e = s.SavePublication(ctx, stale, time.Second); e == nil {
		t.Fatal("forged generation admitted")
	}
	p.Verification = "sha256-readback"
	p, e = s.AdmitPublication(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	admitted := p
	lease, e := s.ArtifactLease(ctx, p.ID, contracts.ID(), contracts.ID(), time.Second, false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Second); e == nil {
		t.Fatal("retired live materialization")
	}
	_, e = s.ArtifactLease(ctx, p.ID, lease.ID, lease.Owner, 0, true)
	if e != nil {
		t.Fatal(e)
	}
	ref := contracts.ID()
	if e = s.ArtifactReference(ctx, p.ID, ref, false); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Second); e == nil {
		t.Fatal("retired retained artifact")
	}
	if e = s.ArtifactReference(ctx, p.ID, ref, true); e != nil {
		t.Fatal(e)
	}
	p, e = s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ArtifactReference(ctx, p.ID, contracts.ID(), false); e == nil {
		t.Fatal("retirement allowed reference")
	}
	if _, e = s.ArtifactLease(ctx, p.ID, contracts.ID(), contracts.ID(), time.Second, false); e == nil {
		t.Fatal("retirement allowed lease")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	dst := newStore(id)
	if e = dst.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	restored, e := dst.Publication(ctx, p.ID)
	if e != nil || restored.State != "retiring" || restored.LeaseUntil != 0 {
		t.Fatalf("restore: %+v %v", restored, e)
	}
	if _, e = dst.FinishRetirement(ctx, p); e == nil {
		t.Fatal("imported authority retained")
	}
	forged := snap
	forged.State = append([]TableData(nil), snap.State...)
	table := forged.State[len(forged.State)-1]
	table.Rows = append([][]json.RawMessage(nil), table.Rows...)
	row := append([]json.RawMessage(nil), table.Rows[0]...)
	var data string
	json.Unmarshal(row[2], &data)
	tampered, _ := decodePublication(data)
	tampered.State = "available"
	raw, _ := json.Marshal(tampered)
	row[2], _ = json.Marshal(string(raw))
	table.Rows[0] = row
	forged.State[len(forged.State)-1] = table
	forged.Digest, _ = forged.digest()
	if e = newStore(id).Restore(ctx, forged); e == nil {
		t.Fatal("rehashed lifecycle forgery restored")
	}
	// An individually valid admission receipt cannot replace later lifecycle
	// history while the newer receipts remain in the transferred catalog.
	raw, _ = json.Marshal(admitted)
	row[2], _ = json.Marshal(string(raw))
	table.Rows[0] = row
	forged.State[len(forged.State)-1] = table
	forged.Digest, _ = forged.digest()
	if e = newStore(id).Restore(ctx, forged); e == nil {
		t.Fatal("old journal paired with newer receipts restored")
	}
	p, e = s.ClaimRetirement(ctx, p.ID, p.Owner, 0, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	p, e = s.FinishRetirement(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	retired, e := s.Export(ctx)
	if e != nil || len(retired.Records.Locations) != 1 || retired.Records.Locations[0].State != "retired" {
		t.Fatal("typed location availability disagrees with retirement")
	}
	if e = newStore(id).Restore(ctx, retired); e != nil {
		t.Fatal("retired location snapshot failed", e)
	}
}
