package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
	"testing"
	"time"
)

func localStore(t *testing.T, id string) *Store {
	t.Helper()
	s, err := OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "catalog.sqlite"), id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSQLiteCatalogContract(t *testing.T) {
	contractSuite(t, func(id string) *Store { return localStore(t, id) })
}

func contractSuite(t *testing.T, open func(string) *Store) {
	ctx := context.Background()
	wid := contracts.ID()
	s := open(wid)
	t.Run("revision and idempotency", func(t *testing.T) {
		op := contracts.ID()
		m := Mutation{OperationID: op, Expected: 0, Settings: []Setting{{Name: "theme", Value: json.RawMessage(`"dark"`)}}}
		r, err := s.Commit(ctx, m)
		if err != nil || r.Revision != 1 {
			t.Fatalf("%+v %v", r, err)
		}
		same, err := s.Commit(ctx, m)
		if err != nil || same.Revision != r.Revision {
			t.Fatalf("replay %+v %v", same, err)
		}
		m.Settings[0].Value = json.RawMessage(`"light"`)
		if _, err = s.Commit(ctx, m); err == nil {
			t.Fatal("changed operation accepted")
		}
		m.OperationID = contracts.ID()
		if _, err = s.Commit(ctx, m); err == nil {
			t.Fatal("stale revision accepted")
		}
	})
	t.Run("immutable scoped provenance and exact time", func(t *testing.T) {
		rev, _ := s.Revision(ctx)
		a := contracts.ID()
		asset := contracts.ID()
		snap := contracts.ID()
		ns := time.Date(2026, 10, 5, 12, 0, 0, 123456789, time.UTC).UnixNano()
		rows := Records{Artifacts: []Artifact{{ID: a, Digest: string(make([]byte, 0)), Size: 0, Kind: "source"}}, Assets: []Asset{{ID: asset, ArtifactID: a, Role: "original", DurationUS: 1000000}}, Metadata: []MetadataSnapshot{{ID: snap, AssetID: asset, State: "no-embedded-metadata", Extractor: "fixture", Version: "1", Captured: Instant{ISO: time.Unix(0, ns).UTC().Format(time.RFC3339Nano), UnixNS: ns}}}}
		rows.Artifacts[0].Digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		m := Mutation{OperationID: contracts.ID(), Expected: rev, Records: rows}
		if _, err := s.Commit(ctx, m); err != nil {
			t.Fatal(err)
		}
		got, err := s.Export(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got.Records.Metadata[0].Captured.UnixNS != ns {
			t.Fatal("timestamp rounded")
		}
		rev, _ = s.Revision(ctx)
		m.Expected = rev
		m.OperationID = contracts.ID()
		if _, err = s.Commit(ctx, m); err == nil {
			t.Fatal("immutable rewrite accepted")
		}
		rows.Assets = []Asset{{ID: contracts.ID(), ArtifactID: contracts.ID(), Role: "original", DurationUS: 1}}
		rows.Artifacts = nil
		rows.Metadata = nil
		if _, err = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: rows}); err == nil {
			t.Fatal("missing scoped parent accepted")
		}
		after, _ := s.Revision(ctx)
		if after != rev {
			t.Fatal("failed admission changed revision")
		}
	})
	t.Run("snapshot integrity and restore", func(t *testing.T) {
		snap, err := s.Export(ctx)
		if err != nil {
			t.Fatal(err)
		}
		dest := open(wid)
		if err = dest.Restore(ctx, snap); err != nil {
			t.Fatal(err)
		}
		got, _ := dest.Export(ctx)
		if len(got.Records.Metadata) != len(snap.Records.Metadata) || got.Records.Metadata[0].Captured != snap.Records.Metadata[0].Captured {
			t.Fatal("restore mismatch")
		}
		if err = dest.Restore(ctx, snap); err == nil {
			t.Fatal("occupied restore accepted")
		}
		bad := snap
		bad.Revision++
		if err = open(wid).Restore(ctx, bad); err == nil {
			t.Fatal("corrupt restore accepted")
		}
	})
}
