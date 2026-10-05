# Brand Rules Reference (Light-Surface Slice)

Compressed reference for the values and rules the skill enforces. Authoritative source is the ShruggieTech parent-brand identity; this file restates the light-surface slice the docx skill needs so it stays self-contained. Documents render on white paper, so dark-mode tokens are omitted by design.

## Five Immutable Brand Color Tokens

These five hex values do not change at the token level under any circumstance.

| Token | Hex | Role in documents |
|---|---|---|
| Black | `#000000` | Available; not used for body text on white (see accessibility rule) |
| Green Bright | `#2BCC73` | NOT used on white paper (fails WCAG AA on white at 2.4:1) |
| Green Deep | `#00AB21` | TITLE color, hyperlink color, eyebrow label color, accent rules |
| Orange | `#FF5300` | Reserved for emphasis (callouts, alerts); not used in default contract templates |
| Gray Light | `#D1D3D4` | Horizontal rules, secondary borders |

Derived tokens (extended gray palette, table borders) may deviate from these five. Brand tokens may not.

## Light-Surface Accessibility Rule

Green Bright (`#2BCC73`) hits roughly 2.4:1 contrast on white and fails WCAG AA for body text. Do not use Green Bright on any white surface in any document the skill produces. Every accent that would be Green Bright on a dark surface substitutes to Green Deep (`#00AB21`, roughly 3.6:1, acceptable for large text and decorative use) on a light surface. Body text uses near-black (`#0A0A0A`, roughly 19.5:1).

Orange is used directly without a darkened variant when emphasis is explicitly requested.

## Extended Gray Palette (Derived)

These ladder under the five immutable tokens and may be used for hierarchical heading colors and rule lines.

| Token | Hex | Suggested use |
|---|---|---|
| Gray 950 | `#0A0A0A` | Body text, H1, H2 |
| Gray 800 | `#1A1A1A` | H3 |
| Gray 600 | `#6B6B6B` | H4, H5, H6, SUBTITLE, footer text |
| Gray 400 | `#9A9A9A` | Captions, very low-emphasis annotations |
| Gray 200 | `#E5E5E5` | Subtle borders, table cell borders |
| Gray 100 | `#F5F5F5` | Cell shading for alternating rows, code-block backgrounds |

## Typography

Three families, all loaded from bundled local copies under `assets/fonts/` and embedded into every emitted `.docx` via the post-generation step. The production code path never makes a CDN request.

| Role | Family | Weights | Used for |
|---|---|---|---|
| Display, headings, logotype | Space Grotesk | 500 (Medium), 700 (Bold) | TITLE, H1, H2, H3, H4, H5, H6 |
| Body, UI, paragraphs, lists | Geist | 400 (Regular), 500 (Medium for inline bold) | Body paragraphs, list items, SUBTITLE, footer |
| Monospace, eyebrow labels, code | Geist Mono | 400 (Regular) | Code blocks, inline code, eyebrow labels above headings |

Italic policy: only base `Geist-Italic.ttf` ships. Bold-italic, extra-bold-italic, light-italic, medium-italic, regular-italic, semi-bold-italic, and thin-italic variants of Geist are deliberately omitted. Use italic sparingly in prose only. Do not attempt to render bold-italic combinations.

Weight selection rule:

- Space Grotesk 700 for TITLE, H1, H2
- Space Grotesk 500 for H3, H4, H5, H6
- Geist 400 for body
- Geist 500 for inline bold within body text

Deviate only when the operator explicitly requests a heavier or lighter feel for a specific document.

System fonts (Arial, Calibri, Times New Roman, Inter) are graceful-degradation fallbacks only and appear only when font embedding has been disabled (it cannot be in this skill) and the recipient lacks the canonical fonts. Never specify Arial or Calibri as a primary face in any style.

## Bundled Font Files (the only six)

These six files ship in `assets/fonts/` and are the complete embed list. Refresh them when the CDN updates.

| File | Source URL |
|---|---|
| `SpaceGrotesk-Medium.ttf` | `https://cdn.shruggie.tech/brand/fonts/ttf/SpaceGrotesk-Medium.ttf` |
| `SpaceGrotesk-Bold.ttf` | `https://cdn.shruggie.tech/brand/fonts/ttf/SpaceGrotesk-Bold.ttf` |
| `Geist-Regular.ttf` | `https://cdn.shruggie.tech/brand/fonts/ttf/Geist-Regular.ttf` |
| `Geist-Medium.ttf` | `https://cdn.shruggie.tech/brand/fonts/ttf/Geist-Medium.ttf` |
| `Geist-Italic.ttf` | `https://cdn.shruggie.tech/brand/fonts/ttf/Geist-Italic.ttf` |
| `GeistMono-Regular.ttf` | `https://cdn.shruggie.tech/brand/fonts/ttf/GeistMono-Regular.ttf` |

