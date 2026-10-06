//go:build integration

package catalog

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"strings"
	"testing"
)

func postgresStore(t *testing.T, id string) *Store {
	t.Helper()
	dsn := os.Getenv("INSONIC_FIXTURE_POSTGRES")
	if dsn == "" {
		t.Fatal("INSONIC_FIXTURE_POSTGRES is required for integration acceptance")
	}
	cfg, e := pgx.ParseConfig(dsn)
	if e != nil {
		t.Fatal("invalid disposable fixture configuration")
	}
	schema := "test_" + strings.ReplaceAll(contracts.ID(), "-", "")
	auth, _ := json.Marshal(Credentials{cfg.User, cfg.Password})
	secrets := SessionSecrets{"fixture": auth}
	store, e := OpenPostgreSQL(context.Background(), PostgreSQLConfig{Host: cfg.Host, Port: cfg.Port, Database: cfg.Database, Schema: schema, TLSMode: "local", CredentialID: "fixture"}, id, secrets, true)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		store.db.ExecContext(context.Background(), "DROP SCHEMA \""+schema+"\" CASCADE")
		store.Close()
	})
	return store
}
func TestPostgreSQLCatalogContract(t *testing.T) {
	contractSuite(t, func(id string) *Store { return postgresStore(t, id) })
}
func TestPostgreSQLDurableJobs(t *testing.T)   { jobSuite(t, postgresStore(t, contracts.ID())) }
func TestPostgreSQLOrderedOutbox(t *testing.T) { outboxSuite(t, postgresStore(t, contracts.ID())) }
func TestPostgreSQLSecurityAndConcurrency(t *testing.T) {
	securitySuite(t, postgresStore(t, contracts.ID()))
}
func TestPostgreSQLCorpusAssociations(t *testing.T) { corpusSuite(t, postgresStore(t, contracts.ID())) }
func TestCrossBackendRoundTrip(t *testing.T) {
	ctx := context.Background()
	id := contracts.ID()
	local := localStore(t, id)
	job, e := local.StartJob(ctx, contracts.ID(), contracts.ID(), 1000, leaseTTLForTest)
	if e != nil {
		t.Fatal(e)
	}
	snap, e := local.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	server := postgresStore(t, id)
	if e = server.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	if e = server.Complete(ctx, job, "succeeded"); e == nil {
		t.Fatal("restored authority accepted")
	}
	next, e := server.Recover(ctx, contracts.ID(), leaseTTLForTest)
	if e != nil || len(next) != 1 || next[0].Generation != 2 {
		t.Fatalf("recovery %+v %v", next, e)
	}
	if e = server.Complete(ctx, next[0], "succeeded"); e != nil {
		t.Fatal(e)
	}
	back, e := server.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	restored := localStore(t, id)
	if e = restored.Restore(ctx, back); e != nil {
		t.Fatal(e)
	}
	history, e := restored.History(ctx, job.JobID)
	if e != nil || len(history) != 2 || history[1].State != "succeeded" {
		t.Fatalf("round trip %+v %v", history, e)
	}
}

const leaseTTLForTest = 5000000000

func TestPostgreSQLConvergence(t *testing.T) {
	convergenceSuite(t, postgresStore(t, contracts.ID()), func(id string) *Store { return postgresStore(t, id) })
}

func TestPostgreSQLCredentialOptionAliases(t *testing.T) {
	credentialOptionsSuite(t, postgresStore(t, contracts.ID()))
}
func TestPostgreSQLAckReconciliation(t *testing.T) {
	ackReconciliationSuite(t, postgresStore(t, contracts.ID()))
}

func TestPostgreSQLSnapshotCellTypes(t *testing.T) {
	snapshotCellSuite(t, postgresStore(t, contracts.ID()), func(id string) *Store { return postgresStore(t, id) })
}

func TestPostgreSQLCheckpointReceipts(t *testing.T) {
	checkpointReceiptSuite(t, postgresStore(t, contracts.ID()), func(id string) *Store { return postgresStore(t, id) })
}
func TestPostgreSQLSelectedRoutingAndTLS(t *testing.T) {
	cfg, e := pgx.ParseConfig(os.Getenv("INSONIC_FIXTURE_POSTGRES"))
	if e != nil {
		t.Fatal("fixture configuration")
	}
	t.Setenv("PGHOST", "invalid.invalid")
	t.Setenv("PGPORT", "1")
	t.Setenv("PGUSER", "ambient-user")
	t.Setenv("PGPASSWORD", "ambient-password")
	s := postgresStore(t, contracts.ID())
	if s.Backend() != "postgresql" {
		t.Fatal("ambient route or fallback")
	}
	auth, _ := json.Marshal(Credentials{cfg.User, cfg.Password})
	secret := SessionSecrets{"fixture": auth}
	c := PostgreSQLConfig{Host: cfg.Host, Port: cfg.Port, Database: cfg.Database, Schema: "unused", TLSMode: "verify-full", CredentialID: "fixture"}
	if got, e := OpenPostgreSQL(context.Background(), c, contracts.ID(), secret, true); e == nil {
		got.Close()
		t.Fatal("plaintext fallback from TLS")
	}
	c.TLSMode = "local"
	bad, _ := json.Marshal(Credentials{cfg.User, "wrong-fixture-password"})
	if got, e := OpenPostgreSQL(context.Background(), c, contracts.ID(), SessionSecrets{"fixture": bad}, true); e == nil {
		got.Close()
		t.Fatal("invalid authentication accepted")
	}
	if got, e := OpenPostgreSQL(context.Background(), c, contracts.ID(), SessionSecrets{"fixture": json.RawMessage(`{"username":"fixture","password":"x","unknown":true}`)}, true); e == nil {
		got.Close()
		t.Fatal("ambiguous credentials accepted")
	}
}
