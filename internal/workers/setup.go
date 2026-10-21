// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
)

type Prepared struct {
	Python     library.PinnedFile `json:"python"`
	Receipt    library.PinnedFile `json:"environment_receipt"`
	Inspection Inspection         `json:"inspection"`
}

// Reuse writes only a bounded receipt, private probe home and settings bytes.
const ReuseSpaceBytes int64 = 32 << 20

func RequiredSpace(p Profile, mode string) int64 {
	if mode == "reuse" {
		return ReuseSpaceBytes
	}
	return p.MinSpaceBytes
}

// Hash the bytes copied into this attempt, rather than trusting an external
// path that can be replaced between verification and extraction.
func snapshotAsset(ctx context.Context, source, destination string, asset Asset) error {
	in, e := os.Open(source)
	if e != nil {
		return contracts.AssetOpenFailure(ctx, e, assetRole(asset), asset.Name)
	}
	defer in.Close()
	st, e := in.Stat()
	if e != nil {
		return contracts.AssetFailure("asset_unreadable", assetRole(asset), asset.Name)
	}
	if !st.Mode().IsRegular() {
		return contracts.AssetFailure("asset_not_regular", assetRole(asset), asset.Name)
	}
	if st.Size() != asset.Size {
		return contracts.AssetFailure("asset_size", assetRole(asset), asset.Name)
	}
	out, e := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer out.Close()
	h := sha256.New()
	buffer := make([]byte, 128<<10)
	var total int64
	for {
		if ctx.Err() != nil {
			return contracts.Fail("cancelled")
		}
		n, readErr := in.Read(buffer)
		if total+int64(n) > asset.Size {
			return contracts.AssetFailure("asset_size", assetRole(asset), asset.Name)
		}
		if n > 0 {
			if _, e = out.Write(buffer[:n]); e != nil {
				return contracts.Fail("unavailable")
			}
			h.Write(buffer[:n])
			total += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return contracts.AssetFailure("asset_unreadable", assetRole(asset), asset.Name)
		}
	}
	if total != asset.Size {
		return contracts.AssetFailure("asset_size", assetRole(asset), asset.Name)
	}
	if hex.EncodeToString(h.Sum(nil)) != asset.SHA256 {
		return contracts.AssetFailure("asset_hash", assetRole(asset), asset.Name)
	}
	if out.Sync() != nil {
		return contracts.Fail("unavailable")
	}
	return nil
}

