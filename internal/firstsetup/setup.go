// SPDX-License-Identifier: Apache-2.0
// Package firstsetup owns bounded nonsecret bootstrap and journey contracts.
package firstsetup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/gofrs/flock"
	"github.com/shruggietech/insonic/internal/backendsetup"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workers"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var StageIDs = []string{"workspace", "backends", "processing", "services", "import"}

type Stage struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Proof string `json:"proof,omitempty"`
}
type Choices struct {
	ProcessingMode   string `json:"processing_mode"`
	PipelineID       string `json:"pipeline_id,omitempty"`
	PipelineRevision int64  `json:"pipeline_revision,omitempty"`
}
type State struct {
	Status       string  `json:"status"`
	CurrentStage string  `json:"current_stage"`
	Stages       []Stage `json:"stages"`
	Choices      Choices `json:"choices"`
}
type SaveInput struct {
	ExpectedRevision int64 `json:"expected_revision"`
	State            State `json:"state"`
}

func DefaultState() State {
	s := State{Status: "active", CurrentStage: "processing", Choices: Choices{ProcessingMode: "none"}}
	for _, id := range StageIDs {
		state := "pending"
		if id == "workspace" || id == "backends" {
			state = "completed"
		}
		s.Stages = append(s.Stages, Stage{ID: id, State: state})
	}
	return s
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func ValidateState(s State) error {
	if !contains([]string{"active", "deferred", "completed"}, s.Status) || !contains(StageIDs, s.CurrentStage) || len(s.Stages) != len(StageIDs) || !contains([]string{"none", "local", "connected", "custom"}, s.Choices.ProcessingMode) {
		return contracts.Fail("invalid_request")
	}
	if s.Choices.PipelineID != "" && !contracts.ValidID(s.Choices.PipelineID) || s.Choices.PipelineRevision < 0 || s.Choices.PipelineID == "" && s.Choices.PipelineRevision != 0 || s.Choices.PipelineID != "" && s.Choices.PipelineRevision == 0 {
		return contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	for _, v := range s.Stages {
		if !contains(StageIDs, v.ID) || seen[v.ID] || !contains([]string{"pending", "completed", "skipped"}, v.State) || v.Proof != "" && !validDigest(v.Proof) || s.Status == "completed" && v.State == "pending" {
			return contracts.Fail("invalid_request")
		}
		seen[v.ID] = true
	}
	return nil
}
func validDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, e := hex.DecodeString(s)
	return e == nil && s == strings.ToLower(s)
}
func decode(raw []byte, out any) error {
	if len(raw) > 128<<10 || catalog.ValidateJSON(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	if d.Decode(out) != nil {
		return contracts.Fail("invalid_request")
	}
	return nil
}

type Location struct {
	Path              string            `json:"path"`
	ExistingWorkspace bool              `json:"existing_workspace"`
	ControlDirectory  string            `json:"control_directory"`
	ArtifactDirectory string            `json:"artifact_directory"`
	CacheDirectory    string            `json:"cache_directory"`
	AvailableBytes    *uint64           `json:"available_bytes,omitempty"`
	Writability       string            `json:"writability"`
	Estimates         map[string]string `json:"estimates"`
}

func ProfileStore() (Store, error) {
	paths, e := workspace.PlatformPaths()
	if e != nil {
		return Store{}, e
	}
	return Store{Directory: paths.Config}, nil
}
func Preview(path string) (Location, error) {
	exe, e := os.Executable()
	if e != nil {
		return Location{}, contracts.Fail("unavailable")
	}
	return PreviewAt(path, filepath.Dir(exe))
}
func PreviewAt(path, installation string) (Location, error) {
	paths, e := workspace.PlatformPaths()
	if e != nil {
		return Location{}, e
	}
	if path == "" {
		path = filepath.Join(paths.Data, "libraries", "My voice archive")
	}
	if !filepath.IsAbs(path) || !contracts.ValidTextLength(path, 1, 4096) {
		return Location{}, contracts.Fail("invalid_request")
	}
	path = filepath.Clean(path)
	resolved, e := resolveMissing(path)
	if e != nil {
		return Location{}, contracts.Fail("unavailable")
	}
	install, e := resolveMissing(installation)
	if e != nil {
		return Location{}, contracts.Fail("unavailable")
	}
	compare, target := resolved, install
	if runtime.GOOS == "windows" {
		compare = strings.ToLower(compare)
		target = strings.ToLower(target)
	}
	rel, e := filepath.Rel(target, compare)
	sameVolume := strings.EqualFold(filepath.VolumeName(target), filepath.VolumeName(compare))
	if sameVolume && (e != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return Location{}, &contracts.Error{Code: "invalid_request", Message: "Choose a library directory outside the application installation."}
	}
	st, e := os.Stat(resolved)
	if e == nil && !st.IsDir() {
		return Location{}, contracts.Fail("invalid_request")
	}
	if e != nil && !os.IsNotExist(e) {
		return Location{}, contracts.Fail("unavailable")
	}
	v := Location{Path: resolved, ControlDirectory: filepath.Join(resolved, ".insonic"), ArtifactDirectory: filepath.Join(resolved, ".insonic", "artifacts"), CacheDirectory: paths.Cache, Writability: "not_probed; provisioning checks actual write access", Estimates: map[string]string{"control_catalog": "unknown; grows with accepted metadata and job history", "media_models": "unknown until elected import/model previews", "cache": "disposable; required bytes are reported by operation previews"}}
	if _, e := os.Stat(filepath.Join(resolved, ".insonic", "workspace.json")); e == nil {
		v.ExistingWorkspace = true
	}
	if n, e := workers.FreeSpace(resolved); e == nil {
		v.AvailableBytes = &n
	}
	return v, nil
}
func resolveMissing(path string) (string, error) {
	path, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	tail := []string{}
	for {
		actual, e := filepath.EvalSymlinks(path)
		if e == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				actual = filepath.Join(actual, tail[i])
			}
			return actual, nil
		}
		if !os.IsNotExist(e) {
			return "", e
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", e
		}
		tail = append(tail, filepath.Base(path))
		path = parent
	}
}

