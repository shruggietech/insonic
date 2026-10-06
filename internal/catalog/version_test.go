package catalog

import (
	"testing"
)

func TestBackendVersionConstraints(t *testing.T) {
	for _, v := range []struct {
		actual, required string
		match            bool
	}{{"18.6", ">=18,<19", true}, {"3.53.4", "3.53.4", true}, {"3.53.4", ">=3.54", false}, {"18.6", "=18.5", false}, {"18.6", "<18.6", false}} {
		got, e := versionMatches(v.actual, v.required)
		if e != nil || got != v.match {
			t.Fatalf("%s %s %v %v", v.actual, v.required, got, e)
		}
	}
	if _, e := versionMatches("18.6", "unknown"); e == nil {
		t.Fatal("invalid constraint")
	}
}
