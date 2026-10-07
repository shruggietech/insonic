// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestCIRefusesBeforeModelMaterializationOrToolSelection(t *testing.T) {
	t.Setenv("CI", "true")
	session := &Session{service: &Service{}}
	_, _, err := session.worker(context.Background(), "transcribe", "invalid-model-id", nil)
	typed, ok := err.(*contracts.Error)
	if !ok || typed.Code != "engine_ci_forbidden" {
		t.Fatalf("CI guard did not run before model and tool access: %v", err)
	}
}

func TestRoleConfinement(t *testing.T) {
	for _, role := range []string{"model.bin", "embedding/pytorch_model.bin", "plda/plda.npz"} {
		if !validRole(role) {
			t.Fatalf("valid role rejected: %s", role)
		}
	}
	for _, role := range []string{"../model", "/model", "a/../model", `C:\model`, "CON", "a/NUL.bin", "model:ads", ".hidden/model", "a//model", "a/"} {
		if validRole(role) {
			t.Fatalf("unsafe role accepted: %s", role)
		}
	}
}

func TestRationalClock(t *testing.T) {
	stream := library.Stream{Index: 2, Kind: "audio", TimeBase: "1/48000"}
	pts := int64(96001)
	stream.StartPTS = &pts
	m, err := sourceClock(stream)
	if err != nil || m.StartNumerator != "96001" || m.StartDenominator != "48000" {
		t.Fatalf("clock: %+v %v", m, err)
	}
	stream.TimeBase = "0/0"
	if _, err := sourceClock(stream); err == nil {
		t.Fatal("invalid timebase accepted")
	}
}

func TestSourceTurnProjection(t *testing.T) {
	turns, diagnostics, err := sourceTurns([]Turn{{Label: "a", StartUS: 250000, EndUS: 1750000}}, SourceMap{StartNumerator: "96001", StartDenominator: "48000"}, nil)
	if err != nil || len(turns) != 1 || turns[0].StartUS != 2250021 || turns[0].EndUS != 3750020 || len(diagnostics) != 0 {
		t.Fatalf("source projection: %+v %+v %v", turns, diagnostics, err)
	}
	turns, diagnostics, err = sourceTurns([]Turn{{Label: "a", StartUS: 0, EndUS: 1}}, SourceMap{StartNumerator: "1", StartDenominator: "3000000"}, nil)
	if err != nil || len(turns) != 0 || len(diagnostics) != 1 {
		t.Fatalf("collapsed interval: %+v %+v %v", turns, diagnostics, err)
	}
}

func TestPCMValidation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "audio.wav")
	header := make([]byte, 44)
	copy(header, "RIFF")
	binary.LittleEndian.PutUint32(header[4:], 44)
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], 16000)
	binary.LittleEndian.PutUint32(header[28:], 32000)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], 8)
	if err := os.WriteFile(p, append(header, make([]byte, 8)...), 0600); err != nil {
		t.Fatal(err)
	}
	count, err := pcmInfo(p, 1000)
	if err != nil || count != 4 {
		t.Fatalf("PCM: %d %v", count, err)
	}
	if _, err := pcmInfo(p, 1); err == nil {
		t.Fatal("duration limit not enforced")
	}
}

func TestMalformedWorkerOutput(t *testing.T) {
	for _, raw := range []string{`{"turns":[{"label":"a","start_us":9,"end_us":1}]}`, `{"turns":[{"label":"a","start_us":0,"end_us":9999}]}`, `{"turns":[],"extra":true}`} {
		if _, err := decodeDiarization([]byte(raw), 1000); err == nil {
			t.Fatalf("malformed output accepted: %s", raw)
		}
	}
}

func TestCancelledToolNeverLaunches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := verifyTool(ctx, library.Tool{Path: "missing"}); err == nil {
		t.Fatal("cancelled missing tool accepted")
	}
}

func TestRecoveryRemovesOnlyOwnedScratch(t *testing.T) {
	control := t.TempDir()
	for _, name := range []string{"processing-123", "processing-456", "processing-not-owned", "unrelated"} {
		if err := os.Mkdir(filepath.Join(control, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(control, "processing-123", ".insonic-processing-session"), []byte("insonic-processing-scratch-v1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	service := Service{Artifacts: &artifact.Service{Workspace: &workspace.Workspace{Control: control}}}
	if err := service.RecoverScratch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(control, "processing-123")); !os.IsNotExist(err) {
		t.Fatalf("owned scratch retained: %v", err)
	}
	for _, name := range []string{"processing-456", "processing-not-owned", "unrelated"} {
		if _, err := os.Stat(filepath.Join(control, name)); err != nil {
			t.Fatalf("unrelated directory changed: %s %v", name, err)
		}
	}
}

func TestSourceStageBoundsIdentityAndCancellation(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source")
	data := []byte("original")
	if err := os.WriteFile(source, data, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if err := stageSource(context.Background(), source, filepath.Join(directory, "stage"), int64(len(data)), digest); err != nil {
		t.Fatal(err)
	}
	if err := stageSource(context.Background(), source, filepath.Join(directory, "bad-size"), 1, digest); err == nil {
		t.Fatal("oversized changed source copied")
	}
	if err := stageSource(context.Background(), source, filepath.Join(directory, "bad-digest"), int64(len(data)), strings.Repeat("0", 64)); err == nil {
		t.Fatal("changed source accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := stageSource(ctx, source, filepath.Join(directory, "cancelled"), int64(len(data)), digest); err == nil {
		t.Fatal("cancelled stage copied")
	}
	if _, err := os.Stat(filepath.Join(directory, "cancelled")); !os.IsNotExist(err) {
		t.Fatal("cancelled stage created output")
	}
}

// The native harness elects actual bounded decoding only. This test has no
// inference or model setup path, including when tools are already cached.
func TestMappedFixtureDecode(t *testing.T) {
	toolsPath := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE")
	if toolsPath == "" {
		t.Skip("exact media tools elected by native harness")
	}
	raw, err := os.ReadFile(toolsPath)
	if err != nil {
		t.Fatal(err)
	}
	var tools struct {
		FFmpeg library.Tool `json:"ffmpeg"`
	}
	if json.Unmarshal(raw, &tools) != nil || tools.FFmpeg.Path == "" {
		t.Fatal("FFmpeg pin absent")
	}
	input, err := filepath.Abs(filepath.Join("..", "..", "tests", "fixtures", "media", "speech.flac"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(input); err != nil {
		t.Skip("fixture preparation not complete")
	}
	output := filepath.Join(t.TempDir(), "mapped.wav")
	count, err := decodeAudio(context.Background(), tools.FFmpeg, input, output, 0, nil, Config{MaxDurationUS: 60_000_000, TimeoutMS: 30_000, Threads: 2})
	if err != nil || count <= 0 {
		t.Fatalf("decode: %d %v", count, err)
	}
	if _, err = pcmInfo(output, 60_000_000); err != nil {
		t.Fatal(err)
	}
}
