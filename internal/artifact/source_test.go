// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenedSourceDoesNotReopenSubstitutedPath(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if e := os.WriteFile(source, []byte("authorized"), 0600); e != nil {
		t.Fatal(e)
	}
	opened, e := openSource(source)
	if e != nil {
		t.Fatal(e)
	}
	defer opened.Close()
	if e = os.Rename(source, source+".original"); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(source, []byte("substituted"), 0600); e != nil {
		t.Fatal(e)
	}
	content, e := io.ReadAll(opened)
	if e != nil || string(content) != "authorized" {
		t.Fatal("substituted path replaced the opened input", e)
	}
}

func TestSourceSymlinkSwapCannotBeFollowed(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if e := os.WriteFile(source, []byte("authorized"), 0600); e != nil {
		t.Fatal(e)
	}
	var e error
	if e = os.Rename(source, source+".original"); e != nil {
		t.Fatal(e)
	}
	if e = os.Symlink(source+".original", source); e != nil {
		t.Skip("host does not permit test symlink creation", e)
	}
	// Even a link back to the originally authorized inode must be refused.
	if f, e := openSource(source); e == nil {
		f.Close()
		t.Fatal("source symlink followed")
	}
}
