// SPDX-License-Identifier: Apache-2.0
package workspace

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/gofrs/flock"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Profile struct {
	ID                     string         `json:"profile_id"`
	Revision               int64          `json:"profile_revision"`
	Adapter                string         `json:"adapter_id"`
	Version                string         `json:"contract_version"`
	Configuration          map[string]any `json:"configuration"`
	ExpectedBackendVersion string         `json:"expected_backend_version,omitempty"`
}
type Config struct {
	Version               string `json:"schema_version"`
	Kind                  string `json:"kind"`
	WorkspaceID           string `json:"workspace_id"`
	CredentialNamespaceID string `json:"credential_namespace_id,omitempty"`
	DisplayName           string `json:"display_name"`
	ControlDirectory      string `json:"control_directory"`
	Profiles              struct {
		Storage Profile `json:"storage"`
		Catalog Profile `json:"catalog"`
		Graph   Profile `json:"graph"`
	} `json:"profiles"`
}
type Workspace struct {
	Root    string
	Control string
	Config  Config
}

// CredentialNamespace is independent of portable catalog identity. Restores
// preserve the destination's configured credential store without exporting it.
func (w *Workspace) CredentialNamespace() string {
	if w.Config.CredentialNamespaceID != "" {
		return w.Config.CredentialNamespaceID
	}
	return w.Config.WorkspaceID
}

type Paths struct {
	Config  string `json:"config"`
	Data    string `json:"data"`
	Cache   string `json:"cache"`
	Runtime string `json:"runtime"`
}

func PlatformPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, contracts.Fail("unavailable")
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, contracts.Fail("unavailable")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, contracts.Fail("unavailable")
	}
	data := filepath.Join(home, ".local", "share", "insonic")
	if runtime.GOOS == "windows" {
		data = filepath.Join(os.Getenv("LOCALAPPDATA"), "insonic")
	}
	if runtime.GOOS == "darwin" {
		data = filepath.Join(home, "Library", "Application Support", "insonic")
	}
	if runtime.GOOS == "linux" && filepath.IsAbs(os.Getenv("XDG_DATA_HOME")) {
		data = filepath.Join(os.Getenv("XDG_DATA_HOME"), "insonic")
	}
	return Paths{filepath.Join(config, "insonic"), data, filepath.Join(cache, "insonic"), filepath.Join(cache, "insonic", "runtime")}, nil
}

func canonical(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", contracts.Fail("invalid_request")
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", contracts.Fail("unavailable")
	}
	return resolved, nil
}

func Init(root, name string) (*Workspace, error) {
	if name == "" {
		return nil, contracts.Fail("invalid_request")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, contracts.Fail("unavailable")
	}
	root, err := canonical(root)
	if err != nil {
		return nil, err
	}
	control := filepath.Join(root, ".insonic")
	if _, err := os.Lstat(filepath.Join(control, "workspace.json")); !os.IsNotExist(err) {
		return nil, contracts.Fail("conflict")
	}
	created := false
	if err := os.Mkdir(control, 0700); err == nil {
		created = true
	} else if !os.IsExist(err) {
		return nil, contracts.Fail("unavailable")
	}
	if err := SecureDirectory(control, created); err != nil {
		return nil, err
	}
	lock := flock.New(filepath.Join(control, "initialize.lock"))
	if err := lock.Lock(); err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer lock.Unlock()
	file := filepath.Join(control, "workspace.json")
	if _, err := os.Lstat(file); !os.IsNotExist(err) {
		return nil, contracts.Fail("conflict")
	}
	config := Config{Version: contracts.Version, Kind: "workspace-config", WorkspaceID: contracts.ID(), DisplayName: name, ControlDirectory: ".insonic"}
	profile := func(adapter string, options map[string]any) Profile {
		return Profile{ID: contracts.ID(), Revision: 1, Adapter: adapter, Version: contracts.Version, Configuration: options}
	}
	config.Profiles.Storage = profile("filesystem", map[string]any{"root": "artifacts"})
	config.Profiles.Catalog = profile("sqlite", map[string]any{"path": "catalog.sqlite"})
	config.Profiles.Graph = profile("ladybugdb", map[string]any{"path": "graph/library.lbug"})
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if err := schemas.ValidateWorkspace(data); err != nil {
		return nil, contracts.Fail("invalid_request")
	}
	tmp, err := os.CreateTemp(control, ".workspace-*")
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return nil, contracts.Fail("unavailable")
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return nil, contracts.Fail("unavailable")
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return nil, contracts.Fail("unavailable")
	}
	if err := tmp.Close(); err != nil {
		return nil, contracts.Fail("unavailable")
	}
	// Linking provides atomic no-replace publication even against noncooperating writers.
	if err := os.Link(tmp.Name(), file); err != nil {
		return nil, contracts.Fail("conflict")
	}
	return Open(root)
}

func Open(root string) (*Workspace, error) {
	root, err := canonical(root)
	if err != nil {
		return nil, err
	}
	control := filepath.Join(root, ".insonic")
	if err := PrivateDirectory(control); err != nil {
		return nil, err
	}
	file := filepath.Join(control, "workspace.json")
	stat, err := os.Lstat(file)
	if err != nil || !stat.Mode().IsRegular() || stat.Size() > 1<<20 {
		return nil, contracts.Fail("invalid_request")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	var config Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&config); err != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if config.Version != contracts.Version {
		return nil, contracts.Fail("incompatible_version")
	}
	if err := schemas.ValidateWorkspace(data); err != nil {
		return nil, contracts.Fail("invalid_request")
	}
	selected := config.ControlDirectory
	if !filepath.IsAbs(selected) {
		selected = filepath.Join(root, selected)
	}
	selected, err = canonical(selected)
	if err != nil {
		return nil, err
	}
	if err := PrivateDirectory(selected); err != nil {
		return nil, err
	}
	control = selected
	return &Workspace{root, control, config}, nil
}

func Discover(start string) (*Workspace, error) {
	root, err := canonical(start)
	if err != nil {
		return nil, err
	}
	for {
		if _, err := os.Lstat(filepath.Join(root, ".insonic", "workspace.json")); err == nil {
			return Open(root)
		}
		parent := filepath.Dir(root)
		if parent == root {
			return nil, contracts.Fail("not_found")
		}
		root = parent
	}
}

func RuntimeDirectory() (string, error) {
	paths, err := PlatformPaths()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(paths.Cache, 0700); err != nil {
		return "", contracts.Fail("unavailable")
	}
	created := false
	if err := os.Mkdir(paths.Runtime, 0700); err == nil {
		created = true
	} else if !os.IsExist(err) {
		return "", contracts.Fail("unavailable")
	}
	if err := SecureDirectory(paths.Runtime, created); err != nil {
		return "", err
	}
	return paths.Runtime, nil
}

func (w *Workspace) Lock() (*flock.Flock, error) {
	directory, err := RuntimeDirectory()
	if err != nil {
		return nil, err
	}
	root := w.Root
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
	}
	hash := sha256.Sum256([]byte(root))
	// Runtime authority survives control-directory edits and read-only metadata.
	lock := flock.New(filepath.Join(directory, fmt.Sprintf("%x.lock", hash[:16])))
	ok, err := lock.TryLock()
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	if !ok {
		lock.Close()
		return nil, contracts.Fail("conflict")
	}
	return lock, nil
}
