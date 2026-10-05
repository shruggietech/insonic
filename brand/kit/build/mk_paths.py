"""Insonic A1: three separated, rounded voice forms on a 1000-unit grid."""

import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / 'templates'))
import glyphkit as G

GRID = 1000  # Square construction canvas.
FULL_STEM = 125  # Width and endcap diameter of the selected Full voice forms.
REDUCED_STEM = 160  # Strengthened width and endcap diameter for tiny applications.
BARS = ((282.5, 440, 220), (512.5, 265, 470), (742.5, 125, 750))  # Center x, top y, height.
CLEAR_SPACE = 100  # Minimum space around the visible artwork in grid units.
REDUCED_BELOW_PX = 32  # Reduced master takes over at this size and below.

# Keep the selected Full geometry exactly. The owner requested the same three
# forms with strengthened stems for the Reduced master on 2026-10-03.
full = G.center_ink([
    {'role': 'accent',
     'd': G.rounded_rect(cx - FULL_STEM / 2, y, FULL_STEM, height, FULL_STEM / 2)}
    for cx, y, height in BARS
], GRID)
reduced = G.center_ink([
    {'role': 'accent',
     'd': G.rounded_rect(cx - REDUCED_STEM / 2, y, REDUCED_STEM, height, REDUCED_STEM / 2)}
    for cx, y, height in BARS
], GRID)

if __name__ == '__main__':
    print(json.dumps({'grid': GRID, 'full': full, 'reduced': reduced}, indent=2))
