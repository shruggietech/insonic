# Implementation Contract: insonic

Generated from `enforcement/documentation-facts.json` under documentation contract `1.2.0`. This file describes the exact delivered kit and remains authoritative offline.

## Exact versions

| Domain | Version |
| --- | --- |
| Brand Canon | `2.0.0` |
| Interface Canon | `1.0.1` |
| Component recipes | `1.1.0` |
| Web/React adapter | `1.1.0` |
| egui adapter | `1.0.2` |
| WordPress adapter | `1.0.0` |
| BrandBuilder | `3.0.1` |
| Brand | `1.0.0` |

Package identity: `insonic-brand-1.0.0-bb3.0.1` (`insonic-brand-1.0.0-bb3.0.1.zip`)

## Bindings

- **Interface Canon:** `enforcement/interface-canon.json`
- **Component Recipes:** `enforcement/component-recipes.json`
- **Web Adapter:** `web/adapter.json`
- **Web Support Matrix:** `web/support-matrix.json`
- **Egui Adapter:** `native/egui/adapter.json`
- **Egui Support Matrix:** `native/egui/support-matrix.json`
- **Wordpress Adapter:** `wordpress/adapter.json`
- **Wordpress Support Matrix:** `wordpress/support-matrix.json`
- **Wordpress Theme Zip:** `wordpress/insonic-stbb-theme.zip`

## Interface starting point and overrides

This brand uses its own delivered interface binding; do not assume a ShruggieTech house palette.

Interface overrides:
No brand-specific interface overrides are declared. Use the delivered binding and its default rules.

Follow authority in this order: `brand.json` -> `enforcement/bundle.json` -> `enforcement/release-impact.json` -> `enforcement/interface-canon.json` -> `enforcement/component-recipes.json` -> `enforcement/version-policy.json` -> `enforcement/documentation-contract.json` -> `enforcement/consumer-contract.json` -> `human instructions that do not conflict`.

## Verification

- `python3 enforcement/brandbuilder/templates/verify.py .`
- `python3 enforcement/brandbuilder/templates/validate_glyph.py brand.json`

Success means zero verifier problems and zero glyph failures.

## Offline recovery

Verify `bef69601c58d6a6575aa5c45c8908069c1fb3b99320c2c6bf6548d303ae7d682` for `enforcement/distributions/shruggie-brandbuilder-3.0.1.skill`, then extract it to `enforcement/brandbuilder`. Use the delivered bundle and never substitute an unspecified latest release.

Capability gaps stay local at `enforcement/capability-gap.example.json` until a human explicitly authorizes upstream submission.

## Migration impact

No approved identity redesign is included. Brand version `1.0.0` remains distinct from package `insonic-brand-1.0.0-bb3.0.1`.

| Surface | Classification | Guidance |
| --- | --- | --- |
| Identity | unaffected | Existing approved masters and proof bytes remain unchanged. insonic 1.0.0 is a new, separately approved three-form Glacial blue identity. |
| Palette | unaffected | Existing palette values and WCAG 2.1 AA rules remain unchanged. |
| Typography | unaffected | Approved local font families, faces, and licenses remain unchanged. |
| Platform Assets | optional | Regenerate only to adopt corrected type specimen header alignment, explicit LF serialization on Windows and isolated QC captures. Approved logos, icons and social compositions remain source-bound. |
| Web React | unaffected | Web and React adapter APIs and interface token behavior remain unchanged. |
| Egui | unaffected | The egui adapter API and output semantics remain unchanged. |
| Wordpress | unaffected | WordPress adapter 1.0.0 retains its public contract and uses the compatible Brand Canon 2.0.0 source tuple. |
| Documentation | required | Publish insonic guidelines, approved messages, registries and downloads through the standard generated routes. Existing approved message-role uses remain unchanged; the manual links the exact release. |
| Recovery | required | Repinning BrandBuilder 3.0.1 requires its exact new package IDs and checksums. The verified site hosts twelve brands; nine owned archives join the skill and portable distribution, while three client kits remain hosted candidates. Previously published packages remain immutable recovery inputs. |

`required` applies to existing use of that surface, `optional` is an available capability, and `unaffected` requires no migration.