func (s Service) checkpoint(phase string, data map[string]any) error {
	if s.Progress != nil {
		return s.Progress(phase, data)
	}
	return nil
}
func assetRole(a Asset) string {
	if a.Package != "" || strings.HasSuffix(a.Name, ".whl") {
		return "worker_wheel"
	}
	if strings.HasPrefix(a.Name, "uv-") {
		return "worker_installer"
	}
	return "worker_runtime"
}
func verifyAsset(ctx context.Context, source string, a Asset) error {
	file, e := os.Open(source)
	if e != nil {
		return contracts.AssetOpenFailure(ctx, e, assetRole(a), a.Name)
	}
	defer file.Close()
	return contracts.VerifyAssetFile(ctx, file, assetRole(a), a.Name, a.SHA256, a.Size)
}
func (s Service) asset(ctx context.Context, a Asset, offline string) (string, error) {
	if a.Name == "" || path.Base(a.Name) != a.Name || strings.ContainsAny(a.Name, `\:`) {
		return "", contracts.Fail("invalid_request")
	}
	if offline != "" {
		p := filepath.Join(offline, a.Name)
		return p, verifyAsset(ctx, p, a)
	}
	if s.Fetch == nil {
		return "", contracts.Fail("unavailable")
	}
	p, e := s.Fetch(ctx, a)
	if e != nil {
		return "", e
	}
	return p, verifyAsset(ctx, p, a)
}
func (s Service) Prepare(ctx context.Context, election Election) (Prepared, error) {
	var out Prepared
	if e := election.Validate(); e != nil {
		return out, e
	}
	p, e := Selected(election.Role)
	if e != nil {
		return out, e
	}
	if election.ProfileDigest != "" && election.ProfileDigest != p.Digest() {
		return out, contracts.Fail("incompatible_version")
	}
	if election.Mode == "managed" {
		if _, e = s.PlatformPrerequisite(ctx, p); e != nil {
			return out, e
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if e = os.MkdirAll(s.Directory, 0700); e != nil {
		return out, contracts.Fail("unavailable")
	}
	// Each attempt owns a fresh directory. Accepted environments are immutable.
	if e = workspace.SecureDirectory(s.Directory, false); e != nil {
		return out, e
	}
	attempt, e := os.MkdirTemp(s.Directory, "environment-")
	if e != nil {
		return out, contracts.Fail("unavailable")
	}
	if e = workspace.SecureDirectory(attempt, true); e != nil {
		return out, e
	}
	accepted := false
	defer func() {
		if !accepted {
			// The freshly created attempt is the only removal target. Root-relative
			// cleanup cannot traverse outside the verified worker directory.
			if root, err := os.OpenRoot(s.Directory); err == nil {
				root.RemoveAll(filepath.Base(attempt))
				root.Close()
			}
		}
	}()
	python := election.PythonPath
	free, e := FreeSpace(attempt)
	if e != nil {
		return out, contracts.Fail("unavailable")
	}
	if free < uint64(RequiredSpace(p, election.Mode)) {
		return out, contracts.Fail("insufficient_space")
	}
	if election.Mode == "managed" {
		if e = s.checkpoint("runtime-acquisition", map[string]any{"role": election.Role, "total_bytes": p.DownloadBytes}); e != nil {
			return out, e
		}
		runtimeSource, e := s.asset(ctx, p.Runtime, election.OfflineAssets)
		if e != nil {
			return out, e
		}
		runtimeSnapshot := filepath.Join(attempt, "runtime-archive")
		if e = snapshotAsset(ctx, runtimeSource, runtimeSnapshot, p.Runtime); e != nil {
			return out, e
		}
		installerSource, e := s.asset(ctx, p.Installer, election.OfflineAssets)
		if e != nil {
			return out, e
		}
		installerSnapshot := filepath.Join(attempt, "installer-archive")
		if e = snapshotAsset(ctx, installerSource, installerSnapshot, p.Installer); e != nil {
			return out, e
		}
		installDir := filepath.Join(attempt, "installer")
		runtimeDir := filepath.Join(attempt, "runtime")
		os.Mkdir(installDir, 0700)
		os.Mkdir(runtimeDir, 0700)
		format := "tar.gz"
		if runtime.GOOS == "windows" {
			format = "zip"
		}
		if e = extractArchive(ctx, installerSnapshot, installDir, format); e != nil {
			return out, e
		}
		if e = extractArchive(ctx, runtimeSnapshot, runtimeDir, "tar.gz"); e != nil {
			return out, e
		}
		uv, e := findExecutable(installDir, "uv")
		if e != nil {
			return out, e
		}
		basePython := filepath.Join(runtimeDir, "python", "bin", "python3.12")
		if runtime.GOOS == "windows" {
			basePython = filepath.Join(runtimeDir, "python", "python.exe")
		}
		wheelDir := filepath.Join(attempt, "wheels")
		os.Mkdir(wheelDir, 0700)
		var downloaded int64
		for _, a := range p.Wheels {
			if e = s.checkpoint("wheel-acquisition", map[string]any{"role": election.Role, "current_package": a.Package, "downloaded_bytes": downloaded, "total_bytes": p.DownloadBytes}); e != nil {
				return out, e
			}
			source, e := s.asset(ctx, a, election.OfflineAssets)
			if e != nil {
				return out, e
			}
			if e = snapshotAsset(ctx, source, filepath.Join(wheelDir, a.Name), a); e != nil {
				return out, e
			}
			downloaded += a.Size
		}
		lock, e := assets.ReadFile("assets/" + p.Lock)
		if e != nil || Digest(lock) != p.LockSHA256 {
			return out, contracts.Fail("checksum_mismatch")
		}
		lockPath := filepath.Join(attempt, "requirements.txt")
		if os.WriteFile(lockPath, lock, 0600) != nil {
			return out, contracts.Fail("unavailable")
		}
		home := filepath.Join(attempt, "home")
		cache := filepath.Join(attempt, "cache")
		os.Mkdir(home, 0700)
		os.Mkdir(cache, 0700)
		env := append(process.LocalEnvironment(), "HOME="+home, "USERPROFILE="+home, "UV_CACHE_DIR="+cache, "UV_PYTHON_DOWNLOADS=never", "UV_NO_CONFIG=1", "UV_NO_PROGRESS=1", "PIP_NO_INPUT=1")
		run := s.Run
		if run == nil {
			run = process.Capture
		}
		venv := filepath.Join(attempt, "venv")
		if e = s.checkpoint("creating-environment", map[string]any{"role": election.Role}); e != nil {
			return out, e
		}
		for _, args := range [][]string{{"--offline", "--no-config", "--no-progress", "venv", "--python", basePython, venv}, {"--offline", "--no-config", "--no-progress", "pip", "sync", "--python", pythonIn(venv), "--require-hashes", "--only-binary", ":all:", "--no-index", "--find-links", wheelDir, lockPath}} {
			if _, e = run(ctx, process.Spec{Executable: uv, Args: args, Directory: attempt, CleanEnv: true, Env: env, MaxOutput: 1 << 20}); e != nil {
				return out, e
			}
		}
		python = pythonIn(venv)
	}
	if e = s.checkpoint("verifying-environment", map[string]any{"role": election.Role}); e != nil {
		return out, e
	}
	inspection, e := s.Inspect(ctx, election.Role, python, true)
	if e != nil {
		return out, e
	}
	if !inspection.Compatible || !inspection.EngineAvailable || len(inspection.Files) == 0 {
		return out, contracts.Fail("engine_unavailable")
	}
	if election.Mode == "managed" {
		if inspection.PythonVersion != p.Python {
			return out, contracts.Fail("incompatible_version")
		}
		for name, v := range p.Engines {
			if inspection.Packages[name] != v {
				return out, contracts.Fail("incompatible_version")
			}
		}
	}
	r := Receipt{Kind: "worker-environment", Version: contracts.Version, Mode: election.Mode, ProfileDigest: p.Digest(), Inspection: inspection}
	raw, _ := json.Marshal(r)
	if len(raw) > 16<<20 {
		return out, contracts.Fail("output_limit")
	}
	receiptPath := filepath.Join(attempt, "environment.json")
	if e = os.WriteFile(receiptPath, append(raw, '\n'), 0600); e != nil {
		return out, contracts.Fail("unavailable")
	}
	pin, e := Hash(ctx, receiptPath)
	if e != nil {
		return out, e
	}
	if e = VerifyReceipt(ctx, pin, inspection.Python); e != nil {
		return out, e
	}
	inspection.Files = nil
	accepted = true
	return Prepared{Python: inspection.Python, Receipt: pin, Inspection: inspection}, nil
}
func pythonIn(dir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, "Scripts", "python.exe")
	}
	return filepath.Join(dir, "bin", "python")
}
func findExecutable(dir, name string) (string, error) {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var found string
	e := filepath.WalkDir(dir, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() && d.Name() == name {
			if found != "" {
				return contracts.Fail("conflict")
			}
			found = p
		}
		return nil
	})
	if e != nil || found == "" {
		return "", contracts.AssetFailure("asset_tool_missing", "worker_installer", "uv")
	}
	return found, nil
}

