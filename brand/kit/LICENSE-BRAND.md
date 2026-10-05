# Brand assets are reserved

The code, templates and reference documentation in this repository are licensed
under the Apache License 2.0. See `LICENSE`.

Apache-2.0 section 6 already states that the licence grants no permission to use
the licensor's trade names, trademarks, service marks or product names. This
file makes the boundary explicit, because a brand system is unusual in shipping
its own marks as data.

## Not licensed for reuse

- The names **ShruggieTech**, **Shruggie**, **Covarity**, **Cueson**, **Fragcap**,
  **Glitchpad**, and **Go Schedule**, together with the names of every future
  product built with this tool.
- Every wordmark and logo, in any format: SVG masters, PNG exports, favicons,
  ICO files, and the outlined wordmark geometry.
- The **logo path data** committed in each kit's `brand.json` under
  `logo.paths`, and any `build/mk_paths.py` that generates it. Logo geometry is
  the mark expressed as numbers, and it is reserved on the same terms as the
  rendered artwork.
- The endorsement lockup "A ShruggieTech project".

You may reproduce these to refer to ShruggieTech and its products accurately,
for example in an article, a comparison, or documentation of an integration.
You may not use them as your own mark, as part of your own mark, or in a way
that suggests endorsement or affiliation.

## Licensed for reuse under Apache-2.0

Everything else, including:

- `skill/templates/`, in full. `glyphkit.py` and `validate_glyph.py` in particular are
  general-purpose and were written to be useful outside this brand system.
- `skill/references/`, including the variance contract, the canon schema, the glyph
  construction procedure and the portability tiers.
- The generators, the verification and QC scripts, and the site source.
- The token *structure* and the methodology. The specific colour values that
  constitute a ShruggieTech identity are brand assets under the section above;
  the machinery that derives, measures and enforces them is not.

## Fonts

Geist, Geist Mono and Space Grotesk are licensed under the SIL Open Font
License 1.1 and are redistributed unchanged, with their licence texts, in every
generated kit. Neither this file nor `LICENSE` alters those terms.

## Questions

If you want to use something in a way this file does not clearly permit, ask
before you ship it.
