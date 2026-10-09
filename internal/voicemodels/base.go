// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func (s *Service) materializeBase(ctx context.Context, o TrainOptions, directory string) (string, []PreparedInput, error) {
	if o.BaseModelID == "" {
		return "", nil, nil
	}
	install, err := s.Catalog.BaseModel(ctx, o.BaseModelID)
	if err != nil {
		return "", nil, err
	}
	var manifest models.Manifest
	if models.DecodeManifest(install.Manifest, &manifest) != nil || install.Digest != o.BaseDigest || manifest.Digest() != o.BaseDigest || install.State != "available" {
		return "", nil, contracts.Fail("conflict")
	}
	if manifest.Compatibility != nil && manifest.Compatibility.Architecture != o.Adapter.Architecture {
		return "", nil, contracts.Fail("unsupported_capability")
	}
	modelService := models.NewService(s.Artifacts, s.Catalog, s.Secrets)
	if err = modelService.Verify(ctx, o.BaseModelID); err != nil {
		return "", nil, err
	}
	materializations, err := modelService.Materialize(ctx, o.BaseModelID)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		release, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, m := range materializations {
			_ = s.Artifacts.Release(release, m.PublicationID, m.Lease.ID)
		}
	}()
	base := filepath.Join(directory, "base")
	if err = os.Mkdir(base, 0700); err != nil {
		return "", nil, contracts.Fail("unavailable")
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return "", nil, contracts.Fail("unavailable")
	}
	defer root.Close()
	inputs := []PreparedInput{}
	if len(materializations) != len(manifest.Files) {
		return "", nil, contracts.Fail("conflict")
	}
	var total int64
	for index, file := range manifest.Files {
		if !models.ValidRole(file.Role) {
			return "", nil, contracts.Fail("invalid_request")
		}
		total += file.Size
		if total > o.Adapter.Limits.MaxInputBytes {
			return "", nil, contracts.Fail("input_limit")
		}
		parent := path.Dir(file.Role)
		if parent != "." {
			prefix := ""
			for _, part := range strings.Split(parent, "/") {
				prefix = path.Join(prefix, part)
				if err = root.Mkdir(prefix, 0700); err != nil && !os.IsExist(err) {
					return "", nil, contracts.Fail("unavailable")
				}
			}
		}
		source, err := os.Open(materializations[index].Path)
		if err != nil {
			return "", nil, contracts.Fail("unavailable")
		}
		dest, err := root.OpenFile(file.Role, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			source.Close()
			return "", nil, contracts.Fail("unavailable")
		}
		n, copyError := io.Copy(dest, io.LimitReader(&contextReader{ctx: ctx, reader: source}, file.Size+1))
		err = copyError
		source.Close()
		syncErr := dest.Sync()
		closeErr := dest.Close()
		if err != nil || syncErr != nil || closeErr != nil {
			return "", nil, contracts.Fail("unavailable")
		}
		if n != file.Size {
			return "", nil, contracts.Fail("conflict")
		}
		if err = pinned(ctx, library.PinnedFile{Path: filepath.Join(base, filepath.FromSlash(file.Role)), SHA256: file.SHA256}); err != nil {
			return "", nil, err
		}
		inputs = append(inputs, PreparedInput{ID: file.Role, Path: filepath.Join(base, filepath.FromSlash(file.Role)), SHA256: file.SHA256, Size: file.Size})
	}
	return base, inputs, nil
}
