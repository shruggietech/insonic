// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/workspace"
	"time"
)

// Catalog is the shared backend-neutral authority used by runtime operations.
// Both official adapters implement it; callers never receive SQL connections.
type Catalog interface {
	GraphTarget() string
	EnqueueGraphRebuild(context.Context, string) (Receipt, error)
	AdvanceOutboxGeneration(context.Context, string, int64, time.Duration) error
	GraphStatus(context.Context) (map[string]int64, error)
	EnqueueGraphRefresh(context.Context, string, int64, json.RawMessage) (Receipt, error)
	RenewOutbox(context.Context, OutboxClaim, time.Duration) (OutboxClaim, error)
	Extraction(context.Context, string) (Extraction, error)
	CommitExtraction(context.Context, Work, Extraction) (Extraction, error)
	AcceptedExtractionWork(context.Context, string, string) (json.RawMessage, bool, error)
	SavedQueries(context.Context, string) ([]SavedQuery, error)
	PutSavedQuery(context.Context, string, int64, SavedQuery) (SavedQuery, error)
	GraphLayout(context.Context, string) (GraphLayout, error)
	PutGraphLayout(context.Context, string, int64, GraphLayout) (GraphLayout, error)
	Pipeline(context.Context, string) (Pipeline, error)
	Pipelines(context.Context, string, int) ([]Pipeline, string, error)
	PutPipeline(context.Context, string, int64, Pipeline) (Pipeline, error)
	Speaker(context.Context, string) (SpeakerIdentity, error)
	Speakers(context.Context, string, int) ([]SpeakerIdentity, string, error)
	PutSpeaker(context.Context, string, int64, SpeakerIdentity) (SpeakerIdentity, error)
	Term(context.Context, string) (Term, error)
	Terms(context.Context, string, int) ([]Term, string, error)
	PutTerm(context.Context, string, int64, Term) (Term, error)
	ContextInputs(context.Context, ContextFilter) (ContextSnapshot, error)
	CurrentSpeakerReferences(context.Context, SpeakerSelection) (SpeakerSelectionPage, error)
	ComparePriorSpeakerEvidence(context.Context, SpeakerSelection, CurrentReference, string) ([]PriorEvidenceComparison, error)
	ResolveEvidence(context.Context, CurrentReference) (ResolvedEvidence, error)
	ResolveSegment(context.Context, string, int64) (ResolvedSegment, error)
	AcceptedRecordingWork(context.Context, string, string) (json.RawMessage, bool, error)
	QueueDerivedCleanup(context.Context, string, string, string) error
	Recording(context.Context, string) (Recording, error)
	CommitRecording(context.Context, Work, int64, Recording) (Recording, error)
	SpeakerMappings(context.Context, string) ([]SpeakerMapping, error)
	SetSpeakerMapping(context.Context, string, int64, SpeakerMapping) (SpeakerMapping, error)
	EnqueueWork(context.Context, string, string, json.RawMessage) (Work, error)
	Work(context.Context, string) (Work, error)
	Works(context.Context) ([]Work, error)
	ClaimWork(context.Context, string, string, time.Duration) (Work, error)
	RenewWork(context.Context, Work, time.Duration) (Work, error)
	CheckpointWork(context.Context, Work, string, string, json.RawMessage, time.Duration) (Work, error)
	CancelWork(context.Context, string, string) (Work, error)
	RetryWork(context.Context, string, string) (Work, error)
	Library(context.Context, string) (LibraryEntry, error)
	Libraries(context.Context) ([]LibraryEntry, error)
	CommitLibrary(context.Context, Work, LibraryEntry) (LibraryEntry, error)
	CommitAdmission(context.Context, Work, int, int64, LibraryEntry, *Recording, ...json.RawMessage) (LibraryEntry, *Recording, error)
	SkipAdmission(context.Context, Work, int, json.RawMessage) error
	AcceptedAdmissionWork(context.Context, string, int) (json.RawMessage, bool, error)
	UpdateLibrary(context.Context, string, int64, LibraryEntry) (LibraryEntry, error)
	BaseModel(context.Context, string) (BaseModelInstall, error)
	BaseModels(context.Context) ([]BaseModelInstall, error)
	CommitBaseModel(context.Context, Work, BaseModelInstall) (BaseModelInstall, error)
	Cleanups(context.Context) ([]Cleanup, error)
	LegacyCleanupPublications(context.Context, string, int64) ([]Publication, error)
	FinishCleanup(context.Context, string) error
	Backend() string
	Close() error
	RegisterWorkspace(context.Context, *workspace.Workspace) error
	Status(context.Context) (map[string]any, error)
	Revision(context.Context) (int64, error)
	Commit(context.Context, Mutation) (Receipt, error)
	NamedSetting(context.Context, string) (json.RawMessage, int64, error)
	PutNamedSetting(context.Context, string, string, int64, json.RawMessage) (Receipt, error)
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
