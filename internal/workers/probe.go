// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
)

func (s Service) Inspect(parent context.Context, role, python string, snapshot bool) (out Inspection, err error) {
	out = Inspection{Role: role, Packages: map[string]string{}, Devices: map[string]Device{}}
	if role != "recognition" && role != "diarization" {
		return out, contracts.Fail("invalid_request")
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	mode := "inspect"
	if snapshot {
		mode = "snapshot"
	}
	budget := int64(0)
	if deadline, ok := ctx.Deadline(); ok {
		budget = max(0, min(60000, time.Until(deadline).Milliseconds()))
	}
	out.Probe = &Probe{Mode: mode, Stage: "interpreter_hash", Reason: "interpreter_missing", BudgetMS: budget}
	defer func() { out.Probe.ElapsedMS = max(0, time.Since(started).Milliseconds()) }()
	fail := func(stage, reason string) (Inspection, error) {
		out.Probe.Stage, out.Probe.Reason = stage, reason
		out.EngineAvailable, out.Compatible = false, false
		out.Files = nil
		out.Error = "engine_unavailable"
		if reason == "version_mismatch" {
			out.Error = "incompatible_version"
		}
		out.Correction = probeMessages[reason]
		if reason == "cancelled" {
			return out, probeFailure(reason)
		}
		return out, nil
	}
	contextResult := func() (Inspection, error) {
		reason := "deadline"
		if ctx.Err() == context.Canceled {
			reason = "cancelled"
		}
		return fail(out.Probe.Stage, reason)
	}
	if ctx.Err() != nil {
		return contextResult()
	}
	if python == "" {
		return fail("interpreter_hash", "interpreter_missing")
	}
	pin, e := Hash(ctx, python)
	if ctx.Err() != nil {
		return contextResult()
	}
	if e != nil {
		return fail("interpreter_hash", "interpreter_unreadable")
	}
	out.Python = pin
	out.Probe.Stage = "private_home"
	home, e := os.MkdirTemp(s.Directory, "worker-probe-")
	if e != nil {
		return fail("private_home", "home_unavailable")
	}
	defer func() {
		if cleanupErr := os.RemoveAll(home); cleanupErr != nil {
			out.Probe.Stage = "private_home"
			out.Probe.Reason = "home_unavailable"
			out.Compatible = false
			out.EngineAvailable = false
			out.Error = "engine_unavailable"
			out.Correction = probeMessages["home_unavailable"]
		}
	}()
	if e = workspace.SecureDirectory(home, true); e != nil {
		return fail("private_home", "home_unavailable")
	}
	if ctx.Err() != nil {
		return contextResult()
	}
	code, _ := assets.ReadFile("probe.py")
	env := append(process.LocalEnvironment(), "HOME="+home, "USERPROFILE="+home, "HF_HUB_OFFLINE=1", "TRANSFORMERS_OFFLINE=1", "HF_HUB_DISABLE_TELEMETRY=1", "PYANNOTE_METRICS_ENABLED=0", "PYTHONIOENCODING=utf-8", "TOKENIZERS_PARALLELISM=false")
	run := s.Run
	if run == nil {
		run = process.Capture
	}
	out.Probe.Stage = "process_start"
	r, e := run(ctx, process.Spec{Executable: python, Args: []string{"-I", "-B", "-c", string(code), role, mode, filepath.Join(home, probeCheckpoint)}, Directory: home, CleanEnv: true, Env: env, MaxOutput: 16 << 20})
	progress, progressErr := readProbeProgress(home, mode)
	if progressErr == nil && progress != nil {
		out.Probe.Child = progress
		out.Probe.Stage = progress.Stage
	}
	if ctx.Err() != nil {
		return contextResult()
	}
	if progressErr != nil {
		return fail("output_validation", "checkpoint_invalid")
	}
	if r.Truncated {
		return fail("output_validation", "output_limit")
	}
	if e != nil {
		reason := "process_failed"
		if progress == nil && !r.Started {
			reason = "launch_failed"
		}
		return fail(out.Probe.Stage, reason)
	}
	out.Probe.Stage = "output_validation"
	var got struct {
		PythonVersion   string               `json:"python_version"`
		Packages        map[string]string    `json:"packages"`
		Devices         map[string]Device    `json:"devices"`
		CPUs            int                  `json:"cpus"`
		EngineAvailable bool                 `json:"engine_available"`
		Files           []library.PinnedFile `json:"files"`
		Probe           *ProbeProgress       `json:"probe"`
	}
	d := json.NewDecoder(bytes.NewReader(r.Stdout))
	d.DisallowUnknownFields()
	if catalog.ValidateJSON(r.Stdout) != nil || d.Decode(&got) != nil || d.Decode(new(any)) != io.EOF || got.CPUs < 1 || len(got.Files) > 30000 || !got.Probe.valid(mode, true) {
		return fail("output_validation", "output_invalid")
	}
	if progress != nil && !reflect.DeepEqual(progress, got.Probe) {
		return fail("output_validation", "checkpoint_invalid")
	}
	out.Probe.Child = got.Probe
	out.PythonVersion, out.Packages, out.Devices, out.CPUs, out.EngineAvailable, out.Files = got.PythonVersion, got.Packages, got.Devices, got.CPUs, got.EngineAvailable, got.Files
	if out.Devices == nil {
		out.Devices = map[string]Device{}
	}
	if got.Probe.Reason != "success" {
		return fail(got.Probe.Stage, got.Probe.Reason)
	}
	if !got.EngineAvailable {
		return fail(got.Probe.Stage, "output_invalid")
	}
	if snapshot && (len(got.Files) == 0 || int64(len(got.Files)) != got.Probe.Files || got.Probe.HashedFiles != got.Probe.Files) {
		return fail("output_validation", "output_invalid")
	}
	out.Compatible = got.EngineAvailable && ((role == "recognition" && got.Packages["faster-whisper"] == "1.2.1" && got.Packages["ctranslate2"] == "4.8.2") || (role == "diarization" && got.Packages["pyannote-audio"] == "4.0.7"))
	if !out.Compatible {
		return fail("package_versions", "version_mismatch")
	}
	if ctx.Err() != nil {
		return contextResult()
	}
	out.Probe.Stage, out.Probe.Reason = "complete", "success"
	cuda := out.Devices["cuda"]
	if !cuda.Available {
		cuda.Correction = "Use a verified compatible CUDA environment, or explicitly save CPU for future work."
	}
	out.Devices["cuda"] = cuda
	return out, nil
}

// VerifyReceipt checks the frozen dependency bytes before elected engine import.
// Existing advanced tools without a receipt remain supported.
func VerifyReceipt(ctx context.Context, pin, python library.PinnedFile) error {
	if err := probeContextError(ctx); err != nil {
		return err
	}
	if pin.Path == "" && pin.SHA256 == "" {
		return nil
	}
	actual, e := Hash(ctx, pin.Path)
	if err := probeContextError(ctx); err != nil {
		return err
	}
	if e != nil {
		return e
	}
	if actual.SHA256 != pin.SHA256 {
		return contracts.Fail("checksum_mismatch")
	}
	f, e := os.Open(pin.Path)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (16<<20)+1))
	if e != nil || len(b) > 16<<20 || catalog.ValidateJSON(b) != nil {
		return contracts.Fail("invalid_request")
	}
	var r Receipt
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF || r.Kind != "worker-environment" || !contracts.SupportedDocumentVersion(r.Version) || !r.Inspection.EngineAvailable || !r.Inspection.Compatible || r.Inspection.Python.SHA256 != python.SHA256 || !strings.EqualFold(filepath.Clean(r.Inspection.Python.Path), filepath.Clean(python.Path)) || len(r.Inspection.Files) < 1 || len(r.Inspection.Files) > 30000 || r.Inspection.Probe != nil && (!r.Inspection.Probe.valid() || r.Inspection.Probe.Reason != "success") {
		return contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	buffer := make([]byte, 128<<10)
	for _, p := range r.Inspection.Files {
		if seen[p.Path] || len(p.SHA256) != 64 {
			return contracts.Fail("invalid_request")
		}
		seen[p.Path] = true
		actual, e := hashWithBuffer(ctx, p.Path, buffer)
		if err := probeContextError(ctx); err != nil {
			return err
		}
		if e != nil {
			return e
		}
		if actual.SHA256 != p.SHA256 {
			return contracts.Fail("checksum_mismatch")
		}
	}
	return nil
}

// ExistingManaged reuses only an unchanged previously accepted managed CPU
// environment with this exact profile. Reuse never installs into old paths.
func ExistingManaged(ctx context.Context, pin, python library.PinnedFile, p Profile) (Prepared, bool) {
	if ctx.Err() != nil || pin.Path == "" || VerifyReceipt(ctx, pin, python) != nil {
		return Prepared{}, false
	}
	b, e := os.ReadFile(pin.Path)
	if e != nil {
		return Prepared{}, false
	}
	var r Receipt
	if json.Unmarshal(b, &r) != nil || r.Mode != "managed" || r.ProfileDigest != p.Digest() || r.Inspection.Role != p.ID || r.Inspection.PythonVersion != p.Python {
		return Prepared{}, false
	}
	for name, version := range p.Engines {
		if r.Inspection.Packages[name] != version {
			return Prepared{}, false
		}
	}
	r.Inspection.Files = nil
	return Prepared{Python: python, Receipt: pin, Inspection: r.Inspection}, true
}
