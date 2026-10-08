// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shruggietech/insonic/internal/app"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/packageenv"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
)

// QualificationData contains controlled fixture inputs only in the explicit
// native-webview qualification process. Ordinary desktop sessions return none.
func (b *Bridge) NativeQualificationData() contracts.Response {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Result: b.qualification}
}

// QualificationStep emits bounded fixture-only progress, never paths or values.
// Ordinary sessions cannot activate the diagnostic by calling the bound method.
func (b *Bridge) QualificationStep(step string) bool {
	b.mu.RLock()
	active := b.qualification != nil
	b.mu.RUnlock()
	if !active {
		return false
	}
	switch step {
	case "module", "fixtures", "mounted", "ready", "library-import", "audio-playback", "video-playback", "metadata-date", "assembly", "cue-seek", "terms", "speakers", "pipelines", "jobs", "settings", "keyboard-help", "complete":
		fmt.Fprintln(os.Stderr, "Desktop qualification stage:", step)
		return true
	}
	return false
}

// ConfigureQualificationTools supplies source-build companions to a disposable
// test workspace. Extracted packages must exercise their installed defaults.
func ConfigureQualificationTools(w *workspace.Workspace) error {
	_, installed, err := packageenv.Defaults()
	if err != nil || installed {
		return err
	}
	path := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE")
	if path == "" {
		path = filepath.Join("build", "native", "media-tools.json")
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) > 1<<20 {
		return contracts.Fail("unavailable")
	}
	var tools library.Tools
	if json.Unmarshal(raw, &tools) != nil {
		return contracts.Fail("invalid_request")
	}
	if err = os.WriteFile(filepath.Join(w.Control, "media-tools.json"), raw, 0600); err != nil {
		return err
	}
	cueson := os.Getenv("CUESON_EXECUTABLE")
	if cueson == "" {
		return contracts.Fail("unavailable")
	}
	bytes, err := os.ReadFile(cueson)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	digest := sha256.Sum256(bytes)
	config := app.ProcessingTools{Kind: "processing-tools", Version: contracts.Version,
		Cueson: subtitles.Tool{Executable: cueson, ExecutableSHA256: hex.EncodeToString(digest[:])}, Processing: processing.Config{FFmpeg: tools.FFmpeg}}
	if err = app.ValidateProcessingTools(config); err != nil {
		return err
	}
	raw, _ = json.Marshal(config)
	return os.WriteFile(filepath.Join(w.Control, "processing-tools.json"), raw, 0600)
}

// PrepareWebviewQualification admits real media and assembles supplied subtitles
// with deterministic external turns. It does not call recognition/diarization.
func PrepareWebviewQualification(b *Bridge) error {
	root := os.Getenv("INSONIC_DESKTOP_FIXTURE_DIRECTORY")
	if root == "" {
		root = filepath.Join("tests", "fixtures", "media")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	fixtures := map[string]any{"audio_path": filepath.Join(root, "speech.flac"), "video_path": filepath.Join(root, "sintel-dialogue.mkv"), "subtitle_path": filepath.Join(root, "speech.srt"), "fixture_directory": root}
	call := func(operation, item string, data any) contracts.Response {
		var raw json.RawMessage
		if data != nil {
			raw, _ = json.Marshal(data)
		}
		response := b.Operate(contracts.Request{Operation: operation, ItemID: item, Data: raw})
		if response.Error != nil {
			copy := *response.Error
			copy.Message = fmt.Sprintf("%s: %s", operation, copy.Message)
			response.Error = &copy
		}
		return response
	}
	input := library.ImportRequest{Kind: "import-manifest", Version: contracts.Version, Items: []library.Item{{Source: fixtures["audio_path"].(string), Subtitle: fixtures["subtitle_path"].(string), Title: "Qualification audio", NewEntry: true}, {Source: fixtures["video_path"].(string), Title: "Qualification video", NewEntry: true}}}
	admission := call("media.import", "", input)
	if admission.Error != nil {
		return admission.Error
	}
	raw, _ := json.Marshal(admission.Result)
	var queued struct {
		ID string `json:"work_id"`
	}
	if json.Unmarshal(raw, &queued) != nil || !contracts.ValidID(queued.ID) {
		return contracts.Fail("invalid_request")
	}
	deadline := time.Now().Add(45 * time.Second)
	for {
		work := call("work.show", queued.ID, nil)
		if work.Error != nil {
			return work.Error
		}
		raw, _ = json.Marshal(work.Result)
		var state struct {
			State string `json:"state"`
		}
		json.Unmarshal(raw, &state)
		if state.State == "succeeded" {
			break
		}
		if state.State == "failed" || time.Now().After(deadline) {
			return contracts.Fail("operation_failed")
		}
		time.Sleep(100 * time.Millisecond)
	}
	result := call("work.results", queued.ID, nil)
	if result.Error != nil {
		return result.Error
	}
	raw, _ = json.Marshal(result.Result)
	var imported struct {
		Items []library.ItemResult `json:"items"`
	}
	if json.Unmarshal(raw, &imported) != nil || len(imported.Items) != 2 || !contracts.ValidID(imported.Items[0].MediaID) || !contracts.ValidID(imported.Items[1].MediaID) {
		return contracts.Fail("operation_failed")
	}
	audioID := imported.Items[0].MediaID
	assembled := call("recordings.assemble", audioID, app.AssemblyInput{SubtitleFormat: "srt", Turns: []processing.Turn{{Label: "qualification voice", StartUS: 0, EndUS: 1_000_000}}})
	if assembled.Error != nil {
		return assembled.Error
	}
	fixtures["audio_id"] = audioID
	fixtures["video_id"] = imported.Items[1].MediaID
	b.mu.Lock()
	b.qualification = fixtures
	b.mu.Unlock()
	return nil
}
