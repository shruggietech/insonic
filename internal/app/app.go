// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"sync"
	"time"
)

type Attempt struct {
	JobID       string `json:"job_id"`
	AttemptID   string `json:"attempt_id"`
	WorkspaceID string `json:"workspace_id"`
	SessionID   string `json:"runtime_session_id"`
	Generation  uint64 `json:"claim_generation"`
	State       string `json:"state"`
	DurationMS  int    `json:"duration_ms"`
}
type entry struct {
	Attempt
	cancel context.CancelFunc
}
type App struct {
	Workspace *workspace.Workspace
	Session   string
	mu        sync.Mutex
	attempts  map[string]*entry
}

func New(w *workspace.Workspace) *App {
	return &App{Workspace: w, Session: contracts.ID(), attempts: make(map[string]*entry)}
}

func (a *App) start(id string, generation uint64, duration int) Attempt {
	ctx, cancel := context.WithCancel(context.Background())
	attempt := Attempt{id, contracts.ID(), a.Workspace.Config.WorkspaceID, a.Session, generation, "running", duration}
	a.attempts[id] = &entry{attempt, cancel}
	go func() {
		timer := time.NewTimer(time.Duration(duration) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			a.Complete(id, generation, "succeeded")
		}
	}()
	return attempt
}
func (a *App) Start(duration int) (Attempt, error) {
	if duration < 1 || duration > 60000 {
		return Attempt{}, contracts.Fail("invalid_request")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.attempts) >= 1024 {
		return Attempt{}, contracts.Fail("conflict")
	}
	return a.start(contracts.ID(), 1, duration), nil
}
func (a *App) Show(id string) (Attempt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e, ok := a.attempts[id]
	if !ok {
		return Attempt{}, contracts.Fail("not_found")
	}
	return e.Attempt, nil
}
func (a *App) Cancel(id string) (Attempt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e, ok := a.attempts[id]
	if !ok {
		return Attempt{}, contracts.Fail("not_found")
	}
	if e.State == "running" {
		e.State = "cancelled"
		e.cancel()
	}
	return e.Attempt, nil
}
func (a *App) Retry(id string) (Attempt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e, ok := a.attempts[id]
	if !ok {
		return Attempt{}, contracts.Fail("not_found")
	}
	if e.State != "cancelled" && e.State != "failed" {
		return Attempt{}, contracts.Fail("conflict")
	}
	return a.start(id, e.Generation+1, e.DurationMS), nil
}
func (a *App) Complete(id string, generation uint64, state string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	e, ok := a.attempts[id]
	if !ok || e.Generation != generation || e.State != "running" || (state != "succeeded" && state != "failed") {
		return false
	}
	e.State = state
	e.cancel()
	return true
}
func (a *App) Active() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for _, e := range a.attempts {
		if e.State == "running" {
			n++
		}
	}
	return n
}
func (a *App) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, e := range a.attempts {
		e.cancel()
		if e.State == "running" {
			e.State = "cancelled"
		}
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
	if req.Kind != "runtime-request" || !contracts.ValidID(req.RequestID) ||
		(req.Operation != "jobs.start" && req.DurationMS != 0) ||
		(req.Operation != "jobs.show" && req.Operation != "jobs.cancel" && req.Operation != "jobs.retry" && req.JobID != "") {
		response.Error = contracts.Fail("invalid_request")
		return response
	}
	var result any
	var err error
	switch req.Operation {
	case "workspace.show":
		result = map[string]any{"workspace_id": req.WorkspaceID, "display_name": a.Workspace.Config.DisplayName, "adapters": []string{a.Workspace.Config.Profiles.Storage.Adapter, a.Workspace.Config.Profiles.Catalog.Adapter, a.Workspace.Config.Profiles.Graph.Adapter}}
	case "doctor":
		paths, pathErr := workspace.PlatformPaths()
		err = pathErr
		result = map[string]any{"paths": paths, "attempt_persistence": "runtime-session", "adapters": []contracts.Capability{{AdapterID: a.Workspace.Config.Profiles.Storage.Adapter, ContractVersion: contracts.Version, State: "configured", Operations: []string{}}, {AdapterID: a.Workspace.Config.Profiles.Catalog.Adapter, ContractVersion: contracts.Version, State: "configured", Operations: []string{}}, {AdapterID: a.Workspace.Config.Profiles.Graph.Adapter, ContractVersion: contracts.Version, State: "configured", Operations: []string{}}}}
	case "jobs.start":
		result, err = a.Start(req.DurationMS)
	case "jobs.show", "jobs.cancel", "jobs.retry":
		if !contracts.ValidID(req.JobID) {
			err = contracts.Fail("invalid_request")
			break
		}
		if req.Operation == "jobs.show" {
			result, err = a.Show(req.JobID)
		} else if req.Operation == "jobs.cancel" {
			result, err = a.Cancel(req.JobID)
		} else {
			result, err = a.Retry(req.JobID)
		}
	default:
		err = contracts.Fail("invalid_request")
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
