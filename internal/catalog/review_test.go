package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func credentialOptionsSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	rev, e := s.Revision(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, key := range []string{"Password", "CLIENT_SECRET", "token", "Authorization", "apiKey", "X-API-Key", "refresh.token", "private-key", "AWS_SECRET_ACCESS_KEY", "service_password", "bearerToken", "passphrase", "clientSecret"} {
		value, _ := json.Marshal(map[string]any{"nested": []any{map[string]any{key: "fixture-credential-must-not-persist"}}})
		if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Settings: []Setting{{Name: "options", Value: value}}}); e == nil {
			t.Fatalf("credential field %q admitted", key)
		}
		for _, record := range []any{Profile{ID: contracts.ID(), Revision: 1, Role: "catalog", Adapter: "extension", Version: contracts.Version, Configuration: value}, Dataset{ID: contracts.ID(), SpeakerID: contracts.ID(), ManifestArtifactID: contracts.ID(), Options: value}, TrainingRun{ID: contracts.ID(), DatasetID: contracts.ID(), SpeakerID: contracts.ID(), JobID: contracts.ID(), PreparationArtifactID: contracts.ID(), Adapter: "fixture", Options: value}} {
			if validateRecord(record) == nil {
				t.Fatalf("%T admitted credential field %q", record, key)
			}
		}
	}
	if !nonsecret(json.RawMessage(`{"credential_id":"opaque-reference","max_tokens":2048,"tokenizer":"fixture","secret_ref":"opaque-reference"}`)) {
		t.Fatal("nonsecret references or model options rejected")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	data, _ := json.Marshal(snap)
	if strings.Contains(string(data), "fixture-credential-must-not-persist") {
		t.Fatal("credential leaked into export")
	}
	if snap.Revision != rev {
		t.Fatal("rejected credentials changed authority")
	}
}
func TestSQLiteCredentialOptionAliases(t *testing.T) {
	credentialOptionsSuite(t, localStore(t, contracts.ID()))
}

func ackReconciliationSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	target := contracts.ID()
	for i := 0; i < 3; i++ {
		rev, e := s.Revision(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Projections: []Projection{{Target: target, Document: json.RawMessage(`{"replay":true}`)}}}); e != nil {
			t.Fatal(e)
		}
	}
	claim, e := s.ClaimOutbox(ctx, target, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.AcknowledgeOutbox(ctx, claim); e != nil {
		t.Fatal(e)
	}
	expireOutboxLease(t, s, claim)
	if e = s.AcknowledgeOutbox(ctx, claim); e != nil {
		t.Fatal("committed ack could not reconcile after expiry", e)
	}
	next, e := s.ClaimOutbox(ctx, target, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.AcknowledgeOutbox(ctx, next); e != nil {
		t.Fatal(e)
	}
	if e = s.AcknowledgeOutbox(ctx, claim); e != nil {
		t.Fatal("committed ack could not reconcile after takeover/checkpoint advance", e)
	}
	rev, e := s.Revision(ctx)
	if e != nil {
		t.Fatal(e)
	}
	bad := claim
	bad.Document = json.RawMessage(`{"replay":false}`)
	if e = s.AcknowledgeOutbox(ctx, bad); e == nil {
		t.Fatal("changed committed event identity reconciled")
	}
	uncommitted, e := s.ClaimOutbox(ctx, target, next.OwnerID, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	uncommitted.OwnerID = claim.OwnerID
	uncommitted.Generation = claim.Generation
	if e = s.AcknowledgeOutbox(ctx, uncommitted); e == nil {
		t.Fatal("obsolete owner published a new ack")
	}
	after, _ := s.Revision(ctx)
	if after != rev {
		t.Fatal("reconciliation changed revision")
	}
}
func TestSQLiteAckReconciliation(t *testing.T) {
	ackReconciliationSuite(t, localStore(t, contracts.ID()))
}

func snapshotCellSuite(t *testing.T, s *Store, empty func(string) *Store) {
	ctx := context.Background()
	if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: 0, Settings: []Setting{{Name: "choice", Value: json.RawMessage(`"kept"`)}}}); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := json.Marshal(snap)
	if e != nil {
		t.Fatal(e)
	}
	for _, change := range []struct {
		table  string
		column int
		raw    string
	}{{"operation_receipt", 0, "1"}, {"operation_receipt", 2, `"1"`}, {"setting_revision", 0, "1"}} {
		var bad Snapshot
		if e = json.Unmarshal(encoded, &bad); e != nil {
			t.Fatal(e)
		}
		for i := range bad.State {
			if bad.State[i].Name == change.table {
				bad.State[i].Rows[0][change.column] = json.RawMessage(change.raw)
			}
		}
		bad.Digest, e = bad.digest()
		if e != nil {
			t.Fatal(e)
		}
		dest := empty(s.workspace)
		if e = dest.Restore(ctx, bad); e == nil {
			t.Fatalf("%s column %d coerced invalid cell", change.table, change.column)
		}
		typed, ok := e.(*contracts.Error)
		if !ok || typed.Code != "invalid_request" {
			t.Fatalf("backend-dependent input error %v", e)
		}
		if rev, e := dest.Revision(ctx); e != nil || rev != 0 {
			t.Fatal("malformed restore changed authority", e)
		}
	}
}
func TestSQLiteSnapshotCellTypes(t *testing.T) {
	snapshotCellSuite(t, localStore(t, contracts.ID()), func(id string) *Store { return localStore(t, id) })
}

