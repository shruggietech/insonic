// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

type modifyingWriter struct {
	bytes.Buffer
	source  *os.File
	changed bool
}

func (w *modifyingWriter) Write(p []byte) (int, error) {
	if !w.changed {
		w.changed = true
		if _, e := w.source.WriteAt([]byte("changed"), 0); e != nil {
			return 0, e
		}
	}
	return w.Buffer.Write(p)
}
func TestStableCopyRejectsChangedOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source")
	if e := os.WriteFile(path, bytes.Repeat([]byte("original"), 1024), 0600); e != nil {
		t.Fatal(e)
	}
	f, e := os.OpenFile(path, os.O_RDWR, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	_, _, e = copyStable(context.Background(), path, f, &modifyingWriter{source: f})
	if e == nil {
		t.Fatal("changed source admitted")
	}
}
func TestStableCopyBindsExactBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source")
	if e := os.WriteFile(path, []byte("original"), 0600); e != nil {
		t.Fatal(e)
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	var out bytes.Buffer
	digest, size, e := copyStable(context.Background(), path, f, &out)
	if e != nil || size != 8 || out.String() != "original" || len(digest) != 64 {
		t.Fatalf("copy %q %d %v", digest, size, e)
	}
}
