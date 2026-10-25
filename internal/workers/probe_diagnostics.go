// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

const probeCheckpoint = "probe-progress.json"
const probeProgressLimit = 16 << 10
const probeMaximumMS int64 = 3600000
const probeMaximumBytes int64 = 9007199254740991

// Probe keeps native elapsed time separate from the child's monotonic origin.
type Probe struct {
	Mode      string         `json:"mode"`
	Stage     string         `json:"stage"`
	Reason    string         `json:"reason"`
	ElapsedMS int64          `json:"elapsed_ms"`
	BudgetMS  int64          `json:"budget_ms"`
	Child     *ProbeProgress `json:"child,omitempty"`
}
type ProbeProgress struct {
	Mode        string            `json:"mode"`
	Stage       string            `json:"stage"`
	Reason      string            `json:"reason"`
	ElapsedMS   int64             `json:"elapsed_ms"`
	Files       int64             `json:"files"`
	HashedFiles int64             `json:"hashed_files"`
	HashedBytes int64             `json:"hashed_bytes"`
	Checkpoints []ProbeCheckpoint `json:"checkpoints"`
}
type ProbeCheckpoint struct {
	Stage       string `json:"stage"`
	ElapsedMS   int64  `json:"elapsed_ms"`
	Files       int64  `json:"files"`
	HashedFiles int64  `json:"hashed_files"`
	HashedBytes int64  `json:"hashed_bytes"`
}

var childProbeStages = []string{"package_versions", "library_import", "cpu_prerequisites", "cuda_prerequisites", "dependency_inventory", "dependency_hashing", "complete"}
var probeMessages = map[string]string{
	"success":                "",
	"interpreter_missing":    "Choose a configured absolute Python interpreter or prepare the managed CPU environment.",
	"interpreter_unreadable": "The selected interpreter cannot be verified. Check its file and access permissions.",
	"home_unavailable":       "The private probe directory cannot be prepared. Check library control-directory access.",
	"launch_failed":          "The selected interpreter could not start. Inspect its platform and executable prerequisites.",
	"process_failed":         "The probe process failed. Inspect compatible CPU dependencies or prepare a fresh managed environment.",
	"deadline":               "Worker verification exhausted its enclosing deadline. Inspect the retained stage and timings before retrying.",
	"cancelled":              "Worker verification was cancelled; no new environment was accepted.",
	"output_invalid":         "The worker probe returned an invalid result. Use a compatible interpreter and current complete package.",
	"output_limit":           "The worker probe exceeded its bounded result size. Inspect the environment inventory.",
	"package_missing":        "A required worker package is missing. Prepare the managed environment or install the pinned dependencies.",
	"version_mismatch":       "Worker package versions do not match the supported profile. Prepare a compatible environment.",
	"import_failed":          "A required worker library could not import. Check its native dependencies or prepare the managed CPU environment.",
	"cpu_unavailable":        "Required CPU operations are unavailable. Prepare compatible CPU dependencies.",
	"inventory_limit":        "The environment exceeds the 30000-file inventory bound. Select a dedicated compatible environment.",
	"inventory_unreadable":   "The dependency inventory could not be read or hashed. Check file access and prepare a fresh environment.",
	"checkpoint_invalid":     "The worker progress report is invalid or unavailable. Use a current complete package and writable private probe directory.",
}

