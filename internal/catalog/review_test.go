package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
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
	claim, e := s.ClaimOutbox(ctx, target, contracts.ID(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.AcknowledgeOutbox(ctx, claim); e != nil {
		t.Fatal(e)
	}
	time.Sleep(1100 * time.Millisecond)
	if e = s.AcknowledgeOutbox(ctx, claim); e != nil {
		t.Fatal("committed ack could not reconcile after expiry", e)
	}
	next, e := s.ClaimOutbox(ctx, target, contracts.ID(), time.Second)
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
	uncommitted, e := s.ClaimOutbox(ctx, target, next.OwnerID, time.Second)
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
