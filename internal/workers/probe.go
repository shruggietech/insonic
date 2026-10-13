// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
)

func (s Service) Inspect(ctx context.Context, role, python string, snapshot bool) (Inspection, error) {
	out := Inspection{Role: role, Packages: map[string]string{}, Devices: map[string]Device{}, Correction: "Choose managed CPU setup or inspect a compatible installed environment."}
	if role != "recognition" && role != "diarization" {
		return out, contracts.Fail("invalid_request")
	}
	if python == "" {
		out.Error = "engine_unavailable"
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pin, e := Hash(ctx, python)
	if e != nil {
		out.Error = "engine_unavailable"
		return out, nil
	}
	out.Python = pin
	home, e := os.MkdirTemp(s.Directory, "worker-probe-")
	if e != nil {
		return out, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(home)
	if e = workspace.SecureDirectory(home, true); e != nil {
		return out, e
	}
	code, _ := assets.ReadFile("probe.py")
	mode := "inspect"
	if snapshot {
		mode = "snapshot"
	}
	env := append(process.LocalEnvironment(), "HOME="+home, "USERPROFILE="+home, "HF_HUB_OFFLINE=1", "TRANSFORMERS_OFFLINE=1", "HF_HUB_DISABLE_TELEMETRY=1", "PYANNOTE_METRICS_ENABLED=0", "PYTHONIOENCODING=utf-8", "TOKENIZERS_PARALLELISM=false")
	run := s.Run
	if run == nil {
		run = process.Capture
	}
	r, e := run(ctx, process.Spec{Executable: python, Args: []string{"-I", "-B", "-c", string(code), role, mode}, Directory: home, CleanEnv: true, Env: env, MaxOutput: 16 << 20})
	if ctx.Err() != nil && ctx.Err() == context.Canceled {
		return out, contracts.Fail("cancelled")
	}
	if e != nil {
		out.Error = "engine_unavailable"
		return out, nil
	}
	var got struct {
		PythonVersion   string               `json:"python_version"`
		Packages        map[string]string    `json:"packages"`
		Devices         map[string]Device    `json:"devices"`
		CPUs            int                  `json:"cpus"`
		EngineAvailable bool                 `json:"engine_available"`
		Files           []library.PinnedFile `json:"files"`
	}
	d := json.NewDecoder(bytes.NewReader(r.Stdout))
	d.DisallowUnknownFields()
	if d.Decode(&got) != nil || d.Decode(new(any)) != io.EOF || got.CPUs < 1 || len(got.Files) > 30000 {
		out.Error = "engine_unavailable"
		return out, nil
	}
	out.PythonVersion, out.Packages, out.Devices, out.CPUs, out.EngineAvailable, out.Files = got.PythonVersion, got.Packages, got.Devices, got.CPUs, got.EngineAvailable, got.Files
	out.Compatible = got.EngineAvailable && ((role == "recognition" && got.Packages["faster-whisper"] == "1.2.1" && got.Packages["ctranslate2"] == "4.8.2") || (role == "diarization" && got.Packages["pyannote-audio"] == "4.0.7"))
	if !out.Compatible {
		out.Error = "incompatible_version"
		if !got.EngineAvailable {
			out.Error = "engine_unavailable"
		}
	} else {
		out.Correction = ""
	}
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
	if pin.Path == "" && pin.SHA256 == "" {
		return nil
	}
	actual, e := Hash(ctx, pin.Path)
	if e != nil || actual.SHA256 != pin.SHA256 {
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
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF || r.Kind != "worker-environment" || r.Version != contracts.Version || !r.Inspection.EngineAvailable || !r.Inspection.Compatible || r.Inspection.Python.SHA256 != python.SHA256 || !strings.EqualFold(filepath.Clean(r.Inspection.Python.Path), filepath.Clean(python.Path)) || len(r.Inspection.Files) < 1 || len(r.Inspection.Files) > 30000 {
		return contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	for _, p := range r.Inspection.Files {
		if seen[p.Path] || len(p.SHA256) != 64 {
			return contracts.Fail("invalid_request")
		}
		seen[p.Path] = true
		actual, e := Hash(ctx, p.Path)
		if e != nil || actual.SHA256 != p.SHA256 {
			return contracts.Fail("checksum_mismatch")
		}
	}
	return nil
}

// ExistingManaged reuses only an unchanged previously accepted managed CPU
// environment with this exact profile. Reuse never installs into old paths.
func ExistingManaged(ctx context.Context, pin, python library.PinnedFile, p Profile) (Prepared, bool) {
	if pin.Path == "" || VerifyReceipt(ctx, pin, python) != nil {
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
