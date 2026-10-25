// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
)

func diagnosticMap(t *testing.T, value any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func TestProbeDeadlineRetainsBoundaryAndNeverLaunches(t *testing.T) {
	dir := t.TempDir()
	python := filepath.Join(dir, "python")
	if err := os.WriteFile(python, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	s := Service{Directory: dir, Run: func(context.Context, process.Spec) (process.CaptureResult, error) {
		called = true
		return process.CaptureResult{}, nil
	}}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	out, _ := s.Inspect(ctx, "recognition", python, false)
	probe, _ := diagnosticMap(t, out)["probe"].(map[string]any)
	if called || probe["reason"] != "deadline" || probe["stage"] != "interpreter_hash" {
		t.Fatalf("expired inspection lost its boundary: called=%v probe=%v", called, probe)
	}
}
func TestReceiptCancellationDoesNotClaimChangedBytes(t *testing.T) {
	dir := t.TempDir()
	python := filepath.Join(dir, "python")
	if err := os.WriteFile(python, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	pin, _ := Hash(context.Background(), python)
	receipt := Receipt{Kind: "worker-environment", Version: contracts.Version, Inspection: Inspection{Role: "recognition", Python: pin, Compatible: true, EngineAvailable: true, Files: []library.PinnedFile{pin}}}
	raw, _ := json.Marshal(receipt)
	name := filepath.Join(dir, "receipt.json")
	if err := os.WriteFile(name, raw, 0600); err != nil {
		t.Fatal(err)
	}
	bound, _ := Hash(context.Background(), name)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := contracts.SafeError(VerifyReceipt(ctx, bound, pin)); got.Code != "cancelled" {
		t.Fatalf("interrupted verification claimed %s", got.Code)
	}
}
func TestMalformedProbeOutputHasDistinctSafeDiagnostic(t *testing.T) {
	dir := t.TempDir()
	python := filepath.Join(dir, "python")
	os.WriteFile(python, []byte("fixture"), 0600)
	s := Service{Directory: dir, Run: func(context.Context, process.Spec) (process.CaptureResult, error) {
		return process.CaptureResult{Stdout: []byte("secret-token invalid-json")}, nil
	}}
	out, err := s.Inspect(context.Background(), "recognition", python, false)
	probe, _ := diagnosticMap(t, out)["probe"].(map[string]any)
	if err != nil || probe["reason"] != "output_invalid" || probe["stage"] != "output_validation" || out.EngineAvailable {
		t.Fatalf("malformed output was collapsed: %v %v", probe, err)
	}
	raw, _ := json.Marshal(out)
	if string(raw) == "" || containsSecret(raw) {
		t.Fatal("raw marker escaped")
	}
}
func containsSecret(raw []byte) bool {
	for i := 0; i+12 <= len(raw); i++ {
		if string(raw[i:i+12]) == "secret-token" {
			return true
		}
	}
	return false
}

func fixtureProbeProgress(mode string) *ProbeProgress {
	stages := []string{"package_versions", "library_import", "cpu_prerequisites", "cuda_prerequisites"}
	if mode == "snapshot" {
		stages = append(stages, "dependency_inventory", "dependency_hashing")
	}
	stages = append(stages, "complete")
	p := &ProbeProgress{Mode: mode, Stage: "complete", Reason: "success"}
	for _, stage := range stages {
		p.Checkpoints = append(p.Checkpoints, ProbeCheckpoint{Stage: stage})
	}
	return p
}

func TestProbeRejectsMalformedAndReorderedProgress(t *testing.T) {
	cases := map[string]func(*ProbeProgress){
		"unknown-stage":    func(p *ProbeProgress) { p.Stage = "secret-token" },
		"missing-stage":    func(p *ProbeProgress) { p.Checkpoints = p.Checkpoints[1:] },
		"reordered":        func(p *ProbeProgress) { p.Checkpoints[0], p.Checkpoints[1] = p.Checkpoints[1], p.Checkpoints[0] },
		"reversed-time":    func(p *ProbeProgress) { p.Checkpoints[0].ElapsedMS = 2; p.ElapsedMS = 1 },
		"negative-files":   func(p *ProbeProgress) { p.Files = -1; p.Checkpoints[len(p.Checkpoints)-1].Files = -1 },
		"overflow-files":   func(p *ProbeProgress) { p.Files = 30001; p.Checkpoints[len(p.Checkpoints)-1].Files = 30001 },
		"unknown-reason":   func(p *ProbeProgress) { p.Reason = "secret-token" },
		"running-terminal": func(p *ProbeProgress) { p.Reason = "running" },
		"invalid-mode":     func(p *ProbeProgress) { p.Mode = "secret-token" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			p := fixtureProbeProgress("inspect")
			change(p)
			if p.valid("inspect", true) {
				t.Fatal("invalid progress accepted")
			}
		})
	}
	p := fixtureProbeProgress("inspect")
	if !p.valid("inspect", true) {
		t.Fatal("valid progress rejected")
	}
}

func TestProbeCheckpointRejectsDuplicateOversizeAndRedirectedFiles(t *testing.T) {
	for _, raw := range []string{
		`{"mode":"inspect","mode":"snapshot"}`,
		strings.Repeat("x", probeProgressLimit+1),
		`{"mode":"inspect","stage":"secret-token"}`,
	} {
		home := t.TempDir()
		os.WriteFile(filepath.Join(home, probeCheckpoint), []byte(raw), 0600)
		if _, err := readProbeProgress(home, "inspect"); err == nil {
			t.Fatal("invalid checkpoint accepted")
		}
	}
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, probeCheckpoint), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := readProbeProgress(home, "inspect"); err == nil {
		t.Fatal("directory accepted as progress")
	}
}

func TestTimedOutProbeRetainsValidatedLastStage(t *testing.T) {
	dir := t.TempDir()
	python := filepath.Join(dir, "python")
	os.WriteFile(python, []byte("fixture"), 0600)
	synctest.Test(t, func(t *testing.T) {
		s := Service{Directory: dir, Run: func(ctx context.Context, spec process.Spec) (process.CaptureResult, error) {
			p := &ProbeProgress{Mode: "inspect", Stage: "library_import", Reason: "running", ElapsedMS: 1, Checkpoints: []ProbeCheckpoint{{Stage: "package_versions"}, {Stage: "library_import", ElapsedMS: 1}}}
			raw, _ := json.Marshal(p)
			if err := os.WriteFile(filepath.Join(spec.Directory, probeCheckpoint), raw, 0600); err != nil {
				t.Fatal(err)
			}
			<-ctx.Done()
			return process.CaptureResult{ExitCode: -1, Stderr: []byte("secret-token")}, ctx.Err()
		}}
		out, err := s.Inspect(context.Background(), "recognition", python, false)
		if err != nil || out.Probe == nil || out.Probe.Reason != "deadline" || out.Probe.Stage != "library_import" || out.Probe.Child == nil || out.Probe.ElapsedMS != 60000 || out.EngineAvailable {
			t.Fatalf("timeout observation: %+v %v", out.Probe, err)
		}
		raw, _ := json.Marshal(out)
		if containsSecret(raw) {
			t.Fatal("raw tool error escaped")
		}
	})
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("owned probe home was not removed")
	}
}

