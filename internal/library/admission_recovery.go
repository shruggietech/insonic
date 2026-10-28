// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

// A report published after its canonical object is the durable completion
// boundary for capture. Reuse only that exact pair under the existing work
// attempt's later acceptance fence. Earlier partial candidates can be rebuilt.
func (s *Service) recoverAdmissionCandidate(ctx context.Context, work catalog.Work, ordinal int, digest string, size int64) (bundle captureBundle, canonical catalog.Publication, reportID string, recovered bool, err error) {
	reportID, err = s.candidateAttempt(ctx, DerivedID(work.ID, "canonical-report-"+strconv.Itoa(ordinal)))
	if err != nil {
		return
	}
	report, e := s.Catalog.Publication(ctx, reportID)
	if code(e) == "not_found" {
		return
	}
	if e != nil {
		err = e
		return
	}
	if report.Kind != "media-metadata-report" {
		err = contracts.Fail("conflict")
		return
	}
	if report.State == "pending" {
		report, err = s.Artifacts.Reconcile(ctx, reportID)
		if err != nil {
			return
		}
	}
	if report.State != "available" {
		err = contracts.Fail("unavailable")
		return
	}
	materialized, e := s.Artifacts.MaterializeBound(ctx, reportID, report.Size)
	if e != nil {
		err = e
		return
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if release := s.Artifacts.Release(cleanup, reportID, materialized.Lease.ID); err == nil && release != nil {
			err, recovered = release, false
		}
	}()
	data, e := os.ReadFile(materialized.Path)
	if e != nil || strict(data, &bundle) != nil {
		err = contracts.Fail("operation_failed")
		return
	}
	if bundle.CanonicalPublication == nil {
		// A pre-binding report cannot prove which canonical bytes it described.
		// Preserve the prior recomputation contract without claiming recovery.
		return
	}
	if bundle.SourceDigest != digest || bundle.SourceSize != size {
		err = contracts.Fail("conflict")
		return
	}
	identity := bundle.CanonicalPublication
	expected, e := s.candidateAttempt(ctx, DerivedID(work.ID, "canonical-"+strconv.Itoa(ordinal)))
	if e != nil {
		err = e
		return
	}
	if identity.ID != expected || bundle.Facts.Canonical == nil || bundle.Facts.Canonical.SourceDigest != digest || bundle.Facts.Canonical.SourceSize != size {
		err = contracts.Fail("conflict")
		return
	}
	canonical, err = s.Catalog.Publication(ctx, identity.ID)
	if err != nil {
		return
	}
	if canonical.State != "available" || canonical.Kind != "canonical-audio" || canonical.Digest != identity.Digest || canonical.Size != identity.Size {
		err = contracts.Fail("conflict")
		return
	}
	if err = s.Artifacts.Verify(ctx, canonical.ID); err != nil {
		return
	}
	recovered = true
	return
}
