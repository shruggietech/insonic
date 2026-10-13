// SPDX-License-Identifier: Apache-2.0
package firstsetup

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func privateStoreDirectory(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "private-config")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	return dir
}

func draftInput(t *testing.T, root string, revision int64) []byte {
	t.Helper()
	raw, _ := json.Marshal(BootstrapInput{ExpectedRevision: revision, Path: root, Plan: json.RawMessage(`{"kind":"backend-plan","display_name":"Fixture"}`), Stage: "workspace", Status: "active"})
	return raw
}
func TestBootstrapRestartReplayAndStaleWrites(t *testing.T) {
	dir := privateStoreDirectory(t)
	target := filepath.Join(t.TempDir(), "library with spaces")
	s := Store{Directory: dir}
	id := contracts.ID()
	raw := draftInput(t, target, 0)
	first, e := s.Save(context.Background(), id, raw)
	if e != nil {
		t.Fatal(e)
	}
	if first.Revision != 1 || first.Plan == nil {
		t.Fatalf("%+v", first)
	}
	reopened := Store{Directory: dir}
	again, e := reopened.Save(context.Background(), id, raw)
	if e != nil || again.Plan.WorkspaceID != first.Plan.WorkspaceID || again.Revision != 1 {
		t.Fatalf("replay %+v %v", again, e)
	}
	newer, e := reopened.Save(context.Background(), contracts.ID(), draftInput(t, target, 1))
	if e != nil {
		t.Fatal(e)
	}
	replay, e := reopened.Save(context.Background(), id, raw)
	if e != nil || replay.AcceptedRevision != 1 || replay.Revision != newer.Revision {
		t.Fatalf("reconciled replay %+v %v", replay, e)
	}
	if _, e = s.Save(context.Background(), contracts.ID(), raw); e == nil {
		t.Fatal("stale write accepted")
	}
	changed := draftInput(t, filepath.Join(t.TempDir(), "different"), 0)
	if _, e = s.Save(context.Background(), id, changed); e == nil {
		t.Fatal("changed request reuse accepted")
	}
	if _, e = os.Stat(target); !os.IsNotExist(e) {
		t.Fatal("draft created library")
	}
}
func TestBootstrapRejectsSecretsAndUnknownFields(t *testing.T) {
	s := Store{Directory: privateStoreDirectory(t)}
	target := filepath.Join(t.TempDir(), "library")
	raw := draftInput(t, target, 0)
	var value map[string]any
	json.Unmarshal(raw, &value)
	value["password"] = "SECRET_SENTINEL"
	bad, _ := json.Marshal(value)
	if _, e := s.Save(context.Background(), contracts.ID(), bad); e == nil {
		t.Fatal("unknown secret accepted")
	}
	value["password"] = nil
	delete(value, "password")
	value["plan"] = map[string]any{"display_name": "Fixture", "password": "SECRET_SENTINEL"}
	bad, _ = json.Marshal(value)
	if _, e := s.Save(context.Background(), contracts.ID(), bad); e == nil {
		t.Fatal("plan secret accepted")
	}
	if _, e := os.Stat(filepath.Join(s.Directory, "setup-draft.json")); !os.IsNotExist(e) {
		t.Fatal("invalid draft persisted")
	}
}
func TestLocationPreviewAndInstallationContainment(t *testing.T) {
	install := t.TempDir()
	if _, e := PreviewAt(filepath.Join(install, "library"), install); e == nil {
		t.Fatal("installation accepted")
	}
	target := filepath.Join(t.TempDir(), "new", "library")
	v, e := PreviewAt(target, install)
	if e != nil || !strings.HasSuffix(v.Path, filepath.Join("new", "library")) || v.ExistingWorkspace {
		t.Fatalf("%+v %v", v, e)
	}
	if _, e = os.Stat(target); !os.IsNotExist(e) {
		t.Fatal("preview mutated destination")
	}
}
func TestTypedStateLimits(t *testing.T) {
	s := DefaultState()
	if e := ValidateState(s); e != nil {
		t.Fatal(e)
	}
	s.Stages = append(s.Stages, s.Stages[0])
	if ValidateState(s) == nil {
		t.Fatal("duplicate stage accepted")
	}
	s = DefaultState()
	s.Choices.PipelineID = "not-id"
	if ValidateState(s) == nil {
		t.Fatal("invalid pipeline accepted")
	}
	s = DefaultState()
	s.Status = "completed"
	if ValidateState(s) == nil {
		t.Fatal("pending completion accepted")
	}
}

func TestBootstrapMissingRevisionAndCorruptReadback(t *testing.T) {
	s := Store{Directory: privateStoreDirectory(t)}
	target := filepath.Join(t.TempDir(), "library")
	var value map[string]any
	json.Unmarshal(draftInput(t, target, 0), &value)
	delete(value, "expected_revision")
	raw, _ := json.Marshal(value)
	if _, e := s.Save(context.Background(), contracts.ID(), raw); e == nil {
		t.Fatal("missing expected revision accepted")
	}
	value["expected_revision"] = nil
	raw, _ = json.Marshal(value)
	if _, e := s.Save(context.Background(), contracts.ID(), raw); e == nil {
		t.Fatal("null expected revision accepted")
	}
	v, e := s.Save(context.Background(), contracts.ID(), draftInput(t, target, 0))
	if e != nil {
		t.Fatal(e)
	}
	v.Plan.Profiles.Storage.Configuration["password"] = "SECRET_SENTINEL"
	corrupt, _ := json.Marshal(storedDraft{Draft: v})
	os.WriteFile(filepath.Join(s.Directory, "setup-draft.json"), corrupt, 0600)
	if _, e = s.Show(); e == nil {
		t.Fatal("corrupt plan exposed")
	}
}
func TestBootstrapConcurrentWritersPreserveOneAcceptance(t *testing.T) {
	s := Store{Directory: privateStoreDirectory(t)}
	raw := draftInput(t, filepath.Join(t.TempDir(), "library"), 0)
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, e := s.Save(context.Background(), contracts.ID(), raw); results <- e }()
	}
	accepted := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted %d writes", accepted)
	}
	v, e := s.Show()
	if e != nil || v.Revision != 1 {
		t.Fatalf("%+v %v", v, e)
	}
}