func archiveName(name string) (string, error) {
	if strings.ContainsAny(name, `\:`) || strings.HasPrefix(name, "/") || !filepath.IsLocal(filepath.FromSlash(path.Clean(name))) {
		return "", contracts.Fail("invalid_request")
	}
	return path.Clean(name), nil
}
func extractArchive(ctx context.Context, source, directory, format string) error {
	root, e := os.OpenRoot(directory)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer root.Close()
	var expanded int64
	seen := map[string]bool{}
	put := func(name string, size int64, mode os.FileMode, reader io.Reader) error {
		if ctx.Err() != nil {
			return contracts.Fail("cancelled")
		}
		n, e := archiveName(name)
		if e != nil {
			return e
		}
		key := n
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if size < 0 || size > 1<<30 || expanded+size > 2<<30 || seen[key] {
			return contracts.Fail("input_limit")
		}
		seen[key] = true
		expanded += size
		if e = root.MkdirAll(path.Dir(n), 0700); e != nil {
			return contracts.Fail("unavailable")
		}
		f, e := root.OpenFile(n, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm()&0700)
		if e != nil {
			return contracts.Fail("invalid_request")
		}
		defer f.Close()
		b := make([]byte, 128<<10)
		var copied int64
		for copied < size {
			if ctx.Err() != nil {
				return contracts.Fail("cancelled")
			}
			want := min(int64(len(b)), size-copied)
			got, e := reader.Read(b[:want])
			if got > 0 {
				if _, err := f.Write(b[:got]); err != nil {
					return contracts.Fail("unavailable")
				}
				copied += int64(got)
			}
			if e != nil && copied != size {
				return contracts.Fail("unavailable")
			}
		}
		return nil
	}
	if format == "zip" {
		z, e := zip.OpenReader(source)
		if e != nil {
			return contracts.Fail("invalid_request")
		}
		defer z.Close()
		if len(z.File) > 30000 {
			return contracts.Fail("input_limit")
		}
		for _, f := range z.File {
			n, e := archiveName(f.Name)
			if e != nil {
				return e
			}
			if f.FileInfo().IsDir() {
				if root.MkdirAll(n, 0700) != nil {
					return contracts.Fail("unavailable")
				}
				continue
			}
			if !f.Mode().IsRegular() {
				return contracts.Fail("invalid_request")
			}
			r, e := f.Open()
			if e != nil {
				return contracts.Fail("invalid_request")
			}
			e = put(n, int64(f.UncompressedSize64), f.Mode(), r)
			r.Close()
			if e != nil {
				return e
			}
		}
		return nil
	}
	if format != "tar.gz" {
		return contracts.Fail("invalid_request")
	}
	f, e := os.Open(source)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer f.Close()
	g, e := gzip.NewReader(f)
	if e != nil {
		return contracts.Fail("invalid_request")
	}
	defer g.Close()
	t := tar.NewReader(g)
	type link struct {
		name, target string
		hard         bool
	}
	links := []link{}
	count := 0
	for {
		h, e := t.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return contracts.Fail("invalid_request")
		}
		count++
		if count > 30000 {
			return contracts.Fail("input_limit")
		}
		n, e := archiveName(h.Name)
		if e != nil {
			return e
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if root.MkdirAll(n, 0700) != nil {
				return contracts.Fail("unavailable")
			}
		case tar.TypeReg, tar.TypeRegA:
			if e = put(n, h.Size, os.FileMode(h.Mode), t); e != nil {
				return e
			}
		case tar.TypeSymlink, tar.TypeLink:
			base := path.Dir(n)
			if h.Typeflag == tar.TypeLink {
				base = "."
			}
			target := path.Join(base, h.Linkname)
			if _, e = archiveName(h.Linkname); e != nil && (strings.HasPrefix(h.Linkname, "/") || strings.ContainsAny(h.Linkname, `\:`)) {
				return e
			}
			if _, e = archiveName(target); e != nil {
				return e
			}
			links = append(links, link{n, h.Linkname, h.Typeflag == tar.TypeLink})
		default:
			return contracts.Fail("invalid_request")
		}
	}
	for _, l := range links {
		if root.MkdirAll(path.Dir(l.name), 0700) != nil {
			return contracts.Fail("unavailable")
		}
		if l.hard {
			e = root.Link(l.target, l.name)
		} else {
			e = root.Symlink(l.target, l.name)
		}
		if e != nil {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
