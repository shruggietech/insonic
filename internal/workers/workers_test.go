// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestProfileClosure(t *testing.T) {
	for _, p := range Inventory().Profiles {
		b, e := assets.ReadFile("assets/" + p.Lock)
		if e != nil || Digest(b) != p.LockSHA256 {
			t.Fatal("lock not bound", p.ID, p.Platform, e)
		}
		var n int64
		seen := map[string]bool{}
		for _, a := range append([]Asset{p.Runtime, p.Installer}, p.Wheels...) {
			if a.Size < 1 || len(a.SHA256) != 64 || !strings.HasPrefix(a.URL, "https://") || seen[a.Name] {
				t.Fatal("invalid asset", a.Name)
			}
			seen[a.Name] = true
			n += a.Size
		}
		if n != p.DownloadBytes {
			t.Fatal("incorrect download budget")
		}
		for _, wheel := range p.Wheels {
			if !strings.HasSuffix(wheel.Name, ".whl") {
				t.Fatal("offline wheel filename is not installable", wheel.Name)
			}
		}
	}
}

func TestGLibcPrerequisitesRejectOldUnknownAndMalformedVersions(t *testing.T) {
	for _, v := range []string{"2.27", "1.99", "", "musl", "bad.28", "2.-1"} {
		if glibcSufficient(v, "2.28") {
			t.Fatal(v)
		}
	}
	for _, v := range []string{"2.28", "2.42", "3.0"} {
		if !glibcSufficient(v, "2.28") {
			t.Fatal(v)
		}
	}
}

func TestCapacityResolvesSameVolumeAndMissingDestinationWithoutCreatingIt(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "not-created", "models")
	shared, e := SameVolume(dir, destination)
	if e != nil || !shared {
		t.Fatal("same volume not detected", shared, e)
	}
	if _, e := FreeSpace(destination); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(dir, "not-created")); !os.IsNotExist(e) {
		t.Fatal("preview created destination")
	}
}
func TestArchiveConfinement(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "C:/escape", "dir/../../escape"} {
		t.Run(name, func(t *testing.T) {
			d := t.TempDir()
			f, e := os.Create(filepath.Join(d, "a.zip"))
			if e != nil {
				t.Fatal(e)
			}
			z := zip.NewWriter(f)
			w, _ := z.Create(name)
			w.Write([]byte("bad"))
			z.Close()
			f.Close()
			out := filepath.Join(d, "out")
			os.Mkdir(out, 0700)
			if extractArchive(context.Background(), filepath.Join(d, "a.zip"), out, "zip") == nil {
				t.Fatal("escaping archive accepted")
			}
		})
	}
}