func TestLocationOnDifferentWindowsVolume(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows volumes")
	}
	cwd, e := os.Getwd()
	if e != nil {
		t.Fatal(e)
	}
	installation := t.TempDir()
	if strings.EqualFold(filepath.VolumeName(cwd), filepath.VolumeName(installation)) {
		t.Skip("fixture needs two available volumes")
	}
	target := filepath.Join(cwd, "preview-only-library")
	if _, e := PreviewAt(target, installation); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(target); !os.IsNotExist(e) {
		t.Fatal("preview mutated destination")
	}
}

func TestBootstrapReadRequiresNormalizedFrozenPlan(t *testing.T) {
	for _, field := range []string{"workspace_id", "schema_version", "profiles"} {
		t.Run(field, func(t *testing.T) {
			s := Store{Directory: privateStoreDirectory(t)}
			_, e := s.Save(context.Background(), contracts.ID(), draftInput(t, filepath.Join(t.TempDir(), "library"), 0))
			if e != nil {
				t.Fatal(e)
			}
			file := filepath.Join(s.Directory, "setup-draft.json")
			raw, _ := os.ReadFile(file)
			var value map[string]any
			json.Unmarshal(raw, &value)
			plan := value["draft"].(map[string]any)["plan"].(map[string]any)
			delete(plan, field)
			corrupt, _ := json.Marshal(value)
			os.WriteFile(file, corrupt, 0600)
			if _, e = s.Show(); e == nil {
				t.Fatal("incomplete authority accepted")
			}
		})
	}
}

func TestBootstrapRetainsNonsecretCredentialStoreChoice(t *testing.T) {
	s := Store{Directory: privateStoreDirectory(t)}
	target := filepath.Join(t.TempDir(), "library")
	var in BootstrapInput
	json.Unmarshal(draftInput(t, target, 0), &in)
	in.CredentialBackend = "vault"
	raw, _ := json.Marshal(in)
	v, e := s.Save(context.Background(), contracts.ID(), raw)
	if e != nil || v.CredentialBackend != "vault" {
		t.Fatalf("%+v %v", v, e)
	}
	again, e := s.Show()
	if e != nil || again.CredentialBackend != "vault" {
		t.Fatalf("%+v %v", again, e)
	}
	v, e = s.Save(context.Background(), contracts.ID(), draftInput(t, target, v.Revision))
	if e != nil || v.CredentialBackend != "vault" {
		t.Fatalf("omitted choice %+v %v", v, e)
	}
	in.CredentialBackend = "invalid"
	raw, _ = json.Marshal(in)
	if _, e = s.Save(context.Background(), contracts.ID(), raw); e == nil {
		t.Fatal("unknown credential store accepted")
	}
}

func TestNewLocationDoesNotCloneProvisionedLibraryIdentity(t *testing.T) {
	s := Store{Directory: privateStoreDirectory(t)}
	old := filepath.Join(t.TempDir(), "old library")
	first, e := s.Save(context.Background(), contracts.ID(), draftInput(t, old, 0))
	if e != nil {
		t.Fatal(e)
	}
	w, e := workspace.InitConfig(old, *first.Plan)
	if e != nil {
		t.Fatal(e)
	}
	planned, _ := json.Marshal(first.Plan)
	next := BootstrapInput{ExpectedRevision: first.Revision, Path: filepath.Join(t.TempDir(), "new library"), Plan: planned, Stage: "workspace", Status: "active"}
	raw, _ := json.Marshal(next)
	v, e := s.Save(context.Background(), contracts.ID(), raw)
	if e != nil {
		t.Fatal(e)
	}
	if v.Plan.WorkspaceID == w.Config.WorkspaceID || v.Plan.Profiles.Storage.ID == w.Config.Profiles.Storage.ID {
		t.Fatal("cloned accepted identity")
	}
	unchanged, e := workspace.Open(old)
	if e != nil || unchanged.Config.WorkspaceID != w.Config.WorkspaceID {
		t.Fatal("source identity changed")
	}
}

func TestBootstrapRejectsNonPrivateDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode fixture")
	}
	dir := filepath.Join(t.TempDir(), "public-config")
	if e := os.Mkdir(dir, 0755); e != nil {
		t.Fatal(e)
	}
	s := Store{Directory: dir}
	if _, e := s.Save(context.Background(), contracts.ID(), draftInput(t, filepath.Join(t.TempDir(), "library"), 0)); e == nil {
		t.Fatal("nonprivate bootstrap store accepted")
	}
	if _, e := os.Stat(filepath.Join(dir, "setup-draft.json")); !os.IsNotExist(e) {
		t.Fatal("rejected store persisted data")
	}
}
