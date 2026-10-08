// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/graph"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/workspace"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Attempt = catalog.Attempt
type worker struct {
	attempt Attempt
	cancel  context.CancelFunc
}
type App struct {
	graphMu           sync.Mutex
	Graph             graph.Adapter
	recordingFactory  func() (*recordingExecution, error)
	hostedClient      *http.Client // Only injected by deterministic protocol fixtures.
	Workspace         *workspace.Workspace
	Session           string
	Catalog           catalog.Catalog
	mu                sync.Mutex
	attempts          map[string]*worker
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	closed            bool
	artifactMu        sync.Mutex
	Artifacts         *artifact.Service
	secrets           contracts.SecretProvider
	secretOwner       *liveSecrets
	workMu            sync.Mutex
	workers           map[string]*realWorker
	settingsMu        sync.Mutex
	playbackMu        sync.Mutex
	playbacks         map[string]*playbackEntry
	playbackPreparing int
	playbackPreview   func(context.Context, *playbackEntry, catalog.LibraryEntry) error
}

const leaseTTL = 5 * time.Second

func New(w *workspace.Workspace) (*App, error) {
	return NewContext(context.Background(), w)
}
func NewContext(ownerContext context.Context, w *workspace.Workspace) (*App, error) {
	provider, err := credentialcmd.Provider(w, nil)
	if err != nil {
		return nil, err
	}
	live := &liveSecrets{manager: provider}
	a, err := newOwnerContext(ownerContext, w, live)
	if err != nil {
		provider.Close()
		return nil, err
	}
	a.secretOwner = live
	return a, nil
}
func NewContextWithSecrets(ctx context.Context, w *workspace.Workspace, provider contracts.SecretProvider) (*App, error) {
	return newOwnerContext(ctx, w, provider)
}
func NewWithSecrets(w *workspace.Workspace, secrets contracts.SecretProvider) (*App, error) {
	return newOwnerContext(context.Background(), w, secrets)
}
func newOwnerContext(ownerContext context.Context, w *workspace.Workspace, secrets contracts.SecretProvider) (*App, error) {
	ctx, cancel := context.WithCancel(ownerContext)
	openCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	store, err := catalog.OpenWorkspace(openCtx, w, secrets, false)
	stop()
	if err != nil {
		cancel()
		return nil, err
	}
	if err = store.RegisterWorkspace(ctx, w); err != nil {
		cancel()
		store.Close()
		return nil, err
	}
	a := &App{Workspace: w, Session: contracts.ID(), Catalog: store, attempts: map[string]*worker{}, workers: map[string]*realWorker{}, playbacks: map[string]*playbackEntry{}, ctx: ctx, cancel: cancel, secrets: secrets}
	if live, ok := secrets.(*liveSecrets); ok {
		a.secretOwner = live
	}
	// The workspace runtime owns this control directory exclusively. Recover
	// disposable results before dispatching recovered work, without requiring
	// an authenticated artifact-store connection merely to remove scratch.
	scratch := processing.Service{Artifacts: &artifact.Service{Workspace: w}}
	if err = scratch.RecoverScratch(ctx); err != nil {
		cancel()
		store.Close()
		return nil, err
	}
	a.recoverWork()
	recovered, err := store.Recover(ctx, a.Session, leaseTTL)
	if err != nil {
		cancel()
		store.Close()
		return nil, err
	}
	a.wg.Add(1)
	go a.maintainGraph(ctx)
	for _, r := range recovered {
		a.attach(r)
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.recoverWork()
				a.mu.Lock()
				if a.closed {
					a.mu.Unlock()
					return
				}
				claims := make([]Attempt, 0, len(a.attempts))
				for _, w := range a.attempts {
					claims = append(claims, w.attempt)
				}
				live, renewalErr := store.RenewClaims(ctx, a.Session, claims, leaseTTL)
				for id, w := range a.attempts {
					if renewalErr != nil || !live[w.attempt.AttemptID] {
						w.cancel()
						delete(a.attempts, id)
					}
				}
				var recovered []Attempt
				var err error
				if len(a.attempts) < 1024 {
					recovered, err = store.RecoverLimit(ctx, a.Session, leaseTTL, min(128, 1024-len(a.attempts)))
				}
				if err == nil {
					for _, r := range recovered {
						a.attach(r)
					}
				}
				a.mu.Unlock()
			}
		}
	}()
	// Storage cleanup can consume its ten-second I/O budget. It must never
	// delay five-second claim renewal or overlap another cleanup sweep.
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		a.recoverDerivedCleanup()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.recoverDerivedCleanup()
			}
		}
	}()
	a.wg.Add(1)
	go a.maintainPlaybacks()
	return a, nil
}

