// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/schemas"
)

func TestProcessingToolsOptionalConfigSerializesToPublishedSchema(t *testing.T) {
	config := ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: subtitles.Tool{Executable: filepath.Join(t.TempDir(), "cueson"), ExecutableSHA256: strings.Repeat("0", 64)}}
	raw, e := json.Marshal(config)
	if e != nil || schemas.ValidateDocument(raw) != nil {
		t.Fatal("absent optional tools serialized as invalid pins", string(raw), e)
	}
	config.Processing.Worker.Path = "partial-pin"
	raw, _ = json.Marshal(config)
	if schemas.ValidateDocument(raw) == nil {
		t.Fatal("explicit partial pin disappeared during serialization")
	}
}

func TestNativeAssemblySuppliedFixtureWithNonemptyTurns(t *testing.T) {
	mediaConfig, cueson := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE"), os.Getenv("CUESON_EXECUTABLE")
	if mediaConfig == "" || cueson == "" {
		t.Skip("exact native media and Cueson tools not selected")
	}
	raw, e := os.ReadFile(mediaConfig)
	if e != nil {
		t.Fatal(e)
	}
	var tools library.Tools
	if e = json.Unmarshal(raw, &tools); e != nil {
		t.Fatal(e)
	}
	binary, e := os.ReadFile(cueson)
	if e != nil {
		t.Fatal(e)
	}
	digest := sha256.Sum256(binary)
	config := ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: subtitles.Tool{Executable: cueson, ExecutableSHA256: hex.EncodeToString(digest[:])}, Processing: processing.Config{FFmpeg: tools.FFmpeg}}
	a := configuredApp(t)
	if e = os.WriteFile(filepath.Join(a.Workspace.Control, "media-tools.json"), raw, 0600); e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(config)
	if e = os.WriteFile(filepath.Join(a.Workspace.Control, "processing-tools.json"), encoded, 0600); e != nil {
		t.Fatal(e)
	}
	source, _ := filepath.Abs("../../tests/fixtures/media/speech.flac")
	srt, _ := filepath.Abs("../../tests/fixtures/media/speech.srt")
	admission := realRequest(a, "media.import", "", library.ImportRequest{Kind: "import-manifest", Version: contracts.Version, Items: []library.Item{{Source: source, Subtitle: srt, NewEntry: true}}})
	if admission.Error != nil {
		t.Fatal(admission.Error)
	}
	done := awaitWork(t, a, admission.Result.(map[string]any)["work_id"].(string))
	var result library.ImportResult
	if e = json.Unmarshal(done.Result, &result); e != nil || len(result.Items) != 1 || result.Items[0].MediaID == "" {
		t.Fatal("native import", done.State, string(done.Result), e)
	}
	id := result.Items[0].MediaID
	for _, turn := range []processing.Turn{{Label: "qualification voice", StartUS: 0, EndUS: 1_000_000}, {Label: "replacement voice", StartUS: 2_500_000, EndUS: 2_750_000}} {
		out := realRequest(a, "recordings.assemble", id, AssemblyInput{SubtitleFormat: "srt", Turns: []processing.Turn{turn}})
		if out.Error != nil {
			t.Fatal("native supplied assembly", turn, out.Error)
		}
		current, e := a.Catalog.Recording(context.Background(), id)
		if e != nil || current.State != "ready" || subtitles.ValidateDocument(current.Document) != nil {
			t.Fatal("assembly did not publish current valid cue document", current.State, e)
		}
		var document struct {
			Cues []struct {
				Attributions []struct {
					Start int64 `json:"start_milliseconds"`
					End   int64 `json:"end_milliseconds"`
				} `json:"speaker_attributions"`
			} `json:"cues"`
		}
		if json.Unmarshal(current.Document, &document) != nil || len(document.Cues) != 1 || len(document.Cues[0].Attributions) != 1 || document.Cues[0].Attributions[0].Start != turn.StartUS/1000 || document.Cues[0].Attributions[0].End != turn.EndUS/1000 {
			t.Fatal("native turn assignment was not retained")
		}
	}
}