type BootstrapInput struct {
	ExpectedRevision  int64           `json:"expected_revision"`
	Path              string          `json:"path"`
	Plan              json.RawMessage `json:"plan"`
	Stage             string          `json:"stage"`
	Status            string          `json:"status"`
	CredentialBackend string          `json:"credential_backend,omitempty"`
}
type bootstrapReceipt struct {
	RequestID string `json:"request_id"`
	Digest    string `json:"digest"`
	Revision  int64  `json:"revision"`
}
type Draft struct {
	Kind              string            `json:"kind"`
	Version           string            `json:"schema_version"`
	Revision          int64             `json:"revision"`
	Path              string            `json:"path"`
	Plan              *workspace.Config `json:"plan"`
	Stage             string            `json:"stage"`
	Status            string            `json:"status"`
	AcceptedRevision  int64             `json:"accepted_revision,omitempty"`
	Location          Location          `json:"location"`
	CredentialBackend string            `json:"credential_backend"`
}
type storedDraft struct {
	Draft    Draft              `json:"draft"`
	Receipts []bootstrapReceipt `json:"receipts"`
}
type Store struct {
	Directory  string
	durability func(string) error
}

func (s Store) read() (storedDraft, error) {
	out := storedDraft{Draft: Draft{Kind: "setup-bootstrap", Version: contracts.Version, Stage: "workspace", Status: "active"}}
	file := filepath.Join(s.Directory, "setup-draft.json")
	stat, e := os.Lstat(file)
	if os.IsNotExist(e) {
		return out, nil
	}
	if e != nil || !stat.Mode().IsRegular() || stat.Size() > 128<<10 {
		return out, contracts.Fail("unavailable")
	}
	raw, e := os.ReadFile(file)
	if e != nil || decode(raw, &out) != nil {
		return out, contracts.Fail("unavailable")
	}
	if out.Draft.Kind != "setup-bootstrap" || out.Draft.Version != contracts.Version || out.Draft.Revision < 1 || len(out.Receipts) > 64 || out.Draft.Plan == nil {
		return out, contracts.Fail("invalid_request")
	}
	if out.Draft.CredentialBackend == "" {
		out.Draft.CredentialBackend = "native"
	}
	if !contains([]string{"native", "vault", "session"}, out.Draft.CredentialBackend) {
		return out, contracts.Fail("invalid_request")
	}
	planned, e := json.Marshal(out.Draft.Plan)
	if e != nil {
		return out, contracts.Fail("invalid_request")
	}
	if e = schemas.ValidateWorkspace(planned); e != nil {
		return out, contracts.Fail("invalid_request")
	}
	normalized, e := backendsetup.Build(planned)
	if e != nil {
		return out, e
	}
	verified, _ := json.Marshal(normalized)
	if !bytes.Equal(planned, verified) {
		return out, contracts.Fail("invalid_request")
	}
	if !filepath.IsAbs(out.Draft.Path) || !contains([]string{"workspace", "backends"}, out.Draft.Stage) || !contains([]string{"active", "deferred"}, out.Draft.Status) {
		return out, contracts.Fail("invalid_request")
	}
	for _, r := range out.Receipts {
		if !contracts.ValidID(r.RequestID) || !validDigest(r.Digest) || r.Revision < 1 || r.Revision > out.Draft.Revision {
			return out, contracts.Fail("invalid_request")
		}
	}
	return out, nil
}
func (s Store) Show() (Draft, error) {
	v, e := s.read()
	if e != nil {
		return Draft{}, e
	}
	v.Draft.Location, e = Preview(v.Draft.Path)
	if e == nil && v.Draft.Path == "" {
		v.Draft.Path = v.Draft.Location.Path
	}
	v.Draft.AcceptedRevision = 0
	return v.Draft, e
}
func (s Store) Save(ctx context.Context, id string, raw []byte) (Draft, error) {
	var in BootstrapInput
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields["expected_revision"]) == 0 || bytes.Equal(bytes.TrimSpace(fields["expected_revision"]), []byte("null")) {
		return Draft{}, contracts.Fail("invalid_request")
	}
	if !contracts.ValidID(id) || decode(raw, &in) != nil || in.ExpectedRevision < 0 || in.ExpectedRevision == math.MaxInt64 || !contains([]string{"workspace", "backends"}, in.Stage) || !contains([]string{"active", "deferred"}, in.Status) {
		return Draft{}, contracts.Fail("invalid_request")
	}
	if in.CredentialBackend != "" && !contains([]string{"native", "vault", "session"}, in.CredentialBackend) {
		return Draft{}, contracts.Fail("invalid_request")
	}
	location, e := Preview(in.Path)
	if e != nil {
		return Draft{}, e
	}
	if in.Path == "" {
		return Draft{}, contracts.Fail("invalid_request")
	}
	var value any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	d.Decode(&value)
	normalized, _ := json.Marshal(value)
	sum := sha256.Sum256(normalized)
	digest := hex.EncodeToString(sum[:])
	if _, e = backendsetup.Build(in.Plan); e != nil {
		return Draft{}, e
	}
	if !filepath.IsAbs(s.Directory) {
		return Draft{}, contracts.Fail("invalid_request")
	}
	_, e = os.Stat(s.Directory)
	created := os.IsNotExist(e)
	if e = os.MkdirAll(s.Directory, 0700); e != nil {
		return Draft{}, contracts.Fail("unavailable")
	}
	if e = workspace.SecureDirectory(s.Directory, created); e != nil {
		return Draft{}, e
	}
	lock := flock.New(filepath.Join(s.Directory, "setup-draft.lock"))
	bound, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ok, e := lock.TryLockContext(bound, 20*time.Millisecond)
	if e != nil || !ok {
		return Draft{}, contracts.Fail("unavailable")
	}
	defer lock.Close()
	current, e := s.read()
	if e != nil {
		return Draft{}, e
	}
	for _, r := range current.Receipts {
		if r.RequestID == id {
			if r.Digest != digest {
				return Draft{}, contracts.Fail("conflict")
			}
			if e = s.persistPublication(); e != nil {
				return Draft{}, contracts.Fail("unavailable")
			}
			current.Draft.AcceptedRevision = r.Revision
			current.Draft.Location, e = Preview(current.Draft.Path)
			return current.Draft, e
		}
	}
	if current.Draft.Revision != in.ExpectedRevision || current.Draft.Revision == math.MaxInt64 {
		return Draft{}, contracts.Fail("conflict")
	}
	var requested workspace.Config
	if decode(in.Plan, &requested) != nil {
		return Draft{}, contracts.Fail("invalid_request")
	}
	if current.Draft.Plan != nil && current.Draft.Path == location.Path {
		prior := current.Draft.Plan
		if requested.WorkspaceID == "" {
			requested.WorkspaceID = prior.WorkspaceID
		}
		if requested.Profiles.Storage.ID == "" {
			requested.Profiles.Storage.ID = prior.Profiles.Storage.ID
		}
		if requested.Profiles.Catalog.ID == "" {
			requested.Profiles.Catalog.ID = prior.Profiles.Catalog.ID
		}
		if requested.Profiles.Graph.ID == "" {
			requested.Profiles.Graph.ID = prior.Profiles.Graph.ID
		}
	}
	if current.Draft.Plan != nil && current.Draft.Path != location.Path {
		if _, e := os.Stat(filepath.Join(current.Draft.Path, ".insonic", "workspace.json")); e == nil && requested.WorkspaceID == current.Draft.Plan.WorkspaceID {
			requested.WorkspaceID = ""
			requested.CredentialNamespaceID = ""
			requested.Profiles.Storage.ID = ""
			requested.Profiles.Catalog.ID = ""
			requested.Profiles.Graph.ID = ""
			requested.Profiles.Storage.Revision = 1
			requested.Profiles.Catalog.Revision = 1
			requested.Profiles.Graph.Revision = 1
		}
	}
	planned, _ := json.Marshal(requested)
	configuration, e := backendsetup.Build(planned)
	if e != nil {
		return Draft{}, e
	}
	revision := current.Draft.Revision + 1
	out := Draft{Kind: "setup-bootstrap", Version: contracts.Version, Revision: revision, Path: location.Path, Plan: &configuration, Stage: in.Stage, Status: in.Status, Location: location}
	out.CredentialBackend = in.CredentialBackend
	if out.CredentialBackend == "" {
		out.CredentialBackend = current.Draft.CredentialBackend
		if out.CredentialBackend == "" {
			out.CredentialBackend = "native"
		}
	}
	receipts := append(current.Receipts, bootstrapReceipt{id, digest, revision})
	if len(receipts) > 64 {
		receipts = receipts[len(receipts)-64:]
	}
	stored := storedDraft{Draft: out, Receipts: receipts}
	data, e := json.Marshal(stored)
	if e != nil || len(data) > 128<<10 {
		return Draft{}, contracts.Fail("input_limit")
	}
	file, e := os.CreateTemp(s.Directory, ".setup-*")
	if e != nil {
		return Draft{}, contracts.Fail("unavailable")
	}
	name := file.Name()
	defer os.Remove(name)
	if e = file.Chmod(0600); e == nil {
		_, e = io.Copy(file, bytes.NewReader(append(data, '\n')))
	}
	if e == nil {
		e = file.Sync()
	}
	closeErr := file.Close()
	if e == nil {
		e = closeErr
	}
	if e == nil {
		e = replaceDraft(name, filepath.Join(s.Directory, "setup-draft.json"))
	}
	if e != nil {
		return Draft{}, contracts.Fail("unavailable")
	}
	if e = s.persistPublication(); e != nil {
		return Draft{}, contracts.Fail("unavailable")
	}
	out.AcceptedRevision = revision
	return out, nil
}

func (s Store) persistPublication() error {
	if s.durability != nil {
		return s.durability(s.Directory)
	}
	return syncPublication(s.Directory)
}
