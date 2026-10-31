// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/workers"
)

func TestNewLocalAdmissionFreezesAutomaticDevice(t *testing.T) {
	a := configuredApp(t)
	for _, device := range []string{"", "cpu", "cuda", "auto"} {
		o, election, err := a.electRecordingOptions(RecordingOptions{Transcription: "generate", Diarization: "run", Recognition: processing.RecognitionOptions{Device: device}, Attribution: processing.DiarizationOptions{Device: device}})
		want := device
		if want == "" {
			want = "auto"
		}
		if err != nil || o.Recognition.Device != want || o.Attribution.Device != want || election.Definition.Recognition.Recognition.Device != want {
			t.Fatalf("device=%q options=%+v election=%+v err=%v", device, o, election, err)
		}
		if err := validateElection(recordingPayload{Options: o, Election: election}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLocalLanguageRejectedBeforeWorkAdmission(t *testing.T) {
	a := configuredApp(t)
	_, _, err := a.electRecordingOptions(RecordingOptions{Transcription: "generate", Recognition: processing.RecognitionOptions{Language: "xx"}})
	if err == nil {
		t.Fatal("unsupported local language admitted")
	}
	safe := contracts.SafeError(err)
	if safe.Validation == nil || safe.Validation.Pointer != "/recognition/language" {
		t.Fatalf("lost correction: %+v", safe)
	}
	// The general option shape remains usable by elected hosted adapters.
	if _, err := processing.ValidateRecognitionOptions(processing.RecognitionOptions{Language: "xx"}, 200); err != nil {
		t.Fatalf("hosted shape narrowed: %v", err)
	}
}

func TestLegacyElectedPayloadKeepsOmittedDevice(t *testing.T) {
	raw := []byte(`{"options":{"transcription":"generate","recognition":{"language":"en"}}}`)
	var p recordingPayload
	if json.Unmarshal(raw, &p) != nil || p.Options.Recognition.Device != "" {
		t.Fatal("legacy payload altered")
	}
	if effectiveRecognitionStage(pipeline.Stage{Adapter: "insonic-http", Mode: "hosted"}).Recognition.Device != "" {
		t.Fatal("hosted selection altered")
	}
}

func TestAutomaticReadinessReportsCPUFallbackWithoutClaimingCUDA(t *testing.T) {
	devices := map[string]workers.Device{"cuda": {DependencyState: "missing_cuda_libraries"}, "cpu": {Available: true, DependencyState: "libraries_available"}}
	for _, policy := range []string{"", "auto", "cuda"} {
		got := devicePrerequisite(policy, devices)
		if !got.Available || got.State != "cpu_fallback_available" || got.Correction == "" {
			t.Fatalf("%s: %+v", policy, got)
		}
	}
	got := devicePrerequisite("cpu", devices)
	if !got.Available || got.State != "libraries_available" {
		t.Fatal(got)
	}
}

func TestNewImportFreezesDiarizationPolicy(t *testing.T) {
	a := configuredApp(t)
	for _, device := range []string{"", "cpu", "cuda", "auto"} {
		selection, err := a.freezeImportSelection(library.Options{}, library.Options{Attribution: "diarize", DiarizationDevice: device}, library.Options{}, true)
		want := device
		if want == "" {
			want = "auto"
		}
		if err != nil || selection.Options.DiarizationDevice != want {
			t.Fatalf("%s: %+v %v", device, selection, err)
		}
	}
	var historical library.ImportRequest
	if json.Unmarshal([]byte(`{"items":[{"source":"old.wav","attribution":"diarize"}]}`), &historical) != nil || historical.Items[0].DiarizationDevice != "" {
		t.Fatal("old import policy changed")
	}
}

func TestTrainingReadinessUsesElectedDevice(t *testing.T) {
	a := configuredApp(t)
	for _, device := range []string{"cpu", "cuda", "auto"} {
		definition := pipeline.Definition{SpeakerTraining: json.RawMessage("{\"adapter\":{\"id\":\"pyannote-profile\"},\"device\":\"" + device + "\"}")}
		result, err := a.readinessDefinition(context.Background(), ReadinessInput{}, &definition)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, prerequisite := range result.Tasks["training"].Prerequisites {
			if prerequisite.Name == device+" device policy" {
				found = true
			}
		}
		if !found {
			t.Fatalf("training readiness ignored %s: %+v", device, result.Tasks["training"].Prerequisites)
		}
	}
}
