---
name: shruggie-docs
description: Build a single self-contained .docx file using the official ShruggieTech parent-brand identity (Space Grotesk and Geist typography embedded, justified body text, brand color tokens, lightbg logo). Use whenever the user asks for a Word document, contract, statement of work, scope of work agreement, master subcontract agreement, internal report, invoice, or letter that should read as a ShruggieTech artifact. Trigger on phrasings like "make me a docx", "draft a Statement of Work", "produce a Master Subcontract", "send an invoice for", "write a letter to", or any .docx request touching a ShruggieTech, Shruggie LLC, or parent-brand surface. Collaborates with the operator on document type, TOC, signatories, and legal fields; default to asking rather than guessing. Skip for Google Docs API output, PDF generation, slide decks, spreadsheets, and React or Next.js component work.
disable-model-invocation: false
---

# Shruggie Docs

Produce a single self-contained `.docx` file that conforms to the ShruggieTech parent-brand identity. The skill bundles the brand color tokens, typography (six TTFs), the lightbg logo bytes, the layout invariants, and the style spec so Claude can emit on-brand Word documents without re-deriving any of them per task. Every generated `.docx` has the six brand TTFs embedded into the archive so recipients without Geist or Space Grotesk installed still render the document with the correct typography. The skill governs six document types: Statement of Work (SOW), Scope of Work Agreement (SOA), Master Subcontract Agreement (MSA), Internal Report, Invoice, and Letter.

## When to Use

Invoke this skill when:

- The user asks for a Word document, `.docx`, or any Word-format deliverable in a ShruggieTech, Shruggie LLC, or parent-brand context.
- The user names a Statement of Work, Scope of Work Agreement, Master Subcontract Agreement, Internal Report, Invoice, or Letter and wants it as a Word file.
- The user iterates on the content of an existing on-brand `.docx` and wants the next revision built the same way.

Do not invoke this skill for:

- Google Docs API output (a separate skill is planned).
- PDF generation (use the appropriate PDF skill or convert from `.docx` downstream).
- Slide decks (`.pptx`) and spreadsheets (`.xlsx`).
- React, Next.js, Svelte, Vue, or other component-framework work.
- Pages that are primarily about a product sub-brand (metadexer, Covarity, Knox.Dance, SparkPlan, rustif, shruggie-indexer, shruggie-feedtools). Each sub-brand maintains its own visual identity. If a parent-brand document mentions a sub-brand in passing that is fine; if the document is mostly about the sub-brand, flag the scope conflict and ask whether to defer to the sub-brand's own system.
- Reformatting the existing SOW/SOA/MSA contract corpus retroactively. The skill governs new documents only.

## Relationship to the public docx skill

shruggie-docs is the canonical `.docx` producer for any ShruggieTech-branded document. It uses docx-js (the same npm `docx` package as the public docx skill at `/mnt/skills/public/docx/`), so the two skills are toolchain-compatible. Their style guidance is not compatible:

- shruggie-docs sets typography, color, page size, margins, and footer per the ShruggieTech brand. Public docx defaults (Arial 12pt, 1-inch margins, black titles) must NOT be applied to shruggie-docs output.
- Use the public docx skill for exactly two things: (a) the final `python /mnt/skills/public/docx/scripts/office/validate.py` pass after `embed-fonts.py`, and (b) editing existing `.docx` files via unpack-XML-repack when shruggie-docs is not regenerating from scratch.
- When the document carries a ShruggieTech surface (logo, brand colors, Shruggie LLC attribution, SOW/SOA/MSA/Invoice/Letter context), use shruggie-docs and ignore public docx style patterns.

### PDF emission and e-signature workflows

When a generated `.docx` is destined for an e-signature tool (DocuSign, Adobe Sign, PandaDoc, etc.) or any consumer that converts `.docx` to PDF server-side, upload the auto-emitted PDF instead of the `.docx`. E-sign converters use headless engines that do not honor Word's embedded fonts, so a pre-converted PDF preserves brand typography. The `.docx` remains the editable source-of-truth; the PDF is the distribution-ready artifact.

## Instructions

The skill's deliverable is always a single `.docx` file with the six brand TTFs embedded inside the archive. The production code path reads only from local bundled assets (`assets/brand/`, `assets/fonts/`); the CDN URLs in `assets/brand-rules.md` document the canonical origin of those bundled bytes and never appear in the build path.

### Toolchain

