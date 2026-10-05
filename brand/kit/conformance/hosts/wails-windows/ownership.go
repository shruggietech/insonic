package conformance

type Rect struct { X, Y, Width, Height float64 }

func overlaps(left, right Rect) bool {
	return !(left.X+left.Width <= right.X || right.X+right.Width <= left.X || left.Y+left.Height <= right.Y || right.Y+right.Height <= left.Y)
}

func Validate(viewport Rect, reserved []Rect, controls []Rect) []string {
	problems := []string{}
	for _, control := range controls {
		if control.X < viewport.X || control.Y < viewport.Y || control.X+control.Width > viewport.X+viewport.Width || control.Y+control.Height > viewport.Y+viewport.Height {
			problems = append(problems, "host.control-obstructed")
		}
		for _, region := range reserved {
			if overlaps(control, region) { problems = append(problems, "host.titlebar-overlap") }
		}
	}
	return problems
}