func probeStageIndex(stage string) int {
	for i, s := range childProbeStages {
		if s == stage {
			return i
		}
	}
	return -1
}
func validProbeStage(stage string) bool {
	if probeStageIndex(stage) >= 0 {
		return true
	}
	switch stage {
	case "interpreter_hash", "private_home", "process_start", "output_validation", "receipt_verification", "not_started":
		return true
	}
	return false
}
func validProbeReason(reason string) bool { _, ok := probeMessages[reason]; return ok }
func (p *ProbeProgress) valid(mode string, terminal bool) bool {
	if p == nil || p.Mode != mode || probeStageIndex(p.Stage) < 0 || p.ElapsedMS < 0 || p.ElapsedMS > probeMaximumMS || len(p.Checkpoints) < 1 || len(p.Checkpoints) > 16 {
		return false
	}
	if p.Reason == "running" {
		if terminal {
			return false
		}
	} else if !validProbeReason(p.Reason) {
		return false
	}
	if p.Reason == "success" && p.Stage != "complete" || p.Stage == "complete" && p.Reason != "success" {
		return false
	}
	expected := []string{"package_versions", "library_import", "cpu_prerequisites", "cuda_prerequisites"}
	if mode == "snapshot" {
		expected = append(expected, "dependency_inventory", "dependency_hashing")
	}
	expected = append(expected, "complete")
	if len(p.Checkpoints) > len(expected) {
		return false
	}
	lastIndex := -1
	var elapsed, files, hashed, size int64
	for i, o := range p.Checkpoints {
		if o.Stage != expected[i] {
			return false
		}
		index := probeStageIndex(o.Stage)
		if index <= lastIndex || o.ElapsedMS < elapsed || o.ElapsedMS > p.ElapsedMS || o.Files < files || o.Files > 30000 || o.HashedFiles < hashed || o.HashedFiles > o.Files || o.HashedBytes < size || o.HashedBytes > probeMaximumBytes {
			return false
		}
		lastIndex, elapsed, files, hashed, size = index, o.ElapsedMS, o.Files, o.HashedFiles, o.HashedBytes
	}
	last := p.Checkpoints[len(p.Checkpoints)-1]
	return (p.Reason != "success" || len(p.Checkpoints) == len(expected)) && last.ElapsedMS == p.ElapsedMS && last.Stage == p.Stage && p.Files == files && p.HashedFiles == hashed && p.HashedBytes == size
}
func (p *Probe) valid() bool {
	return p != nil && (p.Mode == "inspect" || p.Mode == "snapshot") && validProbeStage(p.Stage) && validProbeReason(p.Reason) && (p.Reason != "success" || p.Stage == "complete" && p.Child != nil) && p.ElapsedMS >= 0 && p.ElapsedMS <= probeMaximumMS && p.BudgetMS >= 0 && p.BudgetMS <= 60000 && (p.Child == nil || p.Child.valid(p.Mode, p.Reason == "success"))
}
func probeContextError(ctx context.Context) error {
	if ctx.Err() == context.Canceled {
		return contracts.Condition("cancelled", "worker_probe_cancelled")
	}
	if ctx.Err() != nil {
		return contracts.Condition("engine_unavailable", "worker_probe_deadline")
	}
	return nil
}
func probeFailure(reason string) error {
	code := "engine_unavailable"
	if reason == "version_mismatch" {
		code = "incompatible_version"
	}
	if reason == "output_limit" {
		code = "output_limit"
	}
	if reason == "cancelled" {
		code = "cancelled"
	}
	return contracts.Condition(code, "worker_probe_"+reason)
}
func readProbeProgress(home, mode string) (*ProbeProgress, error) {
	name := filepath.Join(home, probeCheckpoint)
	st, err := os.Lstat(name)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil || !st.Mode().IsRegular() || st.Size() > probeProgressLimit {
		return nil, contracts.Fail("invalid_request")
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(probeCheckpoint)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, probeProgressLimit+1))
	if err != nil || len(raw) > probeProgressLimit || catalog.ValidateJSON(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	var p ProbeProgress
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || d.Decode(new(any)) != io.EOF || !p.valid(mode, false) {
		return nil, contracts.Fail("invalid_request")
	}
	return &p, nil
}

// Valid validates optional observations read from frozen receipts or work facts.
func (p *Probe) Valid() bool { return p.valid() }

// ContextError preserves cancellation and deadline expiry without claiming byte damage.
func ContextError(ctx context.Context) error { return probeContextError(ctx) }

func UnattemptedInspection(ctx context.Context, role string) Inspection {
	reason := "deadline"
	if ctx.Err() == context.Canceled {
		reason = "cancelled"
	}
	return Inspection{Role: role, Packages: map[string]string{}, Devices: map[string]Device{}, Probe: &Probe{Mode: "inspect", Stage: "not_started", Reason: reason}, Correction: probeMessages[reason]}
}