// Caller holds mu after construction. Only current durable authority is attached.
func (a *App) attach(attempt Attempt) {
	if a.closed || attempt.State != "running" || attempt.SessionID != a.Session {
		return
	}
	current, err := a.Catalog.ShowJob(a.ctx, attempt.JobID)
	if err != nil || current.AttemptID != attempt.AttemptID || current.State != "running" {
		return
	}
	if old := a.attempts[attempt.JobID]; old != nil {
		if old.attempt.AttemptID == attempt.AttemptID {
			return
		}
		old.cancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.attempts[attempt.JobID] = &worker{attempt: attempt, cancel: cancel}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		// Work duration is monotonic and relative; server audit clock skew must
		// not shorten or extend the qualification operation.
		timer := time.NewTimer(time.Duration(attempt.DurationMS) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			a.Complete(attempt.JobID, attempt.Generation, "succeeded")
		}
	}()
}
func (a *App) Start(duration int) (Attempt, error) { return a.startRequest(contracts.ID(), duration) }
func (a *App) startRequest(op string, duration int) (Attempt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return Attempt{}, contracts.Fail("unavailable")
	}
	if len(a.attempts) >= 1024 {
		return a.Catalog.ReconcileJob(a.ctx, op, "start", "", duration)
	}
	out, err := a.Catalog.StartJob(a.ctx, op, a.Session, duration, leaseTTL)
	if err == nil {
		a.attach(out)
	}
	return out, err
}
func (a *App) Show(id string) (Attempt, error)   { return a.Catalog.ShowJob(a.ctx, id) }
func (a *App) Cancel(id string) (Attempt, error) { return a.cancelRequest(contracts.ID(), id) }
func (a *App) cancelRequest(op, id string) (Attempt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return Attempt{}, contracts.Fail("unavailable")
	}
	out, err := a.Catalog.CancelJob(a.ctx, op, id)
	if err == nil {
		if w := a.attempts[id]; w != nil && w.attempt.AttemptID == out.AttemptID && out.State != "running" {
			w.cancel()
			delete(a.attempts, id)
		}
	}
	return out, err
}
func (a *App) Retry(id string) (Attempt, error) { return a.retryRequest(contracts.ID(), id) }
func (a *App) retryRequest(op, id string) (Attempt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return Attempt{}, contracts.Fail("unavailable")
	}
	if len(a.attempts) >= 1024 {
		return a.Catalog.ReconcileJob(a.ctx, op, "retry", id, 0)
	}
	out, err := a.Catalog.RetryJob(a.ctx, op, id, a.Session, leaseTTL)
	if err == nil {
		a.attach(out)
	}
	return out, err
}
func (a *App) Complete(id string, generation int64, state string) bool {
	if state != "succeeded" && state != "failed" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	w := a.attempts[id]
	if a.closed || w == nil || w.attempt.Generation != generation {
		return false
	}
	if err := a.Catalog.Complete(a.ctx, w.attempt, state); err != nil {
		// Stop renewal after an unknown/failed acceptance; recovery reconciles
		// durable authority instead of leaving a finished worker alive forever.
		w.cancel()
		delete(a.attempts, id)
		return false
	}
	w.cancel()
	delete(a.attempts, id)
	return true
}
func (a *App) Active() int {
	a.playbackMu.Lock()
	playbackCount := len(a.playbacks) + a.playbackPreparing
	a.playbackMu.Unlock()
	a.mu.Lock()
	n := len(a.attempts)
	a.mu.Unlock()
	a.workMu.Lock()
	defer a.workMu.Unlock()
	return n + len(a.workers) + playbackCount
}
func (a *App) Close() {
	// In-flight catalog calls can hold mu while waiting on the server. Cancel
	// their context first so shutdown can acquire mu and drain workers.
	a.cancel()
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return
	}
	a.closed = true
	a.cancel()
	a.mu.Unlock()
	a.wg.Wait()
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	a.closePlaybacks(ctx)
	a.Catalog.InterruptOwner(ctx, a.Session)
	a.artifactMu.Lock()
	if a.Artifacts != nil {
		a.Artifacts.Close()
	}
	a.artifactMu.Unlock()
	a.graphMu.Lock()
	if a.Graph != nil {
		a.Graph.Close()
	}
	a.graphMu.Unlock()
	a.Catalog.Close()
	if a.secretOwner != nil {
		a.secretOwner.Close()
	}
}

