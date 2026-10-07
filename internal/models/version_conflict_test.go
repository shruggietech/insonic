// SPDX-License-Identifier: Apache-2.0
package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelLogicalVersionRejectsChangedManifest(t *testing.T) {
	s, db := modelServiceFixture(t)
	body := []byte("model fixture")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL, body)}
	id := completedModel(t, s, modelWork(t, db, "models.acquire", m))
	changed := m
	changed.Revision = "changed-upstream"
	for _, kind := range []string{"models.register", "models.acquire"} {
		if _, e := s.Execute(context.Background(), modelWork(t, db, kind, changed)); e == nil {
			t.Fatal("changed manifest admitted", kind)
		}
	}
	all, e := db.BaseModels(context.Background())
	if e != nil || len(all) != 1 || all[0].ID != id || all[0].Digest != m.Digest() || all[0].State != "available" {
		t.Fatal("logical model version changed", all, e)
	}
	if got := completedModel(t, s, modelWork(t, db, "models.acquire", m)); got != id {
		t.Fatal("same manifest replay changed identity")
	}
	// Tuple encoding avoids collisions between names/versions containing ':' .
	first := fixtureManifest()
	first.Name, first.ModelVersion = "a:b", "c"
	second := fixtureManifest()
	second.Name, second.ModelVersion = "a", "b:c"
	id1 := completedModel(t, s, modelWork(t, db, "models.register", first))
	id2 := completedModel(t, s, modelWork(t, db, "models.register", second))
	if id1 == id2 {
		t.Fatal("logical version tuple collision")
	}
}
