// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/library"
)

func TestInvocationLimitsBoundCaptureWithoutMutatingSharedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mapped.wav")
	if e := os.WriteFile(path, make([]byte, 44), 0600); e != nil {
		t.Fatal(e)
	}
	original := Config{MaxOutputBytes: 16 << 20, TimeoutMS: 600000}
	elected, e := applyExecutionLimits(original, path, &ExecutionLimits{MaxAudioBytes: 44, MaxResponseBytes: 100, TimeoutMS: 20})
	if e != nil || elected.MaxOutputBytes != 100 || elected.TimeoutMS != 20 || original.MaxOutputBytes != 16<<20 || original.TimeoutMS != 600000 {
		t.Fatal("stage bounds mutated global config or were ignored")
	}
	if _, e = applyExecutionLimits(original, path, &ExecutionLimits{MaxAudioBytes: 43, MaxResponseBytes: 100, TimeoutMS: 20}); e == nil {
		t.Fatal("mapped audio stage limit ignored")
	}
}

func TestConfigurationBoundsAndLazyToolSelection(t *testing.T) {
	if err := ValidateConfig(Config{}); err != nil {
		t.Fatal("omitted inference tools rejected", err)
	}
	selected := library.PinnedFile{Path: filepath.Join(t.TempDir(), "not-installed-python"), SHA256: strings.Repeat("a", 64)}
	if err := ValidateConfig(Config{RecognitionPython: selected}); err != nil {
		t.Fatal("configuration unexpectedly accessed unselected file bytes", err)
	}
	for _, config := range []Config{
		{Threads: -1}, {Threads: 65}, {TimeoutMS: -1}, {TimeoutMS: 86_400_001},
		{MaxInputBytes: -1}, {MaxInputBytes: (64 << 30) + 1},
		{MaxDurationUS: 604_800_000_001}, {MaxOutputBytes: (16 << 20) + 1},
		{RecognitionPython: library.PinnedFile{Path: "relative", SHA256: strings.Repeat("a", 64)}},
		{Worker: library.PinnedFile{Path: selected.Path}},
		{DiarizationPython: library.PinnedFile{SHA256: selected.SHA256}},
		{FFmpeg: library.Tool{Path: selected.Path, SHA256: selected.SHA256}},
	} {
		if err := ValidateConfig(config); err == nil {
			t.Fatalf("invalid configuration admitted: %+v", config)
		}
	}
}
