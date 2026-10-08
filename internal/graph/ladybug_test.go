//go:build system_ladybug

package graph

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRealLadybugOrderedReferences(t *testing.T) {
	a, e := OpenLadybug(filepath.Join(t.TempDir(), "graph"))
	if e != nil {
		t.Fatal(e)
	}
	qualifyAdapter(t, a)
}
func TestRealLadybugNativeReadOnly(t *testing.T) {
	a, e := OpenLadybug(filepath.Join(t.TempDir(), "graph"))
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if e = a.EnsureSchema(context.Background()); e != nil {
		t.Fatal(e)
	}
	if _, e = a.Query(context.Background(), "ladybug-cypher", "MATCH (n:Entity) RETURN count(n) AS count", nil); e != nil {
		t.Fatal(e)
	}
	if _, e = a.Query(context.Background(), "ladybug-cypher", "CREATE (:Entity {id:'effect'})", nil); e == nil {
		t.Fatal("native write executed")
	}
}