func TestOwnedArchiveSnapshotSurvivesReplacementAndRejectsUncheckedBytes(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.zip")
	f, _ := os.Create(source)
	z := zip.NewWriter(f)
	file, _ := z.Create("uv.exe")
	file.Write([]byte("verified"))
	z.Close()
	f.Close()
	pin, _ := Hash(context.Background(), source)
	st, _ := os.Stat(source)
	asset := Asset{Name: "source.zip", SHA256: pin.SHA256, Size: st.Size()}
	owned := filepath.Join(t.TempDir(), "owned.zip")
	if e := snapshotAsset(context.Background(), source, owned, asset); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(source, []byte("replacement after verification"), 0600)
	out := t.TempDir()
	if e := extractArchive(context.Background(), owned, out, "zip"); e != nil {
		t.Fatal(e)
	}
	body, _ := os.ReadFile(filepath.Join(out, "uv.exe"))
	if string(body) != "verified" {
		t.Fatal("reopened unchecked source", string(body))
	}
	if e := snapshotAsset(context.Background(), source, filepath.Join(t.TempDir(), "bad.zip"), asset); e == nil {
		t.Fatal("unchecked replacement copied")
	}
}
func TestArchiveEscapingLink(t *testing.T) {
	d := t.TempDir()
	f, _ := os.Create(filepath.Join(d, "a.tgz"))
	g := gzip.NewWriter(f)
	w := tar.NewWriter(g)
	w.WriteHeader(&tar.Header{Name: "python/bin/python", Typeflag: tar.TypeSymlink, Linkname: "../../../outside", Mode: 0755})
	w.Close()
	g.Close()
	f.Close()
	out := filepath.Join(d, "out")
	os.Mkdir(out, 0700)
	if extractArchive(context.Background(), filepath.Join(d, "a.tgz"), out, "tar.gz") == nil {
		t.Fatal("escaping link accepted")
	}
}
func TestInspectionDoesNotInheritSecretsOrLoadWeights(t *testing.T) {
	d := t.TempDir()
	exe := filepath.Join(d, "python")
	os.WriteFile(exe, []byte("fixture"), 0700)
	t.Setenv("HF_TOKEN", "must-not-cross")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "must-not-cross")
	s := Service{Directory: d, Run: func(ctx context.Context, spec process.Spec) (process.CaptureResult, error) {
		if !spec.CleanEnv || strings.Contains(strings.Join(spec.Env, "\n"), "must-not-cross") || len(spec.Args) < 5 || strings.Contains(strings.Join(spec.Args, " "), "model_path") {
			t.Fatal("unsafe probe", spec.Args)
		}
		b, _ := json.Marshal(map[string]any{"python_version": "3.12.15", "cpus": 8, "packages": map[string]string{"faster-whisper": "1.2.1", "ctranslate2": "4.8.2", "av": "18.1.0"}, "engine_available": true, "probe": fixtureProbeProgress("inspect"), "devices": map[string]any{"cpu": map[string]any{"available": true, "dependency_state": "available"}, "cuda": map[string]any{"available": false, "hardware_count": 1, "dependency_state": "missing_libraries"}}})
		return process.CaptureResult{Stdout: b}, nil
	}}
	r, e := s.Inspect(context.Background(), "recognition", exe, false)
	if e != nil || !r.EngineAvailable || !r.Compatible || r.Devices["cuda"].Available || r.Python.SHA256 != Digest([]byte("fixture")) {
		t.Fatal(r, e)
	}
}
func TestBrokenProbeRedactsOutput(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "python")
	os.WriteFile(p, []byte("fixture"), 0700)
	s := Service{Directory: d, Run: func(context.Context, process.Spec) (process.CaptureResult, error) {
		return process.CaptureResult{Stderr: []byte("secret-token")}, contracts.Fail("operation_failed")
	}}
	r, e := s.Inspect(context.Background(), "diarization", p, false)
	if e != nil || r.EngineAvailable || strings.Contains(r.Error+r.Correction, "secret-token") || r.Error == "" {
		t.Fatal(r, e)
	}
}

