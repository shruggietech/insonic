// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/workspace"
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
	Workspace   *workspace.Workspace
	Session     string
	Catalog     catalog.Catalog
	mu          sync.Mutex
	attempts    map[string]*worker
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	closed      bool
	artifactMu  sync.Mutex
	Artifacts   *artifact.Service
	secrets     contracts.SecretProvider
	secretOwner *liveSecrets
	workMu      sync.Mutex
	workers     map[string]*realWorker
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
	a := &App{Workspace: w, Session: contracts.ID(), Catalog: store, attempts: map[string]*worker{}, workers: map[string]*realWorker{}, ctx: ctx, cancel: cancel, secrets: secrets}
	if live, ok := secrets.(*liveSecrets); ok {
		a.secretOwner = live
	}
	a.recoverWork()
	recovered, err := store.Recover(ctx, a.Session, leaseTTL)
	if err != nil {
		cancel()
		store.Close()
		return nil, err
	}
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
	a.mu.Lock()
	n := len(a.attempts)
	a.mu.Unlock()
	a.workMu.Lock()
	defer a.workMu.Unlock()
	return n + len(a.workers)
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
	a.Catalog.InterruptOwner(ctx, a.Session)
	a.artifactMu.Lock()
	if a.Artifacts != nil {
		a.Artifacts.Close()
	}
	a.artifactMu.Unlock()
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
		result = map[string]any{"paths": paths, "attempt_persistence": "durable", "storage": details, "adapters": []contracts.Capability{storage, {AdapterID: a.Catalog.Backend(), ContractVersion: contracts.Version, State: "available", Operations: []string{"revisions", "jobs", "outbox", "snapshot"}}, {AdapterID: a.Workspace.Config.Profiles.Graph.Adapter, ContractVersion: contracts.Version, State: "configured", Operations: []string{}}}}
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
		if strings.HasPrefix(req.Operation, "credentials.") {
			result, err = a.credentialDispatch(req)
		} else if req.ItemID != "" || len(req.Data) > 0 || req.Operation == "media.list" || req.Operation == "models.list" || req.Operation == "work.list" {
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
