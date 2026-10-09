// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFetchCommitDoesNotReplaceRacingEmptyDestination(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"stage", "destination"} {
		if err := os.Mkdir(filepath.Join(directory, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err = renameFetched(root, "stage", "destination"); err == nil {
		t.Fatal("replaced an existing destination directory")
	}
	if _, err = os.Stat(filepath.Join(directory, "stage")); err != nil {
		t.Fatal("failed commit lost staging directory", err)
	}
	if err = os.Remove(filepath.Join(directory, "destination")); err != nil {
		t.Fatal(err)
	}
	if err = renameFetched(root, "stage", "destination"); err != nil {
		t.Fatal("new destination commit", err)
	}
}
