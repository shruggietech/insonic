package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func convergenceSuite(t *testing.T, s *Store, empty func(string) *Store) {
	ctx := context.Background()
	owner := contracts.ID()
	t.Run("paged history", func(t *testing.T) {
		a, e := s.StartJob(ctx, contracts.ID(), owner, 1, time.Minute)
		if e != nil {
			t.Fatal(e)
		}
		for i := 0; i < 129; i++ {
			if e = s.Complete(ctx, a, "failed"); e != nil {
				t.Fatal(e)
			}
			a, e = s.RetryJob(ctx, contracts.ID(), a.JobID, owner, time.Minute)
			if e != nil {
				t.Fatal(e)
			}
		}
		page, e := s.HistoryPage(ctx, a.JobID, 0)
		if e != nil || len(page.Attempts) != 128 || page.NextGeneration == nil || *page.NextGeneration != 128 {
			t.Fatalf("first page %+v %v", page, e)
		}
		next, e := s.HistoryPage(ctx, a.JobID, *page.NextGeneration)
		if e != nil || len(next.Attempts) != 2 || next.Attempts[0].Generation != 129 || next.NextGeneration != nil {
			t.Fatalf("next page %+v %v", next, e)
		}
		if _, e = s.HistoryPage(ctx, a.JobID, -1); e == nil {
			t.Fatal("negative cursor")
		}
	})
	t.Run("cancel publication race", func(t *testing.T) {
		for i := 0; i < 12; i++ {
			a, e := s.StartJob(ctx, contracts.ID(), owner, 1, time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			start := make(chan struct{})
			done := make(chan error, 1)
			cancel := make(chan error, 1)
			go func() { <-start; done <- s.Complete(ctx, a, "succeeded") }()
			go func() { <-start; _, e := s.CancelJob(ctx, contracts.ID(), a.JobID); cancel <- e }()
			close(start)
			completeErr, cancelErr := <-done, <-cancel
			if cancelErr != nil {
				t.Fatal(cancelErr)
			}
			current, e := s.ShowJob(ctx, a.JobID)
			if e != nil {
				t.Fatal(e)
			}
			if (current.State == "succeeded") != (completeErr == nil) {
				t.Fatalf("terminal mismatch %s %v", current.State, completeErr)
			}
			if current.State != "succeeded" && current.State != "cancelled" {
				t.Fatal(current.State)
			}
			history, e := s.History(ctx, a.JobID)
			if e != nil || len(history) != 1 || history[0].EndedNS == nil {
				t.Fatal("terminal history", e)
			}
		}
	})
	t.Run("shutdown evidence survives recovery", func(t *testing.T) {
		a, e := s.StartJob(ctx, contracts.ID(), owner, 1, time.Minute)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.InterruptOwner(ctx, owner); e != nil {
			t.Fatal(e)
		}
		before, e := s.History(ctx, a.JobID)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.Recover(ctx, contracts.ID(), time.Minute); e != nil {
			t.Fatal(e)
		}
		after, e := s.History(ctx, a.JobID)
		if e != nil {
			t.Fatal(e)
		}
		if len(after) != 2 || after[0].Reason != "shutdown" || *after[0].EndedNS != *before[0].EndedNS {
			t.Fatal("recovery rewrote shutdown evidence")
		}
	})
	t.Run("recomputed digest cannot hide event revision drift", func(t *testing.T) {
		rev, e := s.Revision(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Projections: []Projection{{Target: contracts.ID(), Document: json.RawMessage(`{"accepted":true}`)}}}); e != nil {
			t.Fatal(e)
		}
		snap, e := s.Export(ctx)
		if e != nil {
			t.Fatal(e)
		}
		for i := range snap.State {
			if snap.State[i].Name == "graph_event" {
				snap.State[i].Rows[0][3] = json.RawMessage(`1`)
			}
		}
		snap.Digest, e = snap.digest()
		if e != nil {
			t.Fatal(e)
		}
		dest := empty(s.workspace)
		if e = dest.Restore(ctx, snap); e == nil {
			t.Fatal("event receipt drift accepted")
		}
		got, _ := dest.Revision(ctx)
		if got != 0 {
			t.Fatal("failed restore partially committed")
		}
	})
}
func TestSQLiteConvergence(t *testing.T) {
	convergenceSuite(t, localStore(t, contracts.ID()), func(id string) *Store { return localStore(t, id) })
}
func TestSelectedBackendVersion(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	w, e := workspace.Init(root, "fixture")
	if e != nil {
		t.Fatal(e)
	}
	w.Config.Profiles.Catalog.ExpectedBackendVersion = ">=3,<4"
	s, e := OpenWorkspace(ctx, w, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, p := range snap.Records.Profiles {
		if p.Role == "catalog" {
			found = p.ExpectedBackendVersion != nil && *p.ExpectedBackendVersion == ">=3,<4"
		}
	}
	if !found {
		t.Fatal("constraint missing from portable profile")
	}
	validateSnapshotSchema(t, snap)
	w.Config.Profiles.Catalog.ExpectedBackendVersion = ">=4"
	if got, e := OpenWorkspace(ctx, w, nil, false); e == nil {
		got.Close()
		t.Fatal("incompatible backend accepted")
	}
	w.Config.Profiles.Catalog.Adapter = "postgresql"
	w.Config.Profiles.Catalog.Configuration = map[string]any{"host": "127.0.0.1", "port": 54329, "database": "fixture", "schema": "unused", "tls_mode": "local", "credential_id": "missing"}
	if got, e := OpenWorkspace(ctx, w, SessionSecrets{}, false); e == nil {
		got.Close()
		t.Fatal("unresolved selected credential accepted")
	}
	if _, e := OpenSQLite(ctx, filepath.Join(root, "extra.sqlite"), "invalid"); e == nil {
		t.Fatal("invalid scope accepted")
	}
}

func validateSnapshotSchema(t *testing.T, snap Snapshot) {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	upstream, e := jsonschema.UnmarshalJSON(bytes.NewReader(subtitles.SchemaBytes()))
	if e != nil {
		t.Fatal(e)
	}
	if e = compiler.AddResource(subtitles.SchemaID, upstream); e != nil {
		t.Fatal(e)
	}
	paths, e := filepath.Glob("../../schemas/v1.0.0/*.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		data, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		if e = compiler.AddResource(doc.(map[string]any)["$id"].(string), doc); e != nil {
			t.Fatal(e)
		}
	}
	schema, e := compiler.Compile("https://raw.githubusercontent.com/shruggietech/insonic/v1.0.0/schemas/v1.0.0/catalog-snapshot.schema.json")
	if e != nil {
		t.Fatal(e)
	}
	data, e := json.Marshal(snap)
	if e != nil {
		t.Fatal(e)
	}
	doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if e = schema.Validate(doc); e != nil {
		t.Fatal("populated exported snapshot violates its registered schema", e)
	}
}
