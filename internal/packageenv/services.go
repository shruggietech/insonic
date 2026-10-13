// SPDX-License-Identifier: Apache-2.0
package packageenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ServiceResource is optional package metadata. Declaration never elects a route or starts a process.
type ServiceResource struct {
	Adapter    string `json:"adapter"`
	Endpoint   string `json:"endpoint"`
	Executable string `json:"executable,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
}

func ServiceResources(ctx context.Context) ([]ServiceResource, error) {
	executable, e := os.Executable()
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	return ReadServiceResources(ctx, resourceDirectory(executable))
}
func ReadServiceResources(ctx context.Context, directory string) ([]ServiceResource, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	f, e := os.Open(filepath.Join(directory, ManifestName))
	if os.IsNotExist(e) {
		return []ServiceResource{}, nil
	}
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	raw, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	f.Close()
	if e != nil || len(raw) > 1<<20 {
		return nil, contracts.Fail("output_limit")
	}
	var m Manifest
	if json.Unmarshal(raw, &m) != nil || m.Kind != "installed-companions" || m.Version != contracts.Version || m.Platform != runtime.GOOS+"_"+runtime.GOARCH {
		return nil, contracts.Fail("incompatible_version")
	}
	if len(m.Services) > 16 {
		return nil, contracts.Fail("output_limit")
	}
	root, e := filepath.EvalSymlinks(directory)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	for i := range m.Services {
		r := &m.Services[i]
		if r.Adapter != "ollama" {
			return nil, contracts.Fail("unsupported_capability")
		}
		if r.Executable == "" {
			continue
		}
		if filepath.IsAbs(r.Executable) || strings.ContainsAny(r.Executable, "\\:") || !digestPattern.MatchString(r.SHA256) {
			return nil, contracts.Fail("invalid_request")
		}
		path, e := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(r.Executable)))
		if e != nil {
			return nil, contracts.Fail("unavailable")
		}
		rel, e := filepath.Rel(root, path)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, contracts.Fail("invalid_request")
		}
		st, e := os.Stat(path)
		if e != nil || !st.Mode().IsRegular() || st.Size() > 512<<20 {
			return nil, contracts.Fail("invalid_request")
		}
		f, e := os.Open(path)
		if e != nil {
			return nil, contracts.Fail("unavailable")
		}
		hash := sha256.New()
		buf := make([]byte, 64<<10)
		for {
			if ctx.Err() != nil {
				f.Close()
				return nil, contracts.Fail("cancelled")
			}
			n, e := f.Read(buf)
			hash.Write(buf[:n])
			if e == io.EOF {
				break
			}
			if e != nil {
				f.Close()
				return nil, contracts.Fail("unavailable")
			}
		}
		f.Close()
		if hex.EncodeToString(hash.Sum(nil)) != r.SHA256 {
			return nil, contracts.Fail("incompatible_version")
		}
		r.Executable = path
	}
	return m.Services, nil
}