Generation runs on top of `docx-js` (the npm `docx` package). The skill bundles a docx-js scaffold at `assets/document-template.js` and a font-embedding post-process at `assets/embed-fonts.py`. The public `docx` skill at `/mnt/skills/public/docx/` (when present in the execution environment) provides `scripts/office/validate.py` for the final OOXML pass; if the public skill is not available, the embed script's built-in `--verify-only` check is the minimum acceptable verification.

Output encoding is UTF-8 without BOM for any text artifact the skill writes. The `.docx` itself is a ZIP archive; the UTF-8 rule applies to the skill source files in `assets/`.

### Collaboration rule (binding)

Document type selection and the most-overridden parameters are operator-collaborative by default. Ask the human operator before committing to any of these:

- Which of the six document types the request maps to, when the request is ambiguous. For example, "draft a contract" maps equally to SOW, SOA, or MSA.
- Whether to include a table of contents, even when the default for the chosen type would auto-include one.
- Signatory names, titles, and party identification on any document carrying legal weight (SOW, SOA, MSA, Letter, Invoice).
- Party metadata fields for SOW, SOA, and MSA. Always ask the operator for these and pass them as `partyMetadata`. The minimum set is Client, Partner, Date. If Date is empty, the scaffold defaults it to today's date in M/D/YYYY.
- Currency, payment terms, and tax handling for Invoice.

Defaults in `assets/doc-type-defaults.json` exist as fallbacks for unattended runs (batch generation, CI pipelines), not as the preferred path. A confused generation produced from defaults is a worse outcome than a clarifying question.

### Brand fidelity

The five immutable brand color tokens are `#000000` (Black), `#2BCC73` (Green Bright), `#00AB21` (Green Deep), `#FF5300` (Orange), and `#D1D3D4` (Gray Light). Their hex values do not change. The full token set, including the extended gray palette, is in `assets/brand-rules.md` and the machine-readable `assets/style-spec.json`.

Light-surface accessibility rule (binding): Green Bright fails WCAG AA on white paper (roughly 2.4:1). Green Bright must not appear anywhere in any document the skill produces. Every accent that would be Green Bright on a dark surface substitutes to Green Deep on the light surface that `.docx` output renders on.

### Typography

Three families, all loaded from `assets/fonts/` and embedded into every emitted `.docx`:

- Space Grotesk 500 (Medium) and 700 (Bold): TITLE, H1, H2 use 700; H3, H4, H5, H6 use 500.
- Geist 400 (Regular) and 500 (Medium): Body uses 400; inline bold within body uses 500.
- Geist Mono 400: code blocks, inline code, eyebrow labels.

Italic policy: only base `Geist-Italic.ttf` ships. Do not attempt bold-italic combinations. Use italic sparingly in prose only. The six TTFs in `assets/fonts/` are the complete embed list; do not embed Geist Pixel and do not embed Geist weights other than 400, 500, and base italic.

System fonts (Arial, Calibri, Times New Roman, Inter) are graceful-degradation fallbacks only. Never specify Arial or Calibri as a primary face in any style.

### Document layout (invariants)

- Page size: US Letter portrait, 612 PT x 792 PT (8.5 inch x 11 inch).
- Margins: 36 PT top, 54 PT bottom, 36 PT left, 36 PT right.
- One section, continuous; no header; one right-aligned footer.
- Logo first paragraph of every document: width 180 PT, height computed at build time from the bundled PNG's native aspect ratio rounded to the nearest 0.5 PT, 9 PT margins on all four sides, alt text `ShruggieTech`, centered horizontally. The logo is always loaded from `assets/brand/logo/logo-lightbg.png` on disk; never request it from the CDN.

#### Tables / Total rows

When a table relays a financial sum with the total in the bottom row, set `totalRow: true` on the node. The scaffold renders a 1 PT black top border on the bottom row's cells to signify the total.

#### Images

Embed a screenshot or figure with a first-class `image` content node rather than hand-building an `ImageRun` and passing it through `raw`:

```js
{ type: 'image', path: '/abs/path/to/shot.png', widthPT: 360, altText: 'Dashboard showing p95 latency by service', caption: 'Figure 1. Latency after the rollout.' }
```

The scaffold reads the PNG's native dimensions, computes the height from the aspect ratio, sets the correct media type (so no `.undefined` media part is emitted), centers the image, and emits an optional `Caption` paragraph when `caption` is supplied. Alt text is mandatory: an empty or missing `altText` throws at build time, per the output-hygiene rule that every image carry meaningful alt text. Only PNG is supported. `widthPT` defaults to 360 PT when omitted.

