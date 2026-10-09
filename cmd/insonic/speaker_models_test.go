// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"testing"
)

func TestSpeakerModelCommandsRetainExactIdentity(t *testing.T) {
	id := contracts.ID()
	file := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(file, []byte(`{"speaker_id":"`+id+`"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args     []string
		op, item string
	}{
		{[]string{"models", "dataset", "create", "--input", file}, "models.dataset.create", ""},
		{[]string{"models", "dataset", "list"}, "models.dataset.list", ""},
		{[]string{"models", "dataset", "show", id}, "models.dataset.show", id},
		{[]string{"models", "train", "--input", file}, "models.train", ""},
		{[]string{"models", "speaker", "list", "--input", file}, "models.speaker.list", ""},
		{[]string{"models", "speaker", "show", id}, "models.speaker.show", id},
		{[]string{"models", "profile", "set", id, "--input", file}, "models.profile.set", id},
	} {
		op, item, _, err := parseDomain(tc.args)
		if err != nil || op != tc.op || item != tc.item {
			t.Fatalf("%v: %s %s %v", tc.args, op, item, err)
		}
	}
	op, item, raw, err := parseDomain([]string{"models", "speaker", "fetch", id, "--destination", "output models"})
	var request struct {
		Destination string `json:"destination"`
	}
	if err != nil || op != "models.speaker.fetch" || item != id || json.Unmarshal(raw, &request) != nil || !filepath.IsAbs(request.Destination) {
		t.Fatal(op, item, string(raw), err)
	}
	op, _, raw, err = parseDomain([]string{"models", "list", "--speaker", id, "--limit", "25"})
	var list struct {
		Speaker string `json:"speaker_id"`
		Limit   int    `json:"limit"`
	}
	if err != nil || op != "models.speaker.list" || json.Unmarshal(raw, &list) != nil || list.Speaker != id || list.Limit != 25 {
		t.Fatal(op, string(raw), err)
	}
}

func TestSpeakerModelCommandsRejectWrongVersionAndFlags(t *testing.T) {
	id := contracts.ID()
	for _, args := range [][]string{
		{"models", "speaker", "show", "latest"},
		{"models", "speaker", "show", id, "--input", "ignored"},
		{"models", "speaker", "fetch", id},
		{"models", "dataset", "create"},
		{"models", "train", id},
		{"models", "profile", "set", "name", "--input", "ignored"},
		{"models", "list", "--speaker", id, "--speaker", id},
		{"models", "list", "--speaker", id, "--limit", "101"},
	} {
		if _, _, _, err := parseDomain(args); err == nil {
			t.Fatal("invalid command admitted", args)
		}
	}
}
