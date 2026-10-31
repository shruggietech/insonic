// SPDX-License-Identifier: Apache-2.0
// Package workers prepares optional isolated local engines without model loading.
package workers

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
)

//go:embed assets/* probe.py native.py
var assets embed.FS

type Asset struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size_bytes"`
	Package string `json:"package,omitempty"`
	Version string `json:"version,omitempty"`
}
type Profile struct {
	ID            string            `json:"id"`
	Platform      string            `json:"platform"`
	Python        string            `json:"python"`
	Device        string            `json:"device"`
	Engines       map[string]string `json:"engines"`
	Runtime       Asset             `json:"runtime"`
	Installer     Asset             `json:"installer"`
	Wheels        []Asset           `json:"wheels"`
	Lock          string            `json:"lock"`
	LockSHA256    string            `json:"lock_sha256"`
	DownloadBytes int64             `json:"download_bytes"`
	MinSpaceBytes int64             `json:"min_space_bytes"`
	MinimumGLibc  string            `json:"linux_minimum_glibc,omitempty"`
}
type Catalog struct {
	Kind     string    `json:"kind"`
	Version  string    `json:"schema_version"`
	Profiles []Profile `json:"profiles"`
}

func Inventory() Catalog {
	var c Catalog
	data, _ := assets.ReadFile("assets/catalog.json")
	json.Unmarshal(data, &c)
	return c
}
func Selected(role string) (Profile, error) {
	for _, p := range Inventory().Profiles {
		if p.ID == role && p.Platform == runtime.GOOS+"_"+runtime.GOARCH {
			return p, nil
		}
	}
	return Profile{}, contracts.Fail("unsupported_capability")
}
func (p Profile) Digest() string { data, _ := json.Marshal(p); return Digest(data) }
func Digest(data []byte) string  { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func Hash(ctx context.Context, path string) (library.PinnedFile, error) {
	return hashWithBuffer(ctx, path, nil)
}
func hashWithBuffer(ctx context.Context, path string, buffer []byte) (library.PinnedFile, error) {
	out := library.PinnedFile{Path: path}
	if err := probeContextError(ctx); err != nil {
		return out, err
	}
	if !filepath.IsAbs(path) {
		return out, contracts.Fail("invalid_request")
	}
	f, e := os.Open(path)
	if e != nil {
		return out, contracts.Fail("unavailable")
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		return out, contracts.Fail("invalid_request")
	}
	h := sha256.New()
	b := buffer
	if len(b) == 0 {
		b = make([]byte, 128<<10)
	}
	for {
		if ctx.Err() != nil {
			return out, probeContextError(ctx)
		}
		n, e := f.Read(b)
		h.Write(b[:n])
		if e == io.EOF {
			break
		}
		if e != nil {
			return out, contracts.Fail("unavailable")
		}
	}
	if err := probeContextError(ctx); err != nil {
		return out, err
	}
	out.SHA256 = hex.EncodeToString(h.Sum(nil))
	return out, nil
}

type Device struct {
	Available       bool   `json:"available"`
	HardwareCount   int    `json:"hardware_count"`
	DependencyState string `json:"dependency_state"`
	Correction      string `json:"correction,omitempty"`
}
type Inspection struct {
	Probe           *Probe               `json:"probe,omitempty"`
	Role            string               `json:"role"`
	Python          library.PinnedFile   `json:"python"`
	PythonVersion   string               `json:"python_version"`
	Packages        map[string]string    `json:"packages"`
	Devices         map[string]Device    `json:"devices"`
	CPUs            int                  `json:"cpus"`
	EngineAvailable bool                 `json:"engine_available"`
	Compatible      bool                 `json:"compatible"`
	Error           string               `json:"error,omitempty"`
	Correction      string               `json:"correction,omitempty"`
	Files           []library.PinnedFile `json:"files,omitempty"`
}
type Receipt struct {
	Mode          string     `json:"mode,omitempty"`
	Kind          string     `json:"kind"`
	Version       string     `json:"schema_version"`
	ProfileDigest string     `json:"profile_digest"`
	Inspection    Inspection `json:"inspection"`
}
type Election struct {
	Role             string `json:"role"`
	Mode             string `json:"mode"`
	PythonPath       string `json:"python_path,omitempty"`
	OfflineAssets    string `json:"offline_assets,omitempty"`
	ExpectedRevision string `json:"expected_revision,omitempty"`
	ProfileDigest    string `json:"profile_digest,omitempty"`
}

func (e Election) Validate() error {
	if e.Role != "recognition" && e.Role != "diarization" || e.Mode != "managed" && e.Mode != "reuse" || e.Mode == "reuse" && !filepath.IsAbs(e.PythonPath) || e.Mode == "managed" && e.PythonPath != "" || e.OfflineAssets != "" && !filepath.IsAbs(e.OfflineAssets) || e.Mode == "reuse" && e.OfflineAssets != "" {
		return contracts.Fail("invalid_request")
	}
	return nil
}

type Service struct {
	Directory string
	Run       func(context.Context, process.Spec) (process.CaptureResult, error)
	Fetch     func(context.Context, Asset) (string, error)
	Progress  func(string, map[string]any) error
}
