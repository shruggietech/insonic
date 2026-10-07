// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"errors"
)

// RecoverLegacy adopts only pending migration obligations at the elected storage
// profile. No old location implies verified availability: actual bytes are read
// and hashed before creating a fenced admission. Other profiles remain pending.
func (s *Service) RecoverLegacy(ctx context.Context) error {
	selected := s.Workspace.Config.Profiles.Storage
	candidates, e := s.Catalog.LegacyCleanupPublications(ctx, selected.ID, selected.Revision)
	if e != nil {
		return e
	}
	var failures []error
	for _, p := range candidates {
		if existing, err := s.Catalog.Publication(ctx, p.ID); err == nil && existing.State != "pending" {
			continue
		}
		p.Owner = s.Owner
		// Check selected existing bytes first, before recording any admission.
		if err := Verify(ctx, s.Store, p); err != nil {
			failures = append(failures, err)
			continue
		}
		p, e = s.Catalog.BeginPublication(ctx, p, s.TTL)
		if e != nil {
			failures = append(failures, e)
			continue
		}
		if p.State != "pending" {
			continue
		}
		if e = Verify(ctx, s.Store, p); e != nil {
			failures = append(failures, e)
			continue
		}
		p.Verification = "sha256-readback"
		if _, e = s.Catalog.AdmitPublication(ctx, p); e != nil {
			failures = append(failures, e)
		}
	}
	return errors.Join(failures...)
}
