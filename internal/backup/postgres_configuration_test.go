// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
)

func TestGeneratedPostgreSQLProfileExportsWithinWorkspaceContract(t *testing.T) {
	for _, explicitOptional := range []bool{false, true} {
		t.Run(map[bool]string{false: "omitted-optionals", true: "configured-optionals"}[explicitOptional], func(t *testing.T) {
			ctx := context.Background()
			w, db, assets := fixture(t)
			publish(t, assets, "portable immutable fixture", "canonical-audio")
			config := catalog.PostgreSQLConfig{Host: "db.example.test", Port: 5432, Database: "fixture", Schema: "insonic", TLSMode: "local"}
			if explicitOptional {
				config.TLSMode = "verify-full"
				config.CAFile = "/selected/ca.pem"
				config.CredentialID = contracts.ID()
			}
			raw, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			var configuration map[string]any
			if err = json.Unmarshal(raw, &configuration); err != nil {
				t.Fatal(err)
			}
			// This test exercises profile serialization and export validation on a
			// local catalog. Live PostgreSQL source I/O remains in the backend matrix.
			w.Config.Profiles.Catalog.ID = contracts.ID()
			w.Config.Profiles.Catalog.Adapter = "postgresql"
			w.Config.Profiles.Catalog.Configuration = configuration
			workspaceJSON, _ := json.Marshal(w.Config)
			if err = schemas.ValidateWorkspace(workspaceJSON); err != nil {
				t.Fatal("generated PostgreSQL profile violates public contract", err)
			}
			if err = db.RegisterWorkspace(ctx, w); err != nil {
				t.Fatal(err)
			}
			directory := filepath.Join(t.TempDir(), "bundle")
			manifest, err := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: directory})
			if err != nil {
				t.Fatal("export configured PostgreSQL profile", err)
			}
			if _, err = Inspect(ctx, directory); err != nil {
				t.Fatal("inspect exported profile", err)
			}
			for _, key := range []string{"ca_file", "credential_id"} {
				value, exists := manifest.Config.Profiles.Catalog.Configuration[key]
				if exists != explicitOptional || exists && value == "" {
					t.Fatal("optional PostgreSQL configuration lost presence", key)
				}
			}
		})
	}
}
