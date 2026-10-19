// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestS039SummariesSurviveRichResultBounds(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id, request, workspace := contracts.ID(), contracts.ID(), contracts.ID()
	operatorContext = invocation{Path: []string{"recordings", "show"}, Args: []string{"recordings", "show", id}, Verbose: true}
	rich := make([]any, 200)
	for i := range rich {
		rich[i] = map[string]any{"text": strings.Repeat("rich transcript ", 60), "timing": map[string]any{"start": 1}}
	}
	var out, errOut bytes.Buffer
	outputTo(map[string]any{"recording": map[string]any{"id": id, "revision": int64(9007199254740993), "document_digest": strings.Repeat("d", 64), "document": map[string]any{"cues": rich}}}, nil, false, contracts.Response{RequestID: request, WorkspaceID: workspace}, &out, &errOut)
	text := out.String()
	for _, want := range []string{id, "9007199254740993", strings.Repeat("d", 64), request, workspace, "recordings show", "--json"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q", want)
		}
	}
	if len(text) > 65536 || strings.Count(text, "\n") > 130 {
		t.Fatal("display bounds", len(text))
	}
}

func TestS039AllTaskConclusionsAndCorrectionsPrecedeInventories(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"processing", "readiness"}, Args: []string{"processing", "readiness"}}
	environments := map[string]any{}
	for _, name := range []string{"recognition", "diarization"} {
		rows := make([]any, 80)
		for i := range rows {
			rows[i] = map[string]any{"path": strings.Repeat("environment", 40)}
		}
		environments[name] = rows
	}
	tasks := map[string]any{}
	for _, name := range []string{"import", "existing_subtitles", "transcription", "diarization", "training"} {
		tasks[name] = map[string]any{"state": "needs_attention", "usable": false, "actions": []any{"Correct " + name}, "prerequisites": []any{map[string]any{"name": "engine", "available": false, "correction": "Repair " + name}}}
	}
	var out, errOut bytes.Buffer
	outputTo(map[string]any{"tasks": tasks, "environments": environments}, nil, false, contracts.Response{}, &out, &errOut)
	for name := range tasks {
		for _, want := range []string{name, "Correct " + name, "Repair " + name} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("missing %q", want)
			}
		}
	}
}

func TestS039ExactModelIdentityIsQuoted(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"models", "show"}}
	name := " \nmodel\r\nname\t "
	var out, errOut bytes.Buffer
	outputTo(map[string]any{"name": name, "model_version": " ", "upstream_revision": "first\nsecond"}, nil, false, contracts.Response{}, &out, &errOut)
	for _, want := range []string{strconv.Quote(name), strconv.Quote(" "), strconv.Quote("first\nsecond")} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("identity missing %q", want)
		}
	}
}

func TestS039CaptureTextAndContinuationPreserveExactMachineBytes(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id := contracts.ID()
	operatorContext = invocation{Path: []string{"media", "capture"}, Args: []string{"media", "capture", id, "--input", "capture.json"}}
	for _, section := range []string{"metadata", "facts", "dates"} {
		raw := []byte("Readable café capture")
		for _, bounds := range [][2]int{{0, len(raw)}, {0, 13}, {13, len(raw)}} {
			page := raw[bounds[0]:bounds[1]]
			result := map[string]any{"media_id": id, "section": section, "revision": 7, "sha256": strings.Repeat("a", 64), "encoding": "base64", "data": base64.StdEncoding.EncodeToString(page), "offset": bounds[0], "next_offset": bounds[1], "complete": bounds[1] == len(raw)}
			var out, errOut bytes.Buffer
			outputTo(result, nil, false, contracts.Response{}, &out, &errOut)
			if !utf8.Valid(out.Bytes()) || strings.Contains(out.String(), result["data"].(string)) || !strings.Contains(out.String(), section) || !strings.Contains(out.String(), "capture") {
				t.Errorf("unreadable capture %s", out.String())
			}
			if bounds[0] == 0 && !strings.Contains(out.String(), "Readable") {
				t.Fatal("capture text missing")
			}
			if bounds[0] == 13 || bounds[1] == 13 {
				if !strings.Contains(out.String(), "UTF-8 boundary:") {
					t.Fatal("split UTF-8 not explained")
				}
			}
			if bounds[1] != len(raw) {
				for _, want := range []string{"media capture " + id, "\"section\":\"" + section + "\"", "\"revision\":7", "\"offset\":13", "\"sha256\":\"" + strings.Repeat("a", 64)} {
					if !strings.Contains(out.String(), want) {
						t.Errorf("missing continuation %q", want)
					}
				}
			}
			out.Reset()
			outputTo(result, nil, true, contracts.Response{}, &out, &errOut)
			var response struct{ Result map[string]any }
			if json.Unmarshal(out.Bytes(), &response) != nil || response.Result["data"] != result["data"] {
				t.Fatal("machine bytes changed", out.String())
			}
		}
	}
}

