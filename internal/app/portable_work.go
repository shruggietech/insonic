// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/voicemodels"
)

// portableInvocation is ephemeral. The catalog continues to journal the proven
// source-free election, so rebinding cannot turn target paths into backup data.
func (a *App) bindPortableConfiguration(input json.RawMessage) (json.RawMessage, error) {
	var doc map[string]any
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if catalog.ValidateJSON(input) != nil || decoder.Decode(&doc) != nil || decoder.Decode(new(any)) != io.EOF {
		return nil, contracts.Fail("invalid_request")
	}
	bindings := map[string]string{}
	var contains func(any) bool
	contains = func(v any) bool {
		switch x := v.(type) {
		case string:
			return strings.Contains(x, "<portable-file:")
		case map[string]any:
			for _, item := range x {
				if contains(item) {
					return true
				}
			}
		case []any:
			for _, item := range x {
				if contains(item) {
					return true
				}
			}
		}
		return false
	}
	var collect func(any)
	collect = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if p, ok := x["path"].(string); ok && filepath.IsAbs(p) {
				if d, ok := x["sha256"].(string); ok && len(d) == 64 {
					bindings[catalog.PortableFile(d)] = p
				}
			}
			if p, ok := x["executable"].(string); ok && filepath.IsAbs(p) {
				if d, ok := x["executable_sha256"].(string); ok && len(d) == 64 {
					bindings[catalog.PortableFile(d)] = p
				}
			}
			for _, item := range x {
				collect(item)
			}
		case []any:
			for _, item := range x {
				collect(item)
			}
		}
	}
	if contains(doc) {
		// Read only local elected configuration. Exact digests, not source names
		// or search paths, select each replacement.
		c, e := ReadProcessingTools(a.Workspace)
		if e != nil {
			return nil, e
		}
		media, mediaError := ReadMediaTools(a.Workspace)
		if c.Processing.FFmpeg.Path == "" && mediaError == nil {
			c.Processing.FFmpeg = media.FFmpeg
		}
		var configured any
		raw, _ := json.Marshal(c)
		json.Unmarshal(raw, &configured)
		collect(configured)
		// Standard tool roles use the destination's validated platform binaries.
		// Models, stage settings and custom adapter/support pins stay frozen.
		for _, key := range []string{"tools", "processing_tools"} {
			if elected, ok := doc[key].(map[string]any); ok {
				if source, ok := elected["cueson"].(map[string]any); ok && contains(source) {
					raw, _ := json.Marshal(c.Cueson)
					var target any
					json.Unmarshal(raw, &target)
					elected["cueson"] = target
				}
				if source, ok := elected["processing"].(map[string]any); ok {
					for role, target := range map[string]any{"ffmpeg": c.Processing.FFmpeg, "recognition_python": c.Processing.RecognitionPython, "diarization_python": c.Processing.DiarizationPython, "worker": c.Processing.Worker} {
						if pin, ok := source[role]; ok && contains(pin) {
							switch chosen := target.(type) {
							case library.PinnedFile:
								if e := verifyPortablePin(chosen); e != nil {
									return nil, e
								}
							case library.Tool:
								if e := verifyPortablePin(library.PinnedFile{Path: chosen.Path, SHA256: chosen.SHA256}); e != nil {
									return nil, e
								}
								for _, support := range chosen.SupportFiles {
									if e := verifyPortablePin(support); e != nil {
										return nil, e
									}
								}
								if chosen.Interpreter != nil {
									if e := verifyPortablePin(*chosen.Interpreter); e != nil {
										return nil, e
									}
								}
							}
							raw, _ := json.Marshal(target)
							var value any
							json.Unmarshal(raw, &value)
							source[role] = value
						}
					}
				}
			}
		}
		if mediaError == nil {
			raw, _ := json.Marshal(media)
			json.Unmarshal(raw, &configured)
			collect(configured)
		}
	}
	var bind func(any) error
	bind = func(v any) error {
		switch x := v.(type) {
		case map[string]any:
			for key, item := range x {
				if text, ok := item.(string); ok {
					for token, path := range bindings {
						text = strings.ReplaceAll(text, token, path)
					}
					if strings.Contains(text, "<portable-file:") {
						return contracts.Fail("unavailable")
					}
					x[key] = text
				} else if e := bind(item); e != nil {
					return e
				}
			}
		case []any:
			for i, item := range x {
				if text, ok := item.(string); ok {
					for token, path := range bindings {
						text = strings.ReplaceAll(text, token, path)
					}
					if strings.Contains(text, "<portable-file:") {
						return contracts.Fail("unavailable")
					}
					x[i] = text
				} else if e := bind(item); e != nil {
					return e
				}
			}
		}
		return nil
	}
	if e := bind(doc); e != nil {
		return nil, e
	}
	raw, _ := json.Marshal(doc)
	return raw, nil
}

func (a *App) portableInvocation(w catalog.Work) (json.RawMessage, error) {
	portable, _, e := a.Catalog.PortableWorkInput(a.ctx, w)
	if e != nil || !portable {
		return w.Payload, e
	}
	raw, e := a.bindPortableConfiguration(w.Payload)
	if e != nil {
		return nil, e
	}
	if w.Kind == "recordings.process" || w.Kind == "models.ensure" {
		var p recordingPayload
		if strictPayload(raw, &p) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if w.Kind == "recordings.process" && p.Options.Transcription == "generate" {
			budget, supported := 200, true
			stage := pipeline.Stage{Recognition: p.Options.Recognition}
			if p.Election != nil {
				stage = p.Election.Definition.Recognition
				if p.Election.PipelineID != "" {
					cap, e := pipeline.Capabilities(stage)
					if e != nil {
						return nil, e
					}
					budget, supported = cap.MaxHintBytes, cap.SupportsHints
				}
			}
			compiled, e := a.recognitionContext(p.Options.Context, stage, budget, supported)
			if e != nil {
				return nil, e
			}
			p.Options.Recognition.Hints = compiled.Hints
			p.Options.Recognition.ContextDigest = processing.HintsDigest(compiled.Hints)
			if p.Election != nil {
				p.Election.Context = compiled
				p.Election.Definition.Recognition.Recognition = p.Options.Recognition
			}
		}
		raw, _ = json.Marshal(p)
	} else if w.Kind == "recordings.match" {
		var p voicemodels.MatchPayload
		if strictPayload(raw, &p) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if p.Snapshot.Recording.ID == "" && p.Snapshot.Digest != "" {
			current, e := a.Catalog.FreezeSpeakerMatching(a.ctx, p.Options.RecordingID)
			if e != nil {
				return nil, e
			}
			if current.Digest != p.Snapshot.Digest {
				return nil, contracts.Fail("conflict")
			}
			p.Snapshot = current
		}
		raw, _ = json.Marshal(p)
	}
	return raw, nil
}