### Page break rule (binding)

For SOW, SOA, and MSA, the only H1 that carries `pageBreakBefore: true` is the Signatures section — always the last H1 in the content array. All other H1s flow naturally without a page break. The logo, TITLE, party metadata block, TOC heading, and section 1 all live on page 1. The signature block must occupy a single page; the page break before it guarantees that, and the signature block itself should not span pages.

Page breaks are caller-controlled: each H1 node in the content array carries its own `pageBreakBefore` boolean. Do not set `pageBreakBefore: true` on any H1 except Signatures.

### Justification rule (binding)

All paragraph body text uses JUSTIFY alignment. This applies to every paragraph rendered with the Body style. It does NOT apply to:

- Bullet list items (left-aligned).
- Numbered list items (left-aligned).
- Table cell text (left-aligned).
- Footer text (right-aligned per the footer spec).
- Headings (left-aligned per the style spec; TITLE and SUBTITLE are centered).
- Captions (centered).
- Eyebrow labels (left-aligned).
- Recipient blocks in letters (left-aligned).
- Sender and bill-to blocks in invoices (left-aligned).

The justified body is part of the ShruggieTech document layout standard. Verify in the pre-output checklist.

### Document-type behaviors

| Type | TOC default | Top block | Footer | Tagline in footer |
|---|---|---|---|---|
| SOW | no | TITLE + SUBTITLE | standard | no |
| SOA | yes | TITLE + SUBTITLE | standard | no |
| MSA | yes | TITLE only | standard | no |
| Internal Report | no | TITLE + SUBTITLE | standard | yes |
| Invoice | no | invoice header block | invoice (payment terms) | no |
| Letter | no | date + recipient + subject | minimal (omit on single page) | no |

The full table with `pageBreakRule` and other fields is in `assets/doc-type-defaults.json`. Invoice and Letter layouts have type-specific top blocks documented in detail in that file and in the handoff brief.

#### Signature blocks

Signature blocks use the label `Signature:` followed by an underscore line, then on separate lines `Name:`, `Title:`, and `Date:` rows for each signing party. Never use `By:`. Applies to SOW, SOA, MSA, and Letter.

### Voice rules

Every line of copy the skill emits follows these:

1. Second person. Speak to the reader. The reader is the subject, not the company.
2. No competitive jabs. Do not name competitors. Vendor-displacement narratives are fine as positioning; named attacks are not.
3. No unsubstantiated claims. A capability appears in copy only if it can be backed by concrete work.
4. Confidence without arrogance. State plainly. Avoid `we might be able to help` and avoid `industry-leading experts`.
5. Plain English over consulting-speak. No `synergy`, `leverage`, `best-in-class`, `turnkey`, `solutioning`.
6. Humor is permitted in small doses. The shruggie identity invites a wry self-aware tone; do not force it.

If the operator supplies draft copy that violates a rule, surface the conflict and ask before silently rewriting.

### Prose-first structure rule (binding)

Professional documents default to connected prose. Write full paragraphs with clear topic sentences and transitions. Do not split arguments or explanations into bullets when sentences carry the logic more clearly.

Use bulleted or numbered lists only when the content is genuinely list-shaped: a set of discrete parallel options, an ordered procedure, reference data, or a scan-oriented checklist. If there are three or fewer simple items that read naturally in one sentence, keep them inline as prose.

Do not use bolded lead-in labels followed by a colon as a substitute for sentence structure (for example, `Benefit: ...`). Do not nest bullet hierarchies inside otherwise narrative sections.

If you find yourself adding bullets on nearly every line, stop and convert the section into paragraphs. Headings and short tables remain valid structural tools.

### Citation and footnote rule (binding)

Use citations only when the document type and content warrant sourced factual claims. Typical citation-bearing outputs include research memos, technical reports, white papers, compliance artifacts, and legal documents that rely on external sources. Casual internal notes, informal updates, and documents with no sourced claims should not include citations.

When citations are required, render them as true page footnotes, not an end-of-document reference list:

- In body text, insert a real superscript footnote marker.
- Anchor the corresponding source note at the bottom of the same page via the document format's native footnote mechanism.
- Do not fake superscript with Unicode or manually typed characters.
- Do not collect citations into a single references section at the end of the document.

For docx-js specifically, use the `footnotes` configuration on `Document` plus `FootnoteReferenceRun` markers in body paragraphs. If a workflow is constrained to `python-docx` only, explicitly report that first-class footnotes are not supported and ask whether to switch to docx-js; do not silently substitute an end-of-document citation list.