func TestS039LongExactModelIdentityAndPagingCursorAreNotTruncated(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"models", "show"}}
	name := strings.Repeat("😀", 128)
	var out, errOut bytes.Buffer
	outputTo(map[string]any{"name": name}, nil, false, contracts.Response{}, &out, &errOut)
	if strings.Count(out.String(), "😀") < 128 {
		t.Fatal("maximum Unicode model identity was truncated")
	}
	operatorContext = invocation{Path: []string{"search"}, Args: []string{"search", "fixture"}}
	cursor := strings.Repeat("opaque-cursor-", 60)
	out.Reset()
	outputTo(map[string]any{"total": 500, "omitted": 400, "next_cursor": cursor}, nil, false, contracts.Response{}, &out, &errOut)
	if strings.Count(out.String(), "opaque-cursor-") < 60 {
		t.Fatal("paging authority was truncated")
	}
}

func TestS039ArbitraryQueryDocumentAndIdentityColumnsRemainVisible(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"query", "native"}, Args: []string{"query", "native", "--dialect", "ladybug-cypher", "--file", "query.cypher"}}
	result := map[string]any{"rows": []any{map[string]any{"document": map[string]any{"public": "public nested value"}, "name": "public name", "version": "public version", "payload": "public payload", "owner": "public owner"}}}
	var out, errOut bytes.Buffer
	outputTo(result, nil, false, contracts.Response{}, &out, &errOut)
	for _, want := range []string{"public nested value", "public name", "public version", "public payload", "public owner"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("public query value omitted: %q", want)
		}
	}
	if strings.Contains(out.String(), "embedded transcript") {
		t.Fatal("query column treated as recording document")
	}
}

func TestS039MutationCompleteJSONUsesReadbackCommands(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id := contracts.ID()
	for _, tc := range []struct{ path, args, read []string }{
		{[]string{"query", "save"}, []string{"query", "save", id, "--input", "saved.json"}, []string{"query", "show", id}},
		{[]string{"query", "assistance-set"}, []string{"query", "assistance-set", "--input", "assistance.json"}, []string{"query", "assistance-show"}},
		{[]string{"views", "save"}, []string{"views", "save", id, "--input", "layout.json"}, []string{"views", "show", id}},
	} {
		operatorContext = invocation{Path: tc.path, Args: tc.args}
		var out, errOut bytes.Buffer
		outputTo(map[string]any{"id": id, "revision": 7}, nil, false, contracts.Response{}, &out, &errOut)
		lines := strings.Split(out.String(), "\n")
		found := false
		for _, line := range lines {
			if strings.HasPrefix(line, "Complete JSON:") {
				found = true
				if !strings.Contains(line, strings.Join(tc.read, " ")) || strings.Contains(line, "--input") || strings.Contains(line, strings.Join(tc.path, " ")) {
					t.Errorf("mutation replay presented as readback: %s", line)
				}
			}
		}
		if !found {
			t.Errorf("missing readback for %v", tc.path)
		}
		if _, err := parseInvocation(tc.read); err != nil {
			t.Fatalf("readback grammar rejected: %v", err)
		}
	}
}