For shared architecture and extension guidance, read [/docs/](https://brand.shruggie.tech/docs/). The hosted reference describes only the current generated kit. This bundled contract continues to govern these pinned delivered bytes.

## Brand-specific governed rules

# Agent Contract: insonic

**Read this before writing any UI. It takes a minute and it is binding.**

You are working inside a brand with a fixed vocabulary. If you need a value
that is not in this document, **stop and ask**. Do not invent one, and do not
reach for a stock Tailwind palette class because it is faster.

## The stop condition

Inventing a colour, a spacing value, a radius, a font, or a component prop is
the failure this contract exists to prevent. When the vocabulary below does not
cover what you need, say so and wait.

## Colour: use the slot, never the value

Write `bg-primary`, `text-muted-foreground`, `border-border`. Never write a
hex, an `rgb()`, or `bg-slate-900`.

| Slot | Dark | Light |
| --- | --- | --- |
| `background` | `#0C151B` | `#F8F8F6` |
| `foreground` | `#FFFFFF` | `#0A0A0A` |
| `card` | `#14222C` | `#FFFFFF` |
| `primary` | `#86BDF2` | `#256DA9` |
| `muted-foreground` | `#9A9A9A` | `#6B6B6B` |
| `destructive` | `#E9505F` | `#C0293A` |
| `border` / `input` | `#262626` | `#E5E5E5` |

### Three colour mistakes that get made constantly

1. **White text on the accent.** `#FFFFFF` on `#86BDF2` measures 1.99:1 and
   fails. The legal foreground is `#000000` at 10.57:1. Use
   `text-primary-foreground` and it is handled.
2. **The bright accent as text on a light surface.** `#86BDF2` measures 1.87:1
   on `#F8F8F6`. The light block already substitutes `#256DA9`. Never override it.
3. **`#256DA9` as text.** It measures 3.84:1 on the dark base. It is a fill.
   Its legal foreground is `#FFFFFF` at 5.47:1.

## Spacing and radius

Spacing scale, in px: 4/8/12/16/24/32/48/64/96/120. Nothing between them.

Radii: `rounded-sm` 6 (chips), `rounded-md` 8 (buttons, inputs, popovers),
`rounded-xl` 12 (cards, dialogs), `rounded-2xl` 16, `rounded-full` (badges).
Never `rounded-none`, never an arbitrary `rounded-[...]`.

Layout: content 1200px, narrow 720px. Gutters 24px then 48px then 80px. Section rhythm 120px then 160px then 200px.

## Type

Space Grotesk for display at 500/700. Geist for body at 400/500. Geist Mono for labels,
code, and metadata at 400.

Asking for an undeclared weight makes the renderer synthesise or substitute a face, which prints badly and forces outlined glyphs into PDFs. In mono, carry emphasis with colour.

## Affiliation

This is a ShruggieTech-owned child brand. The only approved ownership endorsement is `A ShruggieTech project`. Keep it outside the logo clear space.



## Density

Two settings ship, and both are correct in the right place. Default for
marketing and reading surfaces; compact for dense tabular data. Do not invent
a third.

## Components and AppFrame

Read `component-recipes.json` and `../web/adapter.json` before composing shared controls. Use `../tokens/interface.css` for semantic custom properties without React. React consumers import static components from `../web/react/server.tsx` and behavior-heavy controls from `../web/react/client.tsx` with exact `radix-ui@1.6.7`.

`AppFrame` is the application shell owner. It owns safe areas, dynamic viewport behavior, root scrolling, fixed chrome, IME obstruction, titlebar avoidance, and global focus unless the selected browser, Tauri, or Wails profile transfers that one responsibility to the host. Never apply the same inset in native and web layers. Product screens, navigation trees, raw style props, and open-ended element substitution are outside the recipe grammar.

## Icons

lucide, inline SVG, `currentColor`, 1.5 to 2px stroke on a 24 grid. Do not
install another icon library. If lucide lacks a domain symbol, it goes in
`icons/` drawn to the same spec.

## Accessibility, non-negotiable

- Visible 2px focus ring at 2px offset on every interactive element
- Status never carried by colour alone; pair it with a label or a shape
- Respect `prefers-reduced-motion`
- WCAG AA at rendered size

## Copy

insonic copy is calm, clear, precise and technically literate. Put voice archives and speaker identity first. Use familiar nouns and verbs. Keep
sentences short.

Do not reach for: unsubstantiated recognition accuracy; claims of implemented planned capabilities; invented testimonials.

Glacial blue carries the identity on midnight surfaces; deep glacial blue carries it on light surfaces. State colors remain functional, with labels and symbols accompanying them.


Never build a sentence out of `X, not Y`, or `X over Y`, or
`rather than merely Z`. It is the clearest tell of machine-written copy.
Avoid em-dashes; use parentheses, commas, or hyphens. No testimonials, no
feature grids standing in for an explanation, no manufactured urgency.

## Before you call it done

```bash
npx eslint --config enforcement/eslint.brand.mjs .
npx stylelint --config enforcement/stylelint.config.json "**/*.css"
python3 enforcement/brandbuilder/templates/verify.py .
python3 enforcement/brandbuilder/templates/validate_glyph.py brand.json
```

A build that fails any of these is not finished, whatever it looks like.
