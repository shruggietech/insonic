// SPDX-License-Identifier: Apache-2.0
// Package packageenv resolves installed, hash-pinned companion tools without
// persisting installation paths into portable workspace configuration.
package packageenv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/subtitles"
)

const ManifestName = "insonic-companions.json"

type Environment struct {
	Media  library.Tools
	Cueson subtitles.Tool
}

// Manifest paths are installation-relative, except the declared OS Perl
// prerequisite. Each actual runtime tool receives an absolute, pinned path.
type Manifest struct {
	Kind     string         `json:"kind"`
	Version  string         `json:"schema_version"`
	Platform string         `json:"platform"`
	Media    library.Tools  `json:"media"`
	Cueson   subtitles.Tool `json:"cueson"`
	Perl     bool           `json:"system_perl,omitempty"`
}

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Defaults does not use PATH or ambient tool environment overrides. An absent
// manifest means a source build; a present invalid manifest is a hard failure.
func Defaults() (Environment, bool, error) {
	executable, err := os.Executable()
	if err != nil {
		return Environment{}, false, contracts.Fail("unavailable")
	}
	return Read(filepath.Dir(executable))
}

func Read(directory string) (Environment, bool, error) {
	f, err := os.Open(filepath.Join(directory, ManifestName))
	if os.IsNotExist(err) {
		return Environment{}, false, nil
	}
	if err != nil {
		return Environment{}, true, contracts.Fail("unavailable")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return Environment{}, true, contracts.Fail("output_limit")
	}
	var manifest Manifest
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&manifest) != nil || decoder.Decode(new(any)) != io.EOF {
		return Environment{}, true, contracts.Fail("invalid_request")
	}
	if manifest.Kind != "installed-companions" || manifest.Version != contracts.Version || manifest.Platform != runtime.GOOS+"_"+runtime.GOARCH || manifest.Media.Kind != "media-tools" || manifest.Media.Version != contracts.Version {
		return Environment{}, true, contracts.Fail("incompatible_version")
	}
	root, err := filepath.Abs(directory)
	if err != nil {
		return Environment{}, true, contracts.Fail("unavailable")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Environment{}, true, contracts.Fail("unavailable")
	}
	verified := make(map[string]string)
	var totalBytes int64
	resolve := func(pin *library.PinnedFile) error {
		if !digestPattern.MatchString(pin.SHA256) || pin.Path == "" || strings.ContainsAny(pin.Path, "\\:") || filepath.IsAbs(pin.Path) {
			return contracts.Fail("invalid_request")
		}
		for _, part := range strings.Split(pin.Path, "/") {
			if part == ".." || part == "." || part == "" {
				return contracts.Fail("invalid_request")
			}
		}
		path := filepath.Join(root, filepath.FromSlash(pin.Path))
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return contracts.Fail("unavailable")
		}
		relative, err := filepath.Rel(root, resolved)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return contracts.Fail("invalid_request")
		}
		if digest, found := verified[resolved]; found {
			if digest != pin.SHA256 {
				return contracts.Fail("incompatible_version")
			}
		} else {
			info, err := os.Stat(resolved)
			if err != nil || !info.Mode().IsRegular() {
				return contracts.Fail("unavailable")
			}
			totalBytes += info.Size()
			if len(verified) >= 4096 || totalBytes > 1<<30 {
				return contracts.Fail("output_limit")
			}
			if err := verifyFile(resolved, pin.SHA256); err != nil {
				return err
			}
			verified[resolved] = pin.SHA256
		}
		pin.Path = resolved
		return nil
	}
	for _, tool := range []*library.Tool{&manifest.Media.ExifTool, &manifest.Media.FFprobe, &manifest.Media.FFmpeg} {
		pin := library.PinnedFile{Path: tool.Path, SHA256: tool.SHA256}
		if tool.Version == "" || tool.Interpreter != nil {
			return Environment{}, true, contracts.Fail("invalid_request")
		}
		if err := resolve(&pin); err != nil {
			return Environment{}, true, err
		}
		tool.Path = pin.Path
		for i := range tool.SupportFiles {
			if err := resolve(&tool.SupportFiles[i]); err != nil {
				return Environment{}, true, err
			}
		}
	}
	pin := library.PinnedFile{Path: manifest.Cueson.Executable, SHA256: manifest.Cueson.ExecutableSHA256}
	if err := resolve(&pin); err != nil {
		return Environment{}, true, err
	}
	manifest.Cueson.Executable = pin.Path
	if manifest.Perl {
		if runtime.GOOS == "windows" {
			return Environment{}, true, contracts.Fail("invalid_request")
		}
		// Unix Perl is an explicit OS prerequisite, not a selected Python/model
		// worker. Pin its actual local bytes for existing child-tool validation.
		perl, err := filepath.EvalSymlinks("/usr/bin/perl")
		if err != nil {
			return Environment{}, true, contracts.Fail("unavailable")
		}
		file, err := os.Open(perl)
		if err != nil {
			return Environment{}, true, contracts.Fail("unavailable")
		}
		hash := sha256.New()
		_, err = io.Copy(hash, io.LimitReader(file, (64<<20)+1))
		file.Close()
		if err != nil {
			return Environment{}, true, contracts.Fail("unavailable")
		}
		manifest.Media.ExifTool.Interpreter = &library.PinnedFile{Path: perl, SHA256: hex.EncodeToString(hash.Sum(nil))}
	} else if runtime.GOOS != "windows" {
		return Environment{}, true, contracts.Fail("invalid_request")
	}
	return Environment{Media: manifest.Media, Cueson: manifest.Cueson}, true, nil
}

func verifyFile(path, digest string) error {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 512<<20 {
		return contracts.Fail("unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return contracts.Fail("unavailable")
	}
	if hex.EncodeToString(hash.Sum(nil)) != digest {
		return contracts.Fail("incompatible_version")
	}
	return nil
}
