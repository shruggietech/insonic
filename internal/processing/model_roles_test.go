// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
)

var roleValidationPassed = errors.New("required role validation passed; bytes intentionally unavailable")

type roleCatalog struct {
	catalog.Catalog
	install      catalog.BaseModelInstall
	publications int
}

func (c *roleCatalog) BaseModel(context.Context, string) (catalog.BaseModelInstall, error) {
	return c.install, nil
}

func (c *roleCatalog) Publication(context.Context, string) (catalog.Publication, error) {
	c.publications++
	return catalog.Publication{}, roleValidationPassed
}

func TestModelRequiresCanonicalRoleCaseBeforeAccessingBytes(t *testing.T) {
	for _, capability := range []string{"transcription", "diarization"} {
		roles := []string{"config.json", "model.bin", "tokenizer.json", "vocabulary.txt"}
		if capability == "diarization" {
			roles = []string{"config.yaml", "embedding/pytorch_model.bin", "segmentation/pytorch_model.bin", "plda/plda.npz", "plda/xvec_transform.npz"}
		}
		for _, scenario := range []string{"exact", "mixed-case-file", "mixed-case-directory", "case-collision"} {
			t.Run(capability+"/"+scenario, func(t *testing.T) {
				manifest := models.Manifest{Kind: "base-model-manifest", Version: contracts.Version, Name: "role-validation", ModelVersion: "1", Revision: "immutable-1", License: "MIT", Capabilities: []string{capability}}
				for _, role := range roles {
					manifest.Files = append(manifest.Files, models.File{Role: role, SHA256: strings.Repeat("a", 64), Size: 1, URL: "https://models.example.org/" + role})
				}
				switch scenario {
				case "mixed-case-file":
					manifest.Files[0].Role = "Config" + strings.TrimPrefix(manifest.Files[0].Role, "config")
				case "mixed-case-directory":
					index := 0
					if capability == "diarization" {
						index = 1
					}
					manifest.Files[index].Role = strings.ToUpper(manifest.Files[index].Role)
				case "case-collision":
					alias := manifest.Files[0]
					alias.Role = strings.ToUpper(alias.Role)
					manifest.Files = append(manifest.Files, alias)
				}
				ids := []string{}
				for range manifest.Files {
					ids = append(ids, contracts.ID())
				}
				raw, _ := json.Marshal(manifest)
				publications, _ := json.Marshal(ids)
				db := &roleCatalog{install: catalog.BaseModelInstall{State: "available", Digest: manifest.Digest(), Manifest: raw, PublicationIDs: publications}}
				directory := t.TempDir()
				session := &Session{directory: directory, service: &Service{Catalog: db}}
				_, _, err := session.model(context.Background(), contracts.ID(), capability)
				if scenario == "exact" {
					if !errors.Is(err, roleValidationPassed) || db.publications != 1 {
						t.Fatalf("canonical roles did not reach byte validation: %v, calls %d", err, db.publications)
					}
				} else {
					typed, ok := err.(*contracts.Error)
					want := "model_unavailable"
					if scenario == "case-collision" {
						want = "invalid_request"
					}
					if !ok || typed.Code != want || db.publications != 0 {
						t.Fatalf("noncanonical roles reached byte access: %v, calls %d", err, db.publications)
					}
				}
				entries, readErr := os.ReadDir(directory)
				if readErr != nil || len(entries) != 0 {
					t.Fatalf("rejected candidate left scratch: %v, entries %d", readErr, len(entries))
				}
			})
		}
	}
}

func TestModelManifestFenceRunsBeforePublicationAccess(t *testing.T) {
	manifest := models.Manifest{Kind: "base-model-manifest", Version: contracts.Version, Name: "frozen-model", ModelVersion: "1", Revision: "immutable", License: "MIT", Capabilities: []string{"transcription"}}
	ids := []string{}
	for _, role := range []string{"config.json", "model.bin", "tokenizer.json", "vocabulary.txt"} {
		manifest.Files = append(manifest.Files, models.File{Role: role, SHA256: strings.Repeat("a", 64), Size: 1, URL: "https://models.example.org/" + role})
		ids = append(ids, contracts.ID())
	}
	raw, _ := json.Marshal(manifest)
	publications, _ := json.Marshal(ids)
	for _, matching := range []bool{false, true} {
		db := &roleCatalog{install: catalog.BaseModelInstall{State: "available", Digest: manifest.Digest(), Manifest: raw, PublicationIDs: publications}}
		directory := t.TempDir()
		session := &Session{directory: directory, service: &Service{Catalog: db}}
		expected := strings.Repeat("0", 64)
		if matching {
			expected = manifest.Digest()
		}
		_, _, err := session.modelWithDigest(context.Background(), contracts.ID(), "transcription", expected)
		if matching {
			if !errors.Is(err, roleValidationPassed) || db.publications != 1 {
				t.Fatal("matching digest did not reach verified publication boundary")
			}
		} else {
			typed, ok := err.(*contracts.Error)
			if !ok || typed.Code != "conflict" || db.publications != 0 {
				t.Fatal("changed manifest accessed model bytes")
			}
		}
		entries, e := os.ReadDir(directory)
		if e != nil || len(entries) != 0 {
			t.Fatal("model fence left scratch")
		}
	}
}
