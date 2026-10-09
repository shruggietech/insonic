// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func (s *Service) Fetch(ctx context.Context, versionID, destination string) (any, error) {
	if !filepath.IsAbs(destination) || !contracts.ValidID(versionID) {
		return nil, contracts.Fail("invalid_request")
	}
	output, err := s.Catalog.SpeakerOutput(ctx, versionID)
	if err != nil {
		return nil, err
	}
	descriptors, err := ManifestArtifactDescriptors(output.Metadata)
	if err != nil {
		return nil, err
	}
	if len(descriptors) == 0 {
		return nil, contracts.Fail("unsupported_retrieval")
	}
	// Construct a sibling staging directory and rename only after every exact
	// artifact verifies. Existing destinations are never overwritten or merged.
	if _, err = os.Lstat(destination); !os.IsNotExist(err) {
		return nil, contracts.Fail("conflict")
	}
	parent := filepath.Dir(destination)
	parentRoot, err := os.OpenRoot(parent)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer parentRoot.Close()
	stageName := ".insonic-fetch-" + contracts.ID()
	if err = parentRoot.Mkdir(stageName, 0700); err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer parentRoot.RemoveAll(stageName)
	stagePath := filepath.Join(parent, stageName)
	if err = workspace.SecureDirectory(stagePath, true); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(stagePath)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer root.Close()
	seen := map[string]bool{}
	for _, descriptor := range descriptors {
		role := descriptor.Role
		if !models.ValidRole(role) || strings.EqualFold(role, "manifest.json") || seen[strings.ToLower(role)] || !contracts.ValidID(descriptor.PublicationID) {
			return nil, contracts.Fail("invalid_request")
		}
		seen[strings.ToLower(role)] = true
		publication, err := s.Catalog.Publication(ctx, descriptor.PublicationID)
		if err != nil {
			return nil, err
		}
		if publication.State != "available" || publication.Digest != descriptor.Digest || publication.Size != descriptor.Size || publication.ArtifactID != descriptor.ArtifactID {
			return nil, contracts.Fail("conflict")
		}
		if err = s.Artifacts.Verify(ctx, publication.ID); err != nil {
			return nil, err
		}
		materialized, err := s.Artifacts.MaterializeBound(ctx, publication.ID, 64<<30)
		if err != nil {
			return nil, err
		}
		err = copyFetched(ctx, root, role, materialized.Path, descriptor)
		release, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		releaseErr := s.Artifacts.Release(release, materialized.PublicationID, materialized.Lease.ID)
		cancel()
		if err != nil {
			return nil, err
		}
		if releaseErr != nil {
			return nil, releaseErr
		}
	}
	manifest, err := root.OpenFile("manifest.json", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	_, err = manifest.Write(output.Metadata)
	syncErr := manifest.Sync()
	closeErr := manifest.Close()
	if err != nil || syncErr != nil || closeErr != nil {
		return nil, contracts.Fail("unavailable")
	}
	root.Close()
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	if err = renameFetched(parentRoot, stageName, filepath.Base(destination)); err != nil {
		return nil, contracts.Fail("conflict")
	}
	return map[string]any{"version_id": versionID, "destination": destination, "artifacts": len(descriptors)}, nil
}
func copyFetched(ctx context.Context, root *os.Root, role, source string, d ManifestArtifactDescriptor) error {
	parent := path.Dir(role)
	if parent != "." {
		prefix := ""
		for _, part := range strings.Split(parent, "/") {
			prefix = path.Join(prefix, part)
			if err := root.Mkdir(prefix, 0700); err != nil && !os.IsExist(err) {
				return contracts.Fail("unavailable")
			}
		}
	}
	out, err := root.OpenFile(role, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	in, err := os.Open(source)
	if err != nil {
		out.Close()
		return contracts.Fail("unavailable")
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(out, h), io.LimitReader(&contextReader{ctx: ctx, reader: in}, d.Size+1))
	in.Close()
	syncErr := out.Sync()
	closeErr := out.Close()
	if err != nil || syncErr != nil || closeErr != nil || n != d.Size || hex.EncodeToString(h.Sum(nil)) != d.Digest {
		return contracts.Fail("conflict")
	}
	return nil
}
func (s *Service) readProfile(ctx context.Context, output catalog.SpeakerOutput) (Profile, error) {
	var empty Profile
	if output.Kind != "voice-embedding" {
		return empty, contracts.Fail("unsupported_capability")
	}
	descriptors, err := ManifestArtifactDescriptors(output.Metadata)
	if err != nil {
		return empty, err
	}
	for _, d := range descriptors {
		if d.Role != "profile.json" || d.Format != "application/json" {
			continue
		}
		p, err := s.Catalog.Publication(ctx, d.PublicationID)
		if err != nil {
			return empty, err
		}
		if p.ArtifactID != d.ArtifactID || p.Digest != d.Digest || p.Size != d.Size || p.State != "available" {
			return empty, contracts.Fail("conflict")
		}
		materialized, err := s.Artifacts.MaterializeBound(ctx, d.PublicationID, 1<<20)
		if err != nil {
			return empty, err
		}
		defer func() {
			release, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.Artifacts.Release(release, materialized.PublicationID, materialized.Lease.ID)
		}()
		raw, err := os.ReadFile(materialized.Path)
		if err != nil || hash(raw) != d.Digest {
			return empty, contracts.Fail("conflict")
		}
		var profile Profile
		if strict(raw, &profile) != nil || profile.Kind != "voice-embedding" || !contracts.ValidID(profile.ModelID) || !digestPattern.MatchString(profile.ModelDigest) || profile.EvidenceUS < 1 {
			return empty, contracts.Fail("invalid_engine_output")
		}
		if _, err = normalized(profile.Vector); err != nil {
			return empty, err
		}
		var manifest struct {
			Compatibility struct {
				Architecture string   `json:"architecture"`
				Operations   []string `json:"supported_operations"`
			} `json:"compatibility"`
		}
		if json.Unmarshal(output.Metadata, &manifest) != nil || manifest.Compatibility.Architecture != "pyannote" || !contains(manifest.Compatibility.Operations, "voice-matching") {
			return empty, contracts.Fail("unsupported_capability")
		}
		return profile, nil
	}
	return empty, contracts.Fail("model_unavailable")
}