### Tagline and attribution

The exact tagline string is `¯\_(ツ)_/¯ We'll figure it out.` Use only in marketing-adjacent documents (Internal Report covers, capabilities one-pagers). Do not paraphrase. Do not place in legal documents (SOW, SOA, MSA), where tone calls for a more formal close.

Attribution conventions:

- `By ShruggieTech` (default footer attribution).
- `A ShruggieTech product` (marketing copy in capabilities documents).
- `Shruggie LLC` (legal-entity attribution; appears in license headers, copyright notices, contract parties, invoice sender block).

The legal name `Shruggie LLC` appears in any document that carries legal weight. The DBA `ShruggieTech` appears in marketing prose, capability descriptions, and operational copy.

### Output hygiene

Every file the skill writes (the `.docx`, this `SKILL.md`, the JSON specs, the JS scaffold, the Python embed script):

- UTF-8 encoding, no BOM.
- LF line endings (not CRLF) for any text file in the skill repo.
- No trailing whitespace on any line in text files.
- A single trailing newline at end of every text file.
- No em-dashes and no en-dashes anywhere in the document, including comments, alt text, footer template strings, and prose. Use parentheses, commas, or standard hyphens instead.
- Every image has meaningful alt text. Decorative-only images are not used in document templates.

### Build procedure

When the operator has confirmed scope (document type, TOC inclusion, signatories, currency for invoices) and you are ready to emit the file:

1. Resolve the document type. Read `assets/doc-type-defaults.json` for the type's defaults (TOC, footer template, page-break rule, tagline inclusion). If anything is ambiguous, ask the operator per the collaboration rule.
2. Read `assets/style-spec.json` to materialize the named-styles map.
3. Construct a `Document` via docx-js using `assets/document-template.js` as the scaffold. Pass `assetsDir` to `buildDocument` so the template can locate the bundled logo and JSON specs.
4. The scaffold configures page size, margins, section, no header, and the right-aligned footer per the resolved type. The first paragraph is the lightbg logo at 180 PT width with build-time-computed height and 9 PT margins on all four sides.
5. Insert the type-appropriate top block: TITLE plus optional SUBTITLE for SOW/SOA/Internal Report, TITLE only for MSA, header block for Invoice, date plus recipient block for Letter.
6. Insert optional TOC (per resolved decision), then body content. The scaffold writes the TOC as static paragraphs (no SDT or TOC field), so recipients do not see a "Right-click to update Table of Contents" prompt. The scaffold applies the named-styles map; do not write per-paragraph overrides except for explicit operator-supplied formatting in the body content.
7. If the resolved document type and content warrant citations, configure native Word footnotes in the docx-js `Document` (`footnotes` map) and place each in-text citation marker with `FootnoteReferenceRun`. Citations must render as true superscript markers with source notes pinned to page bottoms. Never fall back to a document-end citation list for citation-bearing outputs.
8. Pack the `.docx` with `Packer.toBuffer(doc)` and write the buffer to disk. The `doc` value must be the `Document` returned by `buildDocument` - do not construct a `new Document(...)`, `new Footer(...)`, or footer paragraph by hand in the build script. The scaffold's `buildFooterParagraph` reads `footer.alignment` from `style-spec.json` (`"END"` = right-aligned in OOXML) and applies any per-type override from `doc-type-defaults.json` (Letter uses `"CENTER"`). Right alignment is required for SOW, SOA, MSA, Internal Report, and Invoice; constructing a footer outside the scaffold silently drops that alignment.
9. Run the font-embedding post-process: `python assets/embed-fonts.py <path-to-docx>`. This drops the six bundled TTFs into `word/fonts/`, patches `word/fontTable.xml` to reference each embed, writes the font relationships into `word/_rels/fontTable.xml.rels` (the part-relationships file that the `r:id` references in `fontTable.xml` resolve against), and sets the embed flag in `word/settings.xml`. This step is mandatory; there is no flag to skip it.
10. Validate the final file. Prefer `python /mnt/skills/public/docx/scripts/office/validate.py <path>` when the public docx skill is available in the execution environment. Otherwise, rely on the embed script's `--verify-only` check, which confirms the six TTFs are present in the archive and referenced in `fontTable.xml`.
11. Emit PDF. Check for libreoffice availability with `which libreoffice || which soffice`. If found, run `libreoffice --headless --convert-to pdf --outdir <output-dir> <docx-path>`, which writes `./<slug>.pdf` alongside the `.docx` and subsets and embeds Geist and Space Grotesk for PDF-only and e-signature workflows. If not found, skip this step and log exactly: `PDF emission skipped: libreoffice not installed. Install via 'brew install --cask libreoffice' on Mac or the equivalent on Linux to enable.` Do not fail the build.
12. Write to the path the operator specified, or to `./<slug>.docx` where the slug derives from the document title (lowercase, hyphenated).

