package conformance

import "testing"

func TestWindowChromeProfiles(t *testing.T) {
	cases := []struct { viewport Rect; reserved []Rect; controls []Rect }{
		{Rect{0, 0, 1280, 900}, []Rect{{1136, 0, 144, 48}}, []Rect{{16, 8, 600, 44}}},
		{Rect{0, 0, 768, 720}, []Rect{{624, 0, 144, 48}}, []Rect{{16, 8, 500, 44}}},
	}
	for _, test := range cases {
		if failures := Validate(test.viewport, test.reserved, test.controls); len(failures) != 0 { t.Fatalf("unexpected failures: %v", failures) }
	}
}

func TestWindowChromeOverlapFails(t *testing.T) {
	failures := Validate(Rect{0, 0, 768, 720}, []Rect{{624, 0, 144, 48}}, []Rect{{650, 8, 100, 44}})
	if len(failures) != 1 || failures[0] != "host.titlebar-overlap" { t.Fatalf("expected titlebar diagnostic, got %v", failures) }
}