func TestS039OrdinaryPagingAuthoritySurvivesLargeItems(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id, next := contracts.ID(), contracts.ID()
	rich := make([]any, 100)
	for i := range rich {
		rich[i] = map[string]any{"id": contracts.ID(), "detail": strings.Repeat("optional detail ", 80)}
	}
	tests := []struct {
		path, args []string
		key        string
		value      any
		fragments  []string
	}{
		{[]string{"media", "list"}, []string{"media", "list", "--limit", "2"}, "next_id", next, []string{next, "--after", "media list"}},
		{[]string{"models", "list"}, []string{"models", "list"}, "next_id", next, []string{next, "--after", "models list"}},
		{[]string{"work", "list"}, []string{"work", "list"}, "next_id", next, []string{next, "--after", "work list"}},
		{[]string{"terms", "list"}, []string{"terms", "list", "--input", "original.json"}, "next_id", next, []string{next, "after_id", "terms list --input"}},
		{[]string{"models", "speaker", "list"}, []string{"models", "speaker", "list", "--input", "speaker.json"}, "next_id", next, []string{next, "after_id", "models speaker list --input", "Preserve"}},
		{[]string{"jobs", "history"}, []string{"jobs", "history", id}, "next_generation", json.Number("9007199254740993"), []string{"9007199254740993", "jobs history " + id}},
		{[]string{"work", "results"}, []string{"work", "results", id}, "next_ordinal", 100, []string{"100", "--after-ordinal", "work results"}},
		{[]string{"recordings", "cues"}, []string{"recordings", "cues", id}, "next_ordinal", 100, []string{"100", "after_ordinal", "recordings cues " + id, "document_digest"}},
		{[]string{"speakers", "select"}, []string{"speakers", "select", id, "--input", "speaker.json"}, "next_cursor", next, []string{next, "cursor", "speakers select " + id}},
	}
	for _, tc := range tests {
		operatorContext = invocation{Path: tc.path, Args: tc.args}
		result := map[string]any{"items": rich, tc.key: tc.value, "revision": 7, "document_digest": strings.Repeat("d", 64)}
		var out, errOut bytes.Buffer
		outputTo(result, nil, false, contracts.Response{}, &out, &errOut)
		text := out.String()
		detail := strings.Index(text, "Item 1")
		for _, want := range tc.fragments {
			if at := strings.Index(text, want); at < 0 || at > detail {
				t.Errorf("paging authority %q hidden behind detail for %v", want, tc.path)
			}
		}
		if len(text) > 65536 || strings.Count(text, "\n") > 130 {
			t.Fatal("paging expanded output bounds")
		}
	}
}

func TestS039NestedManifestRevisionPrecedesMaterializationInventory(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	id := contracts.ID()
	operatorContext = invocation{Path: []string{"models", "materialize"}, Args: []string{"models", "materialize", id}}
	revision := strings.Repeat("漢", 511) + "終"
	files := make([]any, 64)
	for i := range files {
		files[i] = map[string]any{"role": "weights", "path": strings.Repeat("optional file path ", 30), "sha256": strings.Repeat("a", 64)}
	}
	manifest := map[string]any{"kind": "base-model-manifest", "schema_version": contracts.Version, "name": "fixture", "model_version": "1", "upstream_revision": revision, "files": files}
	var out, errOut bytes.Buffer
	outputTo(map[string]any{"model_id": id, "name": "fixture", "version": "1", "manifest": manifest, "files": files}, nil, false, contracts.Response{}, &out, &errOut)
	text := out.String()
	if strings.Count(text, "漢") < 511 || !strings.Contains(text, "終") || strings.Index(text, "終") > strings.Index(text, "optional file path") {
		t.Fatal("complete nested model revision lost behind file inventory")
	}
	if len(text) > 65536 || strings.Count(text, "\n") > 130 {
		t.Fatal("exact manifest identity expanded output bound")
	}
}
