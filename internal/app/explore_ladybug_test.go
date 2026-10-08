//go:build system_ladybug

// SPDX-License-Identifier: Apache-2.0
package app

import (
	"github.com/shruggietech/insonic/internal/graph"
	"path/filepath"
	"testing"
)

func TestExploreRealLadybugCurrentQueries(t *testing.T) {
	a := configuredApp(t)
	g, e := graph.OpenLadybug(filepath.Join(t.TempDir(), "graph"))
	if e != nil {
		t.Fatal(e)
	}
	exploreBackendSuite(t, a, g)
}
