// SPDX-License-Identifier: Apache-2.0
package secrets

import (
	"context"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestRestoredCatalogIdentityPreservesDestinationVault(t *testing.T) {
	ctx := context.Background()
	w := fixture(t)
	namespace, credential := w.Config.WorkspaceID, contracts.ID()
	if err := Select(w, "vault"); err != nil {
		t.Fatal(err)
	}
	opts := Options{Mode: "vault", Passphrase: []byte("namespace fixture passphrase")}
	before, err := Open(w, opts)
	if err != nil {
		t.Fatal(err)
	}
	if err = before.Add(ctx, credential, []byte("destination-only secret")); err != nil {
		t.Fatal(err)
	}
	before.Close()
	w.Config.CredentialNamespaceID = namespace
	w.Config.WorkspaceID = contracts.ID()
	if mode, err := ReadSelection(w); err != nil || mode != "vault" {
		t.Fatal(mode, err)
	}
	after, err := Open(w, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer after.Close()
	value, err := after.Resolve(ctx, credential)
	if err != nil || string(value) != "destination-only secret" {
		t.Fatal("destination vault did not survive catalog adoption", err)
	}
	clear(value)
	w.Config.CredentialNamespaceID = ""
	if _, err := ReadSelection(w); err == nil {
		t.Fatal("foreign catalog identity implicitly adopted local secrets")
	}
}

func TestNativeCredentialNamespaceAndDefault(t *testing.T) {
	w := fixture(t)
	if w.CredentialNamespace() != w.Config.WorkspaceID {
		t.Fatal("default namespace drift")
	}
	namespace := contracts.ID()
	w.Config.CredentialNamespaceID = namespace
	m, err := Open(w, Options{Mode: "native"})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if m.workspaceID != namespace {
		t.Fatal("native store used restored catalog identity")
	}
}
