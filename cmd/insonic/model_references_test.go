// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestModelReferenceGrammar(t *testing.T) {
	id := contracts.ID()
	for _, tc := range []struct {
		args       []string
		op, target string
	}{
		{[]string{"models", "resolve", "speech-main", "--operation", "transcription"}, "models.resolve", ""},
		{[]string{"models", "resolve", "source:custom/model:v1", "--operation", "diarization"}, "models.resolve", ""},
		{[]string{"models", "resolve", "speaker:" + id, "--operation", "speaker-model-training"}, "models.resolve", ""},
		{[]string{"models", "discover", id}, "models.discover", ""},
		{[]string{"models", "alias", "list"}, "models.alias.list", ""},
		{[]string{"models", "source", "show", id}, "models.source.show", id},
		{[]string{"models", "alias", "remove", id, "--expected-revision", "12"}, "models.alias.remove", id},
	} {
		op, target, raw, err := parseDomain(tc.args)
		if err != nil || op != tc.op || target != tc.target {
			t.Fatalf("%v: %s %s %v", tc.args, op, target, err)
		}
		if len(raw) > 0 && !json.Valid(raw) {
			t.Fatal("invalid payload")
		}
	}
	for _, args := range [][]string{
		{"models", "resolve", "speech-main"},
		{"models", "resolve", "speech-main", "--operation", "guess"},
		{"models", "discover", "friendly-name"},
		{"models", "alias", "show", id, "extra"},
		{"models", "alias", "remove", id, "--expected-revision", "0"},
		{"models", "alias", "remove", id, "--expected-revision", "-1"},
		{"models", "source", "list", "--unexpected", "x"},
	} {
		if _, _, _, err := parseDomain(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestModelReferenceBoundedMutationInput(t *testing.T) {
	id := contracts.ID()
	path := filepath.Join(t.TempDir(), "alias.json")
	raw := []byte(`{"expected_revision":0,"alias":{"id":"` + id + `","name":"speech-main","state":"active","target":{"kind":"base","id":"` + id + `","operation":"transcription"}}}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	op, target, data, err := parseDomain([]string{"models", "alias", "set", id, "--input", path})
	if err != nil || op != "models.alias.set" || target != id || string(data) != string(raw) {
		t.Fatalf("%s %s %s %v", op, target, data, err)
	}
	if err := os.WriteFile(path, []byte(`{"expected_revision":0,"expected_revision":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := parseDomain([]string{"models", "alias", "set", id, "--input", path}); err == nil {
		t.Fatal("duplicate fields")
	}
}
