// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestTrainerDeclaredBytesRejectedBeforePublication(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	for _, mode := range []string{"valid", "hash", "size", "missing", "malformed"} {
		for _, inline := range []bool{false, true} {
			t.Run(mode+"/"+strconv.FormatBool(inline), func(t *testing.T) {
				dir := t.TempDir()
				claim := catalog.Work{ID: contracts.ID(), Generation: 1}
				a := Artifact{Role: "model.bin", Format: "fixture", Size: 3, SHA256: hash([]byte("abc"))}
				if inline {
					a.Data = []byte("abc")
				} else {
					a.Path = "model.bin"
					if mode != "missing" {
						if err := os.WriteFile(filepath.Join(dir, a.Path), []byte("abc"), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				if mode == "hash" {
					a.SHA256 = strings.Repeat("f", 64)
				}
				if mode == "size" {
					a.Size = 4
				}
				if mode == "malformed" {
					a.SHA256 = "invalid"
				}
				p, err := s.publishOutput(ctx, claim, a, dir, 0, "speaker-model")
				if mode == "valid" || mode == "missing" && inline {
					if err != nil || p.State != "available" {
						t.Fatal(p, err)
					}
					return
				}
				want := "checksum_mismatch"
				if mode == "missing" {
					want = "unavailable"
				}
				if mode == "malformed" {
					want = "invalid_engine_output"
				}
				if err == nil || contracts.SafeError(err).Code != want {
					t.Fatalf("want %s got %v", want, err)
				}
				id := models.StableID("speaker-artifact:" + claim.ID + ":1:0:" + a.SHA256)
				if _, e := db.Publication(ctx, id); e == nil || contracts.SafeError(e).Code != "not_found" {
					t.Fatal("rejected output reached catalog", e)
				}
				matches, _ := filepath.Glob(filepath.Join(s.Artifacts.Workspace.Control, "artifact-scratch", id+"*"))
				if len(matches) > 0 {
					t.Fatal("retained rejected stage", matches)
				}
				stages, _ := filepath.Glob(filepath.Join(s.Artifacts.Workspace.Control, "artifact-scratch", "stage-*"))
				if len(stages) > 0 {
					t.Fatal("retained disposable stage", stages)
				}
			})
		}
	}
	var out Output
	if strict([]byte(`{"unexpected":"fixture"}`), &out) == nil {
		t.Fatal("unknown fields admitted")
	}
}
