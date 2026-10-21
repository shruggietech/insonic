// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
)

type LocalAssets struct {
	RequiredBytes int64              `json:"required_bytes"`
	VerifiedBytes int64              `json:"verified_bytes"`
	MissingBytes  int64              `json:"missing_bytes"`
	Failures      []*contracts.Error `json:"failures"`
}

// InspectOffline checks only the elected profile's declared files. MissingBytes
// is the expected size of missing or unusable files, never unrelated tree bytes.
func InspectOffline(ctx context.Context, directory string, p Profile) (LocalAssets, error) {
	out := LocalAssets{Failures: []*contracts.Error{}}
	if !filepath.IsAbs(directory) {
		return out, contracts.Fail("invalid_request")
	}
	for _, a := range append([]Asset{p.Runtime, p.Installer}, p.Wheels...) {
		if !contracts.ValidAssetName(a.Name) || filepath.Base(a.Name) != a.Name {
			return out, contracts.Fail("invalid_request")
		}
		out.RequiredBytes += a.Size
		if err := verifyAsset(ctx, filepath.Join(directory, a.Name), a); err != nil {
			if ctx.Err() != nil {
				return out, contracts.Fail("cancelled")
			}
			out.MissingBytes += a.Size
			out.Failures = append(out.Failures, contracts.SafeError(err))
		} else {
			out.VerifiedBytes += a.Size
		}
	}
	return out, nil
}
