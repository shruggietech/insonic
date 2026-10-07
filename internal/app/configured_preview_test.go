// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/speakers"
	"reflect"
	"testing"
)

func TestConfiguredPreviewMatchesElectedHintsAndDefaultLanguage(t *testing.T) {
	a := configuredApp(t)
	ctx := context.Background()
	for _, value := range []struct{ language, text string }{{"en", "EnglishScopedTerm"}, {"fr", "FrenchScopedTerm"}} {
		if _, err := a.Catalog.PutTerm(ctx, contracts.ID(), 0, catalog.Term{ID: contracts.ID(), Canonical: value.text, Language: value.language, Context: "project"}); err != nil {
			t.Fatal(err)
		}
	}
	definition := localDefinition()
	definition["recognition"].(map[string]any)["recognition"] = map[string]any{"language": "en", "hints": []string{"ConfiguredHint"}}
	pipelineID := contracts.ID()
	raw, _ := json.Marshal(definition)
	saved, err := a.Catalog.PutPipeline(ctx, contracts.ID(), 0, catalog.Pipeline{ID: pipelineID, Name: "Preview", Preset: "local", Configuration: raw})
	if err != nil {
		t.Fatal(err)
	}
	override := pipeline.Stage{Adapter: "faster-whisper", Version: "1", Mode: "local", ModelID: contracts.ID(), Recognition: processing.RecognitionOptions{Language: "fr", Hints: []string{"OverrideHint"}}}
	for _, fixture := range []struct {
		name     string
		override pipeline.Override
		want     []string
	}{{"saved", pipeline.Override{}, []string{"ConfiguredHint", "EnglishScopedTerm"}}, {"overridden", pipeline.Override{Recognition: &override}, []string{"OverrideHint", "FrenchScopedTerm"}}} {
		t.Run(fixture.name, func(t *testing.T) {
			filter := catalog.ContextFilter{Context: "project"}
			options, election, err := a.electRecordingOptions(RecordingOptions{PipelineID: pipelineID, PipelineRevision: saved.Revision, Transcription: "generate", Overrides: fixture.override, Context: filter})
			if err != nil {
				t.Fatal(err)
			}
			response := configuredResult(t, realRequest(a, "pipelines.inspect", pipelineID, map[string]any{"revision": saved.Revision, "overrides": fixture.override, "context": filter}))
			raw, _ := json.Marshal(response["context"])
			var preview speakers.CompiledContext
			if json.Unmarshal(raw, &preview) != nil {
				t.Fatal("invalid preview context")
			}
			if !reflect.DeepEqual(preview.Hints, fixture.want) || !reflect.DeepEqual(preview.Hints, options.Recognition.Hints) || preview.Digest != election.Context.Digest || preview.SnapshotDigest != election.Context.SnapshotDigest {
				t.Fatalf("preview and election differ: %+v %+v", preview, election.Context)
			}
			raw, _ = json.Marshal(response["configuration"])
			var effective pipeline.Definition
			if json.Unmarshal(raw, &effective) != nil {
				t.Fatal("invalid effective configuration")
			}
			if !reflect.DeepEqual(effective.Recognition.Recognition.Hints, options.Recognition.Hints) || effective.Recognition.Recognition.ContextDigest != options.Recognition.ContextDigest {
				t.Fatal("reported recognition configuration differs from election")
			}
		})
	}
	compiled, err := a.compileRecognitionContext(compileInput{PipelineID: pipelineID, PipelineRevision: saved.Revision, Context: "project"})
	if err != nil {
		t.Fatal(err)
	}
	_, election, err := a.electRecordingOptions(RecordingOptions{PipelineID: pipelineID, Transcription: "generate", Context: catalog.ContextFilter{Context: "project"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(compiled.Hints, election.Context.Hints) || compiled.SnapshotDigest != election.Context.SnapshotDigest {
		t.Fatal("term compilation differs from saved election", compiled, election.Context)
	}
	if _, err = a.compileRecognitionContext(compileInput{PipelineID: pipelineID, MaxHintBytes: 8193}); err == nil {
		t.Fatal("oversized explicit budget silently clamped")
	}
}