Do not embed Geist Pixel. Do not embed Geist weights other than 400, 500, and base italic. Do not embed Space Grotesk weights other than 500 and 700. Embedding the full CDN catalog inflates every generated `.docx` by roughly 1.5 MB with no rendering benefit.

## Tagline

The exact tagline string is:

```
¯\_(ツ)_/¯ We'll figure it out.
```

The macron characters (`¯`, U+00AF) and katakana `ツ` (U+30C4) must be preserved byte-for-byte in any document that carries the tagline. Do not paraphrase. Use only in marketing-adjacent documents (Internal Report covers, capabilities one-pagers). Do not place in legal documents (SOW, SOA, MSA), where tone calls for a more formal close. See `doc-type-defaults.json` for which document types include the tagline by default.

## Logo (CDN, source of truth)

The bundled logo at `assets/brand/logo/logo-lightbg.png` is the byte-identical mirror of:

| Variant | URL | Use when |
|---|---|---|
| Light background | `https://cdn.shruggie.tech/brand/logo/logo-lightbg.png` | All `.docx` output (white paper) |

Production code never embeds the CDN URL; the generated `.docx` always inlines the bundled bytes. The CDN URL exists so the bundled asset stays traceable to its canonical origin.

Logo placement on every document: first paragraph, inline anchor, width 180 PT (2.5 inches), height computed at build time from the bundled PNG's native aspect ratio rounded to the nearest 0.5 PT. The bundled file currently has aspect ratio 4.5344:1, which yields 39.5 PT height at 180 PT width; this is computed fresh on every build so a future logo refresh does not require a skill update. Margins are 9 PT on all four sides. Alt text is `ShruggieTech`.

## Voice Rules

These apply to all copy the skill emits, not just headlines.

1. **Second person.** Speak to the reader. The reader is the subject, not the company.
2. **No competitive jabs.** Do not name competitors. Vendor-displacement narratives are fine as positioning; named attacks are not.
3. **No unsubstantiated claims.** A capability appears in copy only if it can be backed by concrete work.
4. **Confidence without arrogance.** State plainly. Avoid `we might be able to help` and avoid `industry-leading experts`.
5. **Plain English over consulting-speak.** No `synergy`, `leverage`, `best-in-class`, `turnkey`, `solutioning`.
6. **Humor is permitted in small doses.** The shruggie identity invites a wry self-aware tone; do not force it.

## Brand Architecture Scope

This skill covers the **ShruggieTech parent brand only** (and `Shruggie LLC` legal-entity attribution). Product sub-brands carry their own visual identities and are out of scope:

- metadexer
- shruggie-indexer
- shruggie-feedtools
- Covarity
- SparkPlan
- Knox.Dance
- rustif

If the operator asks for a document that is primarily about a sub-brand, flag the scope conflict and ask whether to (a) produce a parent-brand document that mentions the sub-brand, or (b) defer to the sub-brand's own identity system. Do not invent visual identity for a sub-brand inside this skill.

## Attribution

Standard formats:

- `By ShruggieTech` (default footer attribution)
- `A ShruggieTech product` (marketing copy in capabilities documents)
- `Shruggie LLC` (legal-entity attribution; appears in license headers, copyright notices, contract parties, invoice sender block)

The legal name `Shruggie LLC` appears in any document that carries legal weight (SOW, SOA, MSA, Invoice, signed Letter). The DBA `ShruggieTech` appears in marketing prose, capability descriptions, and operational copy.

## Output File Rules

Every file the skill writes (including the `.docx` itself, `SKILL.md`, `style-spec.json`, `document-template.js`, this file):

- UTF-8 encoding, no BOM
- LF line endings (not CRLF) for any text file in the skill repo
- No trailing whitespace on any line in text files
- Single trailing newline at end of every text file
- No em-dashes and no en-dashes anywhere in the document, including comments, alt text, footer template strings, and prose in SKILL.md. Use parentheses, commas, or standard hyphens instead.
- Every image has meaningful alt text. Decorative-only images are not used in document templates.