func TestInspectionBindsConfiguredReceiptBeforeNativeProbe(t *testing.T) {
	d := t.TempDir()
	python := filepath.Join(d, "python")
	if err := os.WriteFile(python, []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	pin, _ := Hash(context.Background(), python)
	receipt := Receipt{Kind: "worker-environment", Version: contracts.Version, Inspection: Inspection{Role: "recognition", Python: pin, Compatible: true, EngineAvailable: true, Files: []library.PinnedFile{pin}}}
	raw, _ := json.Marshal(receipt)
	name := filepath.Join(d, "elected receipt.json")
	if err := os.WriteFile(name, raw, 0600); err != nil {
		t.Fatal(err)
	}
	bound, _ := Hash(context.Background(), name)
	called := false
	s := Service{Directory: d, Run: func(_ context.Context, spec process.Spec) (process.CaptureResult, error) {
		called = true
		if len(spec.Args) != 9 || spec.Args[7] != bound.Path || spec.Args[8] != bound.SHA256 || !strings.Contains(spec.Args[3], "_native.set_receipt(sys.argv[4],sys.argv[5])") || strings.Contains(spec.Args[3], bound.Path) {
			t.Fatal("probe did not bind receipt operands", spec.Args)
		}
		return process.CaptureResult{}, contracts.Fail("operation_failed")
	}}
	if _, err := s.Inspect(context.Background(), "recognition", python, false, bound); err != nil || !called {
		t.Fatal("verified receipt did not reach probe", err)
	}
	called = false
	if err := os.WriteFile(name, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := s.Inspect(context.Background(), "recognition", python, false, bound)
	if err != nil || called || out.EngineAvailable || out.Error != "checksum_mismatch" || out.Probe.Stage != "receipt_verification" {
		t.Fatal("changed receipt reached probe", out, err)
	}
}
func TestEnvironmentReceiptDetectsMutation(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "python")
	os.WriteFile(p, []byte("fixture"), 0700)
	pin, _ := Hash(context.Background(), p)
	r := Receipt{Kind: "worker-environment", Version: contracts.Version, ProfileDigest: strings.Repeat("a", 64), Inspection: Inspection{Role: "recognition", Python: pin, Compatible: true, EngineAvailable: true, Files: []library.PinnedFile{pin}}}
	b, _ := json.Marshal(r)
	path := filepath.Join(d, "receipt.json")
	os.WriteFile(path, b, 0600)
	rp, _ := Hash(context.Background(), path)
	if VerifyReceipt(context.Background(), rp, pin) != nil {
		t.Fatal("valid receipt rejected")
	}
	os.WriteFile(p, []byte("changed"), 0700)
	if VerifyReceipt(context.Background(), rp, pin) == nil {
		t.Fatal("changed environment accepted")
	}
}

func TestEnvironmentReceiptPreservesLoaderAliasBinding(t *testing.T) {
	dir := t.TempDir()
	python, original, replacement := filepath.Join(dir, "python"), filepath.Join(dir, "libcudnn_ops.so.9.1"), filepath.Join(dir, "libcudnn_ops.so.9.2")
	for path, body := range map[string]string{python: "interpreter", original: "original bytes", replacement: "replacement bytes"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(dir, "libcudnn_ops.so.9")
	if err := os.Symlink(original, alias); err != nil {
		t.Skip("file symlinks unavailable", err)
	}
	pythonPin, _ := Hash(context.Background(), python)
	originalPin, _ := Hash(context.Background(), original)
	aliasPin, _ := Hash(context.Background(), alias)
	if aliasPin.Path != alias {
		t.Fatal("hashing collapsed loader alias")
	}
	receipt := Receipt{Kind: "worker-environment", Version: contracts.Version, Inspection: Inspection{Role: "recognition", Python: pythonPin, Compatible: true, EngineAvailable: true, Files: []library.PinnedFile{pythonPin, originalPin, aliasPin}}}
	raw, _ := json.Marshal(receipt)
	path := filepath.Join(dir, "receipt.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	pin, _ := Hash(context.Background(), path)
	if err := VerifyReceipt(context.Background(), pin, pythonPin); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(replacement, alias); err != nil {
		t.Fatal(err)
	}
	if err := VerifyReceipt(context.Background(), pin, pythonPin); contracts.SafeError(err).Code != "checksum_mismatch" {
		t.Fatal("retargeted loader binding accepted", err)
	}
	retained, _ := Hash(context.Background(), original)
	if retained.SHA256 != originalPin.SHA256 {
		t.Fatal("original versioned bytes changed")
	}
}
func TestElectionRejectsAmbiguousAndRelativePaths(t *testing.T) {
	for _, e := range []Election{{Role: "all", Mode: "managed"}, {Role: "recognition", Mode: "reuse", PythonPath: "python"}, {Role: "recognition", Mode: "managed", PythonPath: "ignored"}, {Role: "diarization", Mode: "reuse", PythonPath: filepath.Join(t.TempDir(), "python"), OfflineAssets: "unused"}} {
		if e.Validate() == nil {
			t.Fatal("ambiguous election accepted", e)
		}
	}
}

func TestFailedOfflineAndCancelledSetupRemoveOnlyOwnedAttempt(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		dir := t.TempDir()
		sibling := filepath.Join(dir, "accepted-environment")
		os.Mkdir(sibling, 0700)
		os.WriteFile(filepath.Join(sibling, "keep"), []byte("accepted"), 0600)
		ctx, cancel := context.WithCancel(context.Background())
		service := Service{Directory: dir, Run: func(_ context.Context, spec process.Spec) (process.CaptureResult, error) {
			if filepath.Base(spec.Executable) == "getconf" {
				return process.CaptureResult{Stdout: []byte("glibc 2.42\n")}, nil
			}
			t.Fatal("installer ran without verified assets")
			return process.CaptureResult{}, nil
		}}
		if cancelled {
			service.Progress = func(string, map[string]any) error { cancel(); return nil }
		}
		_, e := service.Prepare(ctx, Election{Role: "recognition", Mode: "managed", OfflineAssets: t.TempDir()})
		cancel()
		if e == nil {
			t.Fatal("failed prerequisites accepted")
		}
		all, _ := os.ReadDir(dir)
		if len(all) != 1 || all[0].Name() != "accepted-environment" {
			t.Fatal("attempt was retained or foreign bytes changed", all)
		}
		body, _ := os.ReadFile(filepath.Join(sibling, "keep"))
		if string(body) != "accepted" {
			t.Fatal("accepted environment changed")
		}
	}
}

func TestOfflineSetupCreatesPrivateWorkerDirectoryBeforeAssetVerification(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "existing"}[existing], func(t *testing.T) {
			w, err := workspace.Init(t.TempDir(), "worker directory fixture")
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(w.Control, "workers")
			if existing {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := workspace.SecureDirectory(dir, true); err != nil {
					t.Fatal(err)
				}
			}
			service := Service{Directory: dir, Run: func(_ context.Context, spec process.Spec) (process.CaptureResult, error) {
				if filepath.Base(spec.Executable) == "getconf" {
					return process.CaptureResult{Stdout: []byte("glibc 2.42\n")}, nil
				}
				t.Fatal("installer ran without verified assets")
				return process.CaptureResult{}, nil
			}}
			_, err = service.Prepare(context.Background(), Election{Role: "recognition", Mode: "managed", OfflineAssets: t.TempDir()})
			observed, ok := err.(*contracts.Error)
			profile, _ := Selected("recognition")
			if !ok || observed.Reason != "asset_missing" || observed.Asset == nil || observed.Asset.Name != profile.Runtime.Name {
				t.Fatalf("setup did not reach declared asset verification: %#v", err)
			}
			if err := workspace.PrivateDirectory(dir); err != nil {
				t.Fatal("worker directory is not private", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatal("failed attempt retained", entries, err)
			}
		})
	}
}

func TestPreviousReleaseWorkerReceiptRetainsIntegrityChecks(t *testing.T) {
	directory := t.TempDir()
	python := filepath.Join(directory, "python")
	if err := os.WriteFile(python, []byte("pinned fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	pin, _ := Hash(context.Background(), python)
	receipt := Receipt{Kind: "worker-environment", Version: "1.0.1", Inspection: Inspection{Role: "recognition", Python: pin, Compatible: true, EngineAvailable: true, Files: []library.PinnedFile{pin}}}
	file := filepath.Join(directory, "receipt.json")
	raw, _ := json.Marshal(receipt)
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	bound, _ := Hash(context.Background(), file)
	if err := VerifyReceipt(context.Background(), bound, pin); err != nil {
		t.Fatal("previous receipt rejected", err)
	}
	after, _ := os.ReadFile(file)
	if string(after) != string(raw) {
		t.Fatal("verification rewrote original receipt")
	}
	if err := os.WriteFile(python, []byte("changed"), 0700); err != nil {
		t.Fatal(err)
	}
	if VerifyReceipt(context.Background(), bound, pin) == nil {
		t.Fatal("old version bypassed full environment hashes")
	}
}