func (a *App) Dispatch(req contracts.Request) contracts.Response {
	response := contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: req.RequestID, SessionID: a.Session}
	if !contracts.ValidID(req.RequestID) {
		response.RequestID = ""
	}
	if req.Version != contracts.Version {
		response.Error = contracts.Fail("incompatible_version")
		return response
	}
	if req.WorkspaceID != a.Workspace.Config.WorkspaceID {
		response.Error = contracts.Fail("workspace_mismatch")
		return response
	}
	jobOp := req.Operation == "jobs.show" || req.Operation == "jobs.cancel" || req.Operation == "jobs.retry" || req.Operation == "jobs.history"
	if req.Kind != "runtime-request" || !contracts.ValidID(req.RequestID) || !artifactRequestValid(req) || !domainRequestValid(req) || (req.Operation != "jobs.start" && req.DurationMS != 0) || (!jobOp && req.JobID != "") || (req.Operation != "jobs.history" && req.AfterGeneration != 0) {
		response.Error = contracts.Fail("invalid_request")
		return response
	}
	var result any
	var err error
	switch req.Operation {
	case "workspace.show":
		result = map[string]any{"workspace_id": req.WorkspaceID, "display_name": a.Workspace.Config.DisplayName, "adapters": []string{a.Workspace.Config.Profiles.Storage.Adapter, a.Catalog.Backend(), a.Workspace.Config.Profiles.Graph.Adapter}}
	case "catalog.show":
		result, err = a.Catalog.Status(a.ctx)
	case "doctor":
		paths, pathErr := workspace.PlatformPaths()
		err = pathErr
		storage, details := a.storageDiagnostics()
		cleanups, cleanupErr := a.Catalog.Cleanups(a.ctx)
		cleanupState := map[string]any{"state": "available", "pending": 0}
		if cleanupErr != nil {
			cleanupState["state"] = "unavailable"
		} else {
			pending := 0
			for _, c := range cleanups {
				if c.State == "pending" {
					pending++
				}
			}
			cleanupState["pending"] = pending
		}
		result = map[string]any{"paths": paths, "attempt_persistence": "durable", "storage": details, "derived_cleanup": cleanupState, "adapters": []contracts.Capability{storage, {AdapterID: a.Catalog.Backend(), ContractVersion: contracts.Version, State: "available", Operations: []string{"revisions", "jobs", "outbox", "snapshot"}}, {AdapterID: a.Workspace.Config.Profiles.Graph.Adapter, ContractVersion: contracts.Version, State: "configured", Operations: []string{}}}}
	case "jobs.start":
		result, err = a.startRequest(req.RequestID, req.DurationMS)
	case "jobs.show", "jobs.cancel", "jobs.retry", "jobs.history":
		if !contracts.ValidID(req.JobID) {
			err = contracts.Fail("invalid_request")
			break
		}
		switch req.Operation {
		case "jobs.show":
			result, err = a.Show(req.JobID)
		case "jobs.cancel":
			result, err = a.cancelRequest(req.RequestID, req.JobID)
		case "jobs.retry":
			result, err = a.retryRequest(req.RequestID, req.JobID)
		case "jobs.history":
			result, err = a.Catalog.HistoryPage(a.ctx, req.JobID, req.AfterGeneration)
		}
	default:
		if contracts.ExploreOperation(req.Operation) {
			result, err = a.exploreDispatch(req)
		} else if contracts.DesktopOperation(req.Operation) {
			result, err = a.desktopDispatch(req)
		} else if strings.HasPrefix(req.Operation, "credentials.") {
			result, err = a.credentialDispatch(req)
		} else if configuredOperation(req.Operation) || req.ItemID != "" || len(req.Data) > 0 || req.Operation == "media.list" || req.Operation == "models.list" || req.Operation == "work.list" {
			result, err = a.domainDispatch(req)
		} else {
			result, err = a.artifactDispatch(req)
		}
	}
	if err != nil {
		if typed, ok := err.(*contracts.Error); ok {
			response.Error = typed
		} else {
			response.Error = contracts.Fail("operation_failed")
		}
	} else {
		response.Result = result
	}
	return response
}