func checkpointReceiptSuite(t *testing.T, s *Store, empty func(string) *Store) {
	ctx := context.Background()
	target := contracts.ID()
	if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: 0, Projections: []Projection{{Target: target, Document: json.RawMessage(`{"accepted":true}`)}}}); e != nil {
		t.Fatal(e)
	}
	claim, e := s.ClaimOutbox(ctx, target, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	forged, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for i := range forged.State {
		if forged.State[i].Name == "graph_target" {
			forged.State[i].Rows[0][2] = json.RawMessage(`1`)
		}
	}
	forged.Digest, e = forged.digest()
	if e != nil {
		t.Fatal(e)
	}
	dest := empty(s.workspace)
	if e = dest.Restore(ctx, forged); e == nil {
		t.Fatal("checkpoint without acknowledgement accepted")
	}
	if rev, _ := dest.Revision(ctx); rev != 0 {
		t.Fatal("partial forged restore")
	}
	if e = s.AcknowledgeOutbox(ctx, claim); e != nil {
		t.Fatal(e)
	}
	valid, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = dest.Restore(ctx, valid); e != nil {
		t.Fatal("legitimate checkpoint could not restore", e)
	}
	for _, damage := range []string{"rewind", "digest", "result"} {
		data, _ := json.Marshal(valid)
		var bad Snapshot
		if e = json.Unmarshal(data, &bad); e != nil {
			t.Fatal(e)
		}
		for i := range bad.State {
			if damage == "rewind" && bad.State[i].Name == "graph_target" {
				bad.State[i].Rows[0][2] = json.RawMessage(`0`)
			}
			if bad.State[i].Name == "operation_receipt" {
				for j := range bad.State[i].Rows {
					var id string
					json.Unmarshal(bad.State[i].Rows[j][0], &id)
					if id == operationID("ack", target, claim.OperationID, "1") {
						if damage == "digest" {
							bad.State[i].Rows[j][1], _ = json.Marshal(strings.Repeat("a", 64))
						}
						if damage == "result" {
							bad.State[i].Rows[j][3], _ = json.Marshal(`{"target":"` + contracts.ID() + `","sequence":1}`)
						}
					}
				}
			}
		}
		bad.Digest, e = bad.digest()
		if e != nil {
			t.Fatal(e)
		}
		if e = empty(s.workspace).Restore(ctx, bad); e == nil {
			t.Fatalf("damaged ack %s accepted", damage)
		}
	}
}
func TestSQLiteCheckpointReceipts(t *testing.T) {
	checkpointReceiptSuite(t, localStore(t, contracts.ID()), func(id string) *Store { return localStore(t, id) })
}

func TestLargeCatalogSnapshotRestores(t *testing.T) {
	ctx := context.Background()
	id := contracts.ID()
	source := localStore(t, id)
	title := strings.Repeat("x", (64<<20)+1)
	if _, e := source.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: 0, Records: Records{Media: []Media{{ID: contracts.ID(), Title: title, Class: "audio"}}}}); e != nil {
		t.Fatal(e)
	}
	snap, e := source.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	file, e := os.CreateTemp(t.TempDir(), "large-snapshot-*")
	if e != nil {
		t.Fatal(e)
	}
	defer file.Close()
	if e = json.NewEncoder(file).Encode(snap); e != nil {
		t.Fatal(e)
	}
	info, e := file.Stat()
	if e != nil || info.Size() <= 64<<20 {
		t.Fatal("fixture did not exceed former cap", e)
	}
	if _, e = file.Seek(0, 0); e != nil {
		t.Fatal(e)
	}
	parsed, e := ReadSnapshotReader(file)
	if e != nil {
		t.Fatal("own large export could not be read", e)
	}
	restored := localStore(t, id)
	if e = restored.Restore(ctx, parsed); e != nil {
		t.Fatal(e)
	}
	out, e := restored.Export(ctx)
	if e != nil || len(out.Records.Media) != 1 || out.Records.Media[0].Title != title {
		t.Fatal("large record did not round trip", e)
	}
	for _, raw := range [][]byte{[]byte(`{"kind":"catalog-snapshot","kind":"other"}`), {'"', 0xff, '"'}, []byte(`{} {}`)} {
		if _, e = ReadSnapshotReader(bytes.NewReader(raw)); e == nil {
			t.Fatal("stream validation admitted damaged/ambiguous JSON")
		}
	}
}

func TestSnapshotStreamPreservesSplitUnicode(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	title := "café 🎵 東京"
	if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: 0, Records: Records{Media: []Media{{ID: contracts.ID(), Title: title, Class: "audio"}}}}); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := json.Marshal(snap)
	if e != nil {
		t.Fatal(e)
	}
	read, e := ReadSnapshotReader(iotest.OneByteReader(bytes.NewReader(encoded)))
	if e != nil || read.Records.Media[0].Title != title || read.Digest != snap.Digest {
		t.Fatal("split Unicode changed validated snapshot", e)
	}
	for _, bad := range [][]byte{[]byte("{\"kind\":\"\xf0\x9f"), []byte("{\"kind\":\"\xc3x\"}")} {
		if _, e = ReadSnapshotReader(iotest.OneByteReader(bytes.NewReader(bad))); e == nil {
			t.Fatal("invalid/incomplete split UTF-8 admitted")
		}
	}
}
