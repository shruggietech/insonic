# Glitchpad fonts

Glitchpad uses Space Grotesk for display text, Geist for body copy, and Geist Mono for paths, offsets, code, metadata, and interface labels. WOFF2 files are provided for web use. Static TTF instances are provided for desktop and design tools.

## Available weights

| Family | Weights shipped |
| --- | --- |
| Space Grotesk | 500, 700 |
| Geist | 400, 500 |
| Geist Mono | 400 |

Asking CSS for a weight outside this table makes the browser synthesise it, which produces a faux bold that prints poorly and forces outlined glyphs into exported PDFs. Emphasis in body copy resolves to Geist Medium (500). In mono contexts, carry emphasis with colour.

## Glyph coverage

Geist and Geist Mono have no `U+25A0` BLACK SQUARE and no `U+25C6` BLACK DIAMOND. Draw geometric status shapes in CSS so the renderer never substitutes a system serif.

Geist and Space Grotesk are licensed under the SIL Open Font License 1.1. The
complete license texts are included in `licenses/`.
