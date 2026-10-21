// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/voicemodels"
	"github.com/shruggietech/insonic/internal/workers"
	"os"
	"path/filepath"
	"testing"
)

func TestCustomTrainerChecksPinnedFilesWithoutExecuting(t *testing.T) {
	a := configuredApp(t)
	workerToolsFixture(t, a)
	path := filepath.Join(t.TempDir(), "never-execute")
	os.WriteFile(path, []byte("fixture bytes are not an executable"), 0700)
	pin, _ := workers.Hash(context.Background(), path)
	adapter := voicemodels.Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "local", Architecture: "fixture", OutputKinds: []string{"speaker-model"}, Consumers: []string{"fixture"}, Executable: pin, SupportFiles: []library.PinnedFile{pin}}
	for _, mode := range []string{"valid", "changed", "missing-support"} {
		t.Run(mode, func(t *testing.T) {
			selected := adapter
			if mode == "changed" {
				selected.Executable.SHA256 = workers.Digest([]byte("different"))
			}
			if mode == "missing-support" {
				selected.SupportFiles = []library.PinnedFile{{Path: filepath.Join(t.TempDir(), "missing"), SHA256: pin.SHA256}}
			}
			raw, _ := json.Marshal(voicemodels.TrainingConfiguration{Adapter: selected, Kind: "speaker-model"})
			r, e := a.readinessDefinition(a.ctx, ReadinessInput{}, &pipeline.Definition{SpeakerTraining: raw})
			if e != nil {
				t.Fatal(e)
			}
			var got Prerequisite
			for _, p := range r.Tasks["training"].Prerequisites {
				if p.Name == "elected training adapter" {
					got = p
				}
			}
			if got.Available != (mode == "valid") {
				t.Fatal(mode, got)
			}
			if mode == "valid" && got.State != "file_verified" {
				t.Fatal(got)
			}
		})
	}
}

func TestUnknownPrerequisiteIsDistinctFromUnavailable(t *testing.T) {
	got := composeTask([]Prerequisite{{Name: "remote", State: "configured_not_probed"}})
	if got.Usable || got.State != "unknown" {
		t.Fatal(got)
	}
	got = composeTask([]Prerequisite{{Name: "remote", State: "configured_not_probed"}, {Name: "file", State: "not_configured"}})
	if got.State != "needs_attention" {
		t.Fatal(got)
	}
}

func TestPipelineReadinessUsesElectedTasks(t *testing.T) {
	tasks := map[string]TaskReadiness{"transcription": {Usable: false}, "diarization": {Usable: false}, "training": {Usable: true}}
	d := &pipeline.Definition{SpeakerTraining: json.RawMessage(`{}`)}
	if !electedPipelineUsable(d, tasks) {
		t.Fatal("training-only required unelected processing")
	}
	tasks["training"] = TaskReadiness{}
	if electedPipelineUsable(d, tasks) {
		t.Fatal("unready training admitted")
	}
	if electedPipelineUsable(nil, tasks) || electedPipelineUsable(&pipeline.Definition{}, tasks) {
		t.Fatal("no pipeline elected")
	}
}