### Pre-output checklist

Before writing the final file, verify:

- Page size is 612 PT x 792 PT, portrait.
- Margins are exactly 36 PT top, 54 PT bottom, 36 PT left, 36 PT right.
- The first paragraph contains the lightbg logo with width 180 PT, height computed from the bundled PNG's native aspect ratio, and 9 PT margins on all sides. The logo was loaded from `assets/brand/logo/logo-lightbg.png` on disk, not from the CDN.
- TITLE (for SOW, SOA, MSA, Internal Report) uses Space Grotesk 700, 24 PT, color `#00AB21`, centered. SUBTITLE is used for SOW, SOA, and Internal Report; MSA omits SUBTITLE. Invoice and Letter skip this check because they use document-type-specific top blocks.
- Every body paragraph uses JUSTIFY alignment. List items, table cells, headings, captions, eyebrow labels, footer text, recipient blocks (letters), and sender / bill-to blocks (invoices) are exempted per the justification rule.
- No Body paragraph uses Arial, Inter, Calibri, or Times New Roman. Body is Geist throughout.
- Headings are Space Grotesk (500 or 700 per the style spec).
- Footer matches the per-type footer rule (standard for contracts and reports, payment-terms for invoices, minimal or absent for letters) using OOXML `PAGE` and `NUMPAGES` field codes where page numbers are rendered.
- Green Bright (`#2BCC73`) does not appear anywhere in the document.
- If TOC is included, it renders before any H1 as static paragraphs (not a Word TOC field), and every entry is Geist 11 PT, left-aligned.
- If citations are required by the document type/content, every citation marker is a native footnote reference and every source appears as a page footnote (not in an end-of-document reference list).
- If citations are not required by the document type/content, no footnotes are present.
- The unpacked archive contains all six TTFs under `word/fonts/`, and `word/fontTable.xml` references each one with the appropriate `<w:embedRegular>`, `<w:embedBold>`, or `<w:embedItalic>` element.
- The unpacked archive contains no font files for Geist Pixel or for Geist weights other than 400, 500, and base italic.
- The validator (public docx skill or the embed script's `--verify-only`) returns a clean pass.
- If libreoffice is available on the system, a PDF was emitted at `<output-dir>/<slug>.pdf` with Geist and Space Grotesk subset and embedded.

## Examples

### Example: Statement of Work

**User input:**

```
Draft a SOW for the Acme observability engagement. Six-week phase, fixed price, two milestones, signatures at the bottom.
```

**Expected output:**

A single `.docx` file containing:

- The lightbg logo as the first paragraph, 180 PT wide, 9 PT margins.
- TITLE: `Statement of Work`, Space Grotesk 700, 24 PT, Green Deep, centered.
- SUBTITLE: `Acme Observability Engagement`, Geist 14 PT, Gray 600, centered.
- The party metadata block (between two horizontal rules): `Client: Acme Corp`, `Partner: Shruggie LLC`, `Date: <effective date>`.
- Numbered sections (`1. Engagement Summary`, `2. Scope`, `3. Deliverables`, `4. Milestones and Schedule`, `5. Fees and Payment`, `6. Acceptance Criteria`, `7. Signatures`) with H1 headings.
- Body paragraphs all justified, Geist 11 PT.
- A two-row signature block at the end (Shruggie LLC and Acme), party names and titles supplied by the operator.
- Footer right-aligned: `Statement of Work - Page X of Y`, Geist 10 PT, Gray 600.
- Six TTFs embedded into the archive under `word/fonts/`.

**Content array (page break on Signatures only):**

```js
content: [
  { type: 'h1', text: '1. Engagement Summary' },
  { type: 'body', text: '...' },
  { type: 'h1', text: '2. Scope' },
  { type: 'body', text: '...' },
  { type: 'h1', text: '3. Deliverables' },
  { type: 'body', text: '...' },
  { type: 'h1', text: '4. Milestones and Schedule' },
  { type: 'body', text: '...' },
  { type: 'h1', text: '5. Fees and Payment' },
  { type: 'body', text: '...' },
  { type: 'h1', text: '6. Acceptance Criteria' },
  { type: 'body', text: '...' },
  { type: 'h1', text: '7. Signatures', pageBreakBefore: true }, // only this H1 gets a page break
  { type: 'body', text: '...' },
]
```

Saved to `./acme-observability-sow.docx`.

### Example: Invoice

**User input:**

```
Make me an invoice for $4,800 to Acme Corp for the November observability work. Net 30, no tax.
```

**Expected output:**

A single `.docx` file containing:

- The lightbg logo as the first paragraph.
- Two-column borderless header table: left column has `Shruggie LLC` plus mailing address, email, phone (left-aligned, Geist 11 PT body). Right column has eyebrow labels `INVOICE #`, `ISSUE DATE`, `DUE DATE` with the corresponding values in Body Bold, right-aligned.
- Below the header, `BILL TO:` eyebrow label followed by Acme Corp's name and address.
- Line-items table: columns `DESCRIPTION`, `QTY`, `RATE`, `AMOUNT`. One row for the November observability work at $4,800. Borders Gray 200 0.5 PT. Header row Gray 100 background, Geist Medium 10 PT.
- Summary rows: Subtotal `$4,800.00`, Total `$4,800.00` (Tax row omitted because tax is zero). Total row has a single top border in Gray 950, 1 PT.
- Horizontal rule, then payment-terms block: `Net 30 from issue date`, accepted methods bullet list, remit-to address matching the sender block, contact email.
- Footer right-aligned: `Shruggie LLC | Invoice {INVOICE_NUMBER} | Payment terms: Net 30 from issue date`. Page numbers only render if the invoice spans more than one page.
- Six TTFs embedded.

Saved to `./acme-corp-november-invoice.docx`.

## Additional Resources

- [`assets/brand-rules.md`](assets/brand-rules.md): light-surface brand reference. Color tokens, typography, voice rules, tagline, attribution, sub-brand exclusion list, output file rules. Consult this when you need an exact value rather than guessing.
- [`assets/style-spec.json`](assets/style-spec.json): machine-readable named-styles map. Authoritative source for every style the document scaffold materializes (TITLE, SUBTITLE, H1 through H6, Body, lists, table cells, code, eyebrow, footer, caption, hyperlink, horizontal rule, table default). The scaffold reads this file rather than hardcoding values.
- [`assets/doc-type-defaults.json`](assets/doc-type-defaults.json): per-document-type defaults (TOC, top block, footer, page-break rule, tagline inclusion) for SOW, SOA, MSA, Internal Report, Invoice, and Letter. Includes the binding `operatorCollaborationRequired` block and the invoice line-items table spec.
- [`assets/document-template.js`](assets/document-template.js): docx-js scaffold. Exports `buildDocument({ docType, title, subtitle, content, overrides, assetsDir, partyMetadata })` plus helpers (`Packer`, `measurePngAspect`, `roundToHalf`, `ptToTwip`, `ptToEmu`). No layout or style values are hardcoded; everything comes from the JSON files.
- [`assets/embed-fonts.py`](assets/embed-fonts.py): post-generation font embedding. Unpacks the `.docx`, drops the six bundled TTFs into `word/fonts/`, patches `word/fontTable.xml` and `word/_rels/fontTable.xml.rels`, sets the embed flag in `word/settings.xml`, normalizes the child sequence of `<w:settings>` (`CT_Settings`) and each `<w:font>` (`CT_Font`) to satisfy strict OOXML validators, and repacks. Has a `--verify-only` mode that checks every `r:id` on a `<w:embed*>` resolves to a relationship whose Target is a part present in the archive, and fails on a deliberately broken fixture (font relationships stripped, or stray font rels in `document.xml.rels`).
- [`assets/brand/logo/logo-lightbg.png`](assets/brand/logo/logo-lightbg.png): byte-identical local copy of `https://cdn.shruggie.tech/brand/logo/logo-lightbg.png`. The build path reads from this file on disk; the CDN URL exists only to trace the canonical origin.
- [`assets/fonts/`](assets/fonts/): byte-identical local copies of the six TTFs from `https://cdn.shruggie.tech/brand/fonts/ttf/` (SpaceGrotesk-Medium, SpaceGrotesk-Bold, Geist-Regular, Geist-Medium, Geist-Italic, GeistMono-Regular). These are the complete embed list; refresh them when the CDN updates.