func TestMalformedOptionalReceiptObservationCannotAuthorizeReuse(t *testing.T) {
	dir := t.TempDir()
	python := filepath.Join(dir, "python")
	os.WriteFile(python, []byte("fixture"), 0600)
	pin, _ := Hash(context.Background(), python)
	p := &Probe{Mode: "snapshot", Stage: "complete", Reason: "success", Child: fixtureProbeProgress("snapshot")}
	p.Child.Reason = "secret-token"
	receipt := Receipt{Kind: "worker-environment", Version: contracts.Version, Inspection: Inspection{Role: "recognition", Python: pin, Compatible: true, EngineAvailable: true, Files: []library.PinnedFile{pin}, Probe: p}}
	raw, _ := json.Marshal(receipt)
	name := filepath.Join(dir, "receipt.json")
	os.WriteFile(name, raw, 0600)
	bound, _ := Hash(context.Background(), name)
	if VerifyReceipt(context.Background(), bound, pin) == nil {
		t.Fatal("malformed optional diagnostics accepted")
	}
}

func TestPrepareCancelledBeforeFallbackMakesNoAttempt(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, _ := Selected("recognition")
	calls := 0
	s := Service{Directory: dir, Fetch: func(context.Context, Asset) (string, error) { calls++; return "", nil }, Run: func(context.Context, process.Spec) (process.CaptureResult, error) {
		calls++
		return process.CaptureResult{}, nil
	}}
	if _, ok := ExistingManaged(ctx, library.PinnedFile{}, library.PinnedFile{}, p); ok {
		t.Fatal("cancelled verification reused environment")
	}
	_, err := s.Prepare(ctx, Election{Role: "recognition", Mode: "managed"})
	entries, _ := os.ReadDir(dir)
	if contracts.SafeError(err).Code != "cancelled" || calls != 0 || len(entries) != 0 {
		t.Fatalf("cancelled fallback created work: %d %v %v", calls, entries, err)
	}
}
