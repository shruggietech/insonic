// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"github.com/shruggietech/insonic/internal/workspace"
	"time"
)

// Catalog is the shared backend-neutral authority used by runtime operations.
// Both official adapters implement it; callers never receive SQL connections.
type Catalog interface {
	Backend() string
	Close() error
	RegisterWorkspace(context.Context, *workspace.Workspace) error
	Status(context.Context) (map[string]any, error)
	Revision(context.Context) (int64, error)
	Commit(context.Context, Mutation) (Receipt, error)
	Settings(context.Context) ([]Setting, error)
	Export(context.Context) (Snapshot, error)
	Restore(context.Context, Snapshot) error
	StartJob(context.Context, string, string, int, time.Duration) (Attempt, error)
	ReconcileJob(context.Context, string, string, string, int) (Attempt, error)
	ShowJob(context.Context, string) (Attempt, error)
	History(context.Context, string) ([]Attempt, error)
	HistoryPage(context.Context, string, int64) (HistoryPage, error)
	CancelJob(context.Context, string, string) (Attempt, error)
	RetryJob(context.Context, string, string, string, time.Duration) (Attempt, error)
	Renew(context.Context, Attempt, time.Duration) error
	RenewClaims(context.Context, string, []Attempt, time.Duration) (map[string]bool, error)
	Complete(context.Context, Attempt, string) error
	Recover(context.Context, string, time.Duration) ([]Attempt, error)
	RecoverLimit(context.Context, string, time.Duration, int) ([]Attempt, error)
	InterruptOwner(context.Context, string) error
	ClaimLease(context.Context, string, string, time.Duration) (Lease, error)
	RenewLease(context.Context, Lease, time.Duration) error
	ClaimOutbox(context.Context, string, string, time.Duration) (OutboxClaim, error)
	AcknowledgeOutbox(context.Context, OutboxClaim) error
	Publication(context.Context, string) (Publication, error)
	BeginPublication(context.Context, Publication, time.Duration) (Publication, error)
	SavePublication(context.Context, Publication, time.Duration) (Publication, error)
	AdmitPublication(context.Context, Publication) (Publication, error)
	AbortPublication(context.Context, Publication) (Publication, error)
	ArtifactReference(context.Context, string, string, bool) error
	ArtifactLease(context.Context, string, string, string, time.Duration, bool) (MaterializationLease, error)
	RenewArtifactLease(context.Context, string, string, string, time.Duration) (MaterializationLease, error)
	ExpireMaterializations(context.Context, string) ([]string, error)
	ForgetMaterialization(context.Context, string, string) error
	ClaimRetirement(context.Context, string, string, time.Duration, time.Duration) (Publication, error)
	FinishRetirement(context.Context, Publication) (Publication, error)
}

var _ Catalog = (*Store)(nil)
