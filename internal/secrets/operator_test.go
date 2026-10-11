// SPDX-License-Identifier: Apache-2.0
package secrets

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestSavedVaultStatusWithoutUnlock(t *testing.T) {
	w := fixture(t)
	ctx := context.Background()
	id := contracts.ID()
	m, e := Open(w, Options{Mode: "vault", Passphrase: []byte("operator-passphrase-fixture")})
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Add(ctx, id, []byte("operator-secret-fixture")); e != nil {
		t.Fatal(e)
	}
	m.Close()
	cold, e := Open(w, Options{Mode: "vault"})
	if e != nil {
		t.Fatal(e)
	}
	defer cold.Close()
	status, e := cold.Inspect(ctx, id)
	raw, _ := json.Marshal(status)
	if e != nil || status.State != "locked" || strings.Contains(string(raw), "operator-secret-fixture") {
		t.Fatal(status, e)
	}
}
func TestDuplicateCredentialKeepsActualCause(t *testing.T) {
	m, e := Open(fixture(t), Options{Mode: "session"})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	id := contracts.ID()
	m.Add(context.Background(), id, []byte("fixture"))
	e = m.Add(context.Background(), id, []byte("secret"))
	raw, _ := json.Marshal(e)
	if !strings.Contains(string(raw), "credential_duplicate") || strings.Contains(string(raw), "Another owner") {
		t.Fatal(string(raw))
	}
}
