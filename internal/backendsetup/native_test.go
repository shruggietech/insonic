//go:build system_ladybug

// SPDX-License-Identifier: Apache-2.0
//
package backendsetup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeGraphInitializedBeforeConfigPublication(t *testing.T) {
	c, _ := Build([]byte("{\"display_name\":\"Native graph bootstrap\"}"))
	root := t.TempDir()
	w, e := Provision(context.Background(), root, c, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(w.Root, "graph", "library.lbug")); e != nil {
		t.Fatal("missing selected native graph", e)
	}
}
