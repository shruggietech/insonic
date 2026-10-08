// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

type growingPlaybackSource struct {
	path   string
	output bytes.Buffer
	grown  bool
}

func (w *growingPlaybackSource) Write(raw []byte) (int, error) {
	n, e := w.output.Write(raw)
	if !w.grown {
		w.grown = true
		file, err := os.OpenFile(w.path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return n, err
		}
		_, err = file.Write([]byte("growth"))
		file.Close()
		if err != nil {
			return n, err
		}
	}
	return n, e
}

func TestPlaybackSnapshotBoundRejectsGrowingReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.wav")
	if os.WriteFile(path, []byte("four"), 0600) != nil {
		t.Fatal("source")
	}
	input, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer input.Close()
	output := &growingPlaybackSource{path: path}
	if _, _, e = copyStableBound(context.Background(), path, input, output, 4); e == nil {
		t.Fatal("source growth passed snapshot bound")
	}
	if output.output.Len() > 5 {
		t.Fatal("more than bound plus detection byte copied", output.output.Len())
	}
}
