//go:build system_ladybug

// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"github.com/shruggietech/insonic/internal/graph"
	"path/filepath"
	"testing"
)

func TestRestoredLadybugAcceptedEvidence(t *testing.T) {
	g, e := graph.OpenLadybug(filepath.Join(t.TempDir(), "restore.lbug"))
	if e != nil {
		t.Fatal(e)
	}
	restoredGraphSuite(t, g)
}
