/*
 * shruggie-docs document-template.js
 *
 * docx-js scaffold for the shruggie-docs skill. Reads style-spec.json and
 * doc-type-defaults.json at runtime and constructs a docx Document per the
 * layout spec. No style or layout values are hardcoded here; all values
 * come from the JSON files so the skill can be retuned without code edits.
 *
 * Usage:
 *   const { buildDocument } = require('./document-template');
 *   const doc = buildDocument({
 *     docType: 'SOW',
 *     title: 'Statement of Work',
 *     subtitle: 'Project Alpha',
 *     content: [...],          // array of section descriptors (see below)
 *     overrides: {},           // operator-supplied overrides for type defaults
 *     assetsDir: __dirname,    // absolute path to assets directory
 *     partyMetadata: {         // SOW/SOA/MSA only; key order is preserved
 *       Client: 'Acme Corp',
 *       Partner: 'Shruggie LLC',
 *       Date: 'June 1, 2026',  // optional; defaults to today's M/D/YYYY when empty
 *     },
 *   });
 *   const buffer = await Packer.toBuffer(doc);
 *   fs.writeFileSync(outPath, buffer);
 *   // Then run embed-fonts.py against outPath to embed the six TTFs.
 *
 * The caller is responsible for translating user-facing content into the
 * section-descriptor array. See SKILL.md for the descriptor grammar.
 */

const fs = require('fs');
const path = require('path');

let docx;
try {
  docx = require('docx');
} catch (err) {
  throw new Error(
    'docx-js is required. Run `npm install docx` in the skill execution environment, or invoke this template from a host that has the package installed.'
  );
}

const {
  Document,
  Packer,
  Paragraph,
  TextRun,
  ImageRun,
  Header,
  Footer,
  PageNumber,
  AlignmentType,
  HeightRule,
  Table,
  TableRow,
  TableCell,
  WidthType,
  BorderStyle,
  ShadingType,
  HorizontalPositionAlign,
  VerticalPositionAlign,
  HorizontalPositionRelativeFrom,
  VerticalPositionRelativeFrom,
  PageOrientation,
  VerticalAlign,
  convertInchesToTwip,
  LevelFormat,
  HeadingLevel,
  LineRuleType,
} = docx;

const PT_TO_TWIP = 20;
const ptToTwip = (pt) => Math.round(pt * PT_TO_TWIP);
const ptToEmu = (pt) => Math.round(pt * 12700);

// The exact ShruggieTech tagline. Do not paraphrase. Marketing-adjacent
// documents only (Internal Report covers, capabilities one-pagers).
const TAGLINE = "¯\\_(ツ)_/¯ We'll figure it out.";

function todayMMDDYYYY() {
  const now = new Date();
  return `${now.getMonth() + 1}/${now.getDate()}/${now.getFullYear()}`;
}

function loadJSON(p) {
  return JSON.parse(fs.readFileSync(p, 'utf8'));
}

function alignmentFor(name) {
  switch (name) {
    case 'START': return AlignmentType.START;
    case 'END': return AlignmentType.END;
    case 'CENTER': return AlignmentType.CENTER;
    case 'JUSTIFY': return AlignmentType.JUSTIFIED;
    default: return AlignmentType.START;
  }
}

function measurePngAspect(pngPath) {
  const buf = fs.readFileSync(pngPath);
  if (buf.length < 24) {
    throw new Error(`PNG too small to read dimensions: ${pngPath}`);
  }
  const sig = buf.slice(0, 8).toString('hex');
  if (sig !== '89504e470d0a1a0a') {
    throw new Error(`Not a PNG: ${pngPath}`);
  }
  const width = buf.readUInt32BE(16);
  const height = buf.readUInt32BE(20);
  if (width === 0 || height === 0) {
    throw new Error(`PNG has zero dimension: ${pngPath}`);
  }
  return { width, height, aspect: width / height };
}

function roundToHalf(n) {
  return Math.round(n * 2) / 2;
}

function buildLogoParagraph(styleSpec, assetsDir) {
  const logoCfg = styleSpec.logo;
  const pngPath = path.join(assetsDir, logoCfg.path);
  const { aspect } = measurePngAspect(pngPath);
  const widthPT = logoCfg.widthPT;
  const heightPT = roundToHalf(widthPT / aspect);

  return new Paragraph({
    alignment: alignmentFor(logoCfg.alignment),
    spacing: {
      before: ptToTwip(logoCfg.marginsPT.top),
      after: ptToTwip(logoCfg.marginsPT.bottom),
    },
    indent: {
      left: ptToTwip(logoCfg.marginsPT.left),
      right: ptToTwip(logoCfg.marginsPT.right),
    },
    children: [
      new ImageRun({
        type: 'png',
        data: fs.readFileSync(pngPath),
        transformation: {
          width: ptToEmu(widthPT) / 9525,
          height: ptToEmu(heightPT) / 9525,
        },
        altText: {
          title: logoCfg.altText,
          description: logoCfg.altText,
          name: logoCfg.altText,
        },
      }),
    ],
  });
}

function styleRun(text, styleName, styleSpec, opts) {
  const s = styleSpec.styles[styleName];
  if (!s) throw new Error(`Unknown style: ${styleName}`);
  const runOpts = {
    text,
    font: s.font === 'inherit' ? undefined : s.font,
    size: s.sizePT === 'inherit' ? undefined : Math.round(s.sizePT * 2),
    bold: typeof s.weight === 'number' && s.weight >= 600,
    italics: s.italic === true,
    color: s.color === 'inherit' ? undefined : s.color,
  };
  if (opts && opts.italics) runOpts.italics = true;
  if (opts && opts.bold) runOpts.bold = true;
  return new TextRun(runOpts);
}

function styleParagraph(text, styleName, styleSpec, paragraphOpts) {
  const s = styleSpec.styles[styleName];
  if (!s) throw new Error(`Unknown style: ${styleName}`);
  const spacing = {};
  const extraSpaceAbovePT = paragraphOpts && typeof paragraphOpts.extraSpaceAbovePT === 'number'
    ? paragraphOpts.extraSpaceAbovePT
    : 0;
  if (typeof s.spaceAbovePT === 'number' || extraSpaceAbovePT !== 0) {
    const baseSpaceAbovePT = typeof s.spaceAbovePT === 'number' ? s.spaceAbovePT : 0;
    spacing.before = ptToTwip(baseSpaceAbovePT + extraSpaceAbovePT);
  }
  if (typeof s.spaceBelowPT === 'number') spacing.after = ptToTwip(s.spaceBelowPT);
  if (typeof s.lineSpacing === 'number') {
    spacing.line = s.lineSpacing;
    spacing.lineRule = LineRuleType.AUTO;
  }

  const p = {
    alignment: alignmentFor(s.alignment),
    spacing,
    children: [styleRun(text, styleName, styleSpec)],
  };
  if (paragraphOpts && paragraphOpts.pageBreakBefore) p.pageBreakBefore = true;
  return new Paragraph(p);
}

function buildFooterParagraph(footerCfg, docTitle) {
  const template = footerCfg.template || footerCfg.defaultTemplate;
  const parts = [];
  const split = template.split(/(\{[A-Z_]+\})/g);
  for (const piece of split) {
    if (piece === '{PAGE}') {
      parts.push(new TextRun({ children: [PageNumber.CURRENT], font: 'Geist', size: 20, color: '6B6B6B' }));
    } else if (piece === '{NUMPAGES}') {
      parts.push(new TextRun({ children: [PageNumber.TOTAL_PAGES], font: 'Geist', size: 20, color: '6B6B6B' }));
    } else if (piece === '{DOCUMENT_TITLE}') {
      parts.push(new TextRun({ text: docTitle, font: 'Geist', size: 20, color: '6B6B6B' }));
    } else if (piece.length > 0) {
      parts.push(new TextRun({ text: piece, font: 'Geist', size: 20, color: '6B6B6B' }));
    }
  }
  return new Paragraph({
    alignment: alignmentFor(footerCfg.alignment || 'END'),
    children: parts,
  });
}

function buildTaglineFooterParagraph(footerCfg) {
  return new Paragraph({
    alignment: alignmentFor(footerCfg.alignment || 'END'),
    children: [new TextRun({ text: TAGLINE, font: 'Geist', size: 20, color: '6B6B6B' })],
  });
}

function buildHorizontalRule(styleSpec, overrides) {
  const r = styleSpec.horizontalRule;
  const localOverrides = overrides || {};
  const spaceAbovePT = typeof localOverrides.spaceAbovePT === 'number' ? localOverrides.spaceAbovePT : r.spaceAbovePT;
  const spaceBelowPT = typeof localOverrides.spaceBelowPT === 'number' ? localOverrides.spaceBelowPT : r.spaceBelowPT;
  return new Paragraph({
    spacing: {
      before: ptToTwip(spaceAbovePT),
      after: ptToTwip(spaceBelowPT),
    },
    border: {
      bottom: {
        color: r.color,
        space: 1,
        style: BorderStyle.SINGLE,
        size: Math.max(1, Math.round(r.weightPT * 8)),
      },
    },
    children: [],
  });
}

function buildTOCBlock(content, styleSpec) {
  const tocParagraphs = [styleParagraph('Table of Contents', 'H2', styleSpec)];
  for (const node of content || []) {
    if (node.type === 'h1') {
      tocParagraphs.push(new Paragraph({
        alignment: AlignmentType.START,
        spacing: { after: ptToTwip(4), line: 276, lineRule: LineRuleType.AUTO },
        children: [new TextRun({ text: node.text, font: 'Geist', size: 22, color: '0A0A0A' })],
      }));
    }
  }
  tocParagraphs.push(new Paragraph({ pageBreakBefore: true, children: [] }));
  return tocParagraphs;
}

function renderContent(content, styleSpec, tableDefaults) {
  const out = [];
  let previousNodeType = null;
  // Each numbered list needs its own concrete numbering instance under the
  // shared 'default-numbered' reference, or docx-js runs one continuous
  // counter across every numbered paragraph and later lists continue the
  // earlier count instead of restarting at 1. Increment once per numbered node.
  let numberedInstance = 0;
  for (const node of content) {
    const extraSpaceAbovePT = previousNodeType === 'table' ? 8 : 0;
    switch (node.type) {
      case 'title':
        out.push(styleParagraph(node.text, 'TITLE', styleSpec));
        break;
      case 'subtitle':
        out.push(styleParagraph(node.text, 'SUBTITLE', styleSpec));
        break;
      case 'h1':
        out.push(styleParagraph(node.text, 'H1', styleSpec, {
          pageBreakBefore: node.pageBreakBefore,
          extraSpaceAbovePT,
        }));
        break;
      case 'h2':
        out.push(styleParagraph(node.text, 'H2', styleSpec, { extraSpaceAbovePT }));
        break;
      case 'h3':
        out.push(styleParagraph(node.text, 'H3', styleSpec, { extraSpaceAbovePT }));
        break;
      case 'h4':
        out.push(styleParagraph(node.text, 'H4', styleSpec, { extraSpaceAbovePT }));
        break;
      case 'h5':
        out.push(styleParagraph(node.text, 'H5', styleSpec, { extraSpaceAbovePT }));
        break;
      case 'h6':
        out.push(styleParagraph(node.text, 'H6', styleSpec, { extraSpaceAbovePT }));
        break;
      case 'body':
        out.push(styleParagraph(node.text, 'Body', styleSpec, { extraSpaceAbovePT }));
        break;
      case 'eyebrow':
        out.push(styleParagraph((node.text || '').toUpperCase(), 'EyebrowLabel', styleSpec, { extraSpaceAbovePT }));
        break;
      case 'caption':
        out.push(styleParagraph(node.text, 'Caption', styleSpec, { extraSpaceAbovePT }));
        break;
      case 'hr':
        out.push(buildHorizontalRule(styleSpec));
        break;
      case 'pageBreak':
        out.push(new Paragraph({ children: [], pageBreakBefore: true }));
        break;
      case 'bullets':
        for (let i = 0; i < node.items.length; i++) {
          const item = node.items[i];
          const baseSpaceAbovePT = styleSpec.styles.BulletListItem.spaceAbovePT || 0;
          out.push(new Paragraph({
            alignment: AlignmentType.START,
            numbering: { reference: 'default-bullet', level: 0 },
            spacing: {
              before: ptToTwip(baseSpaceAbovePT + (i === 0 ? extraSpaceAbovePT : 0)),
              after: ptToTwip(4),
              line: 276,
              lineRule: LineRuleType.AUTO,
            },
            children: [styleRun(item, 'BulletListItem', styleSpec)],
          }));
        }
        break;
      case 'numbered':
        for (let i = 0; i < node.items.length; i++) {
          const baseSpaceAbovePT = styleSpec.styles.NumberedListItem.spaceAbovePT || 0;
          out.push(new Paragraph({
            alignment: AlignmentType.START,
            numbering: { reference: node.reference || 'default-numbered', level: 0, instance: numberedInstance },
            spacing: {
              before: ptToTwip(baseSpaceAbovePT + (i === 0 ? extraSpaceAbovePT : 0)),
              after: ptToTwip(4),
              line: 276,
              lineRule: LineRuleType.AUTO,
            },
            children: [styleRun(node.items[i], 'NumberedListItem', styleSpec)],
          }));
        }
        numberedInstance++;
        break;
      case 'codeBlock':
        out.push(new Paragraph({
          alignment: AlignmentType.START,
          spacing: { before: ptToTwip(4), after: ptToTwip(4), line: 276, lineRule: LineRuleType.AUTO },
          shading: { type: ShadingType.CLEAR, fill: 'F5F5F5', color: 'auto' },
          children: [styleRun(node.text, 'CodeBlock', styleSpec)],
        }));
        break;
      case 'table':
        out.push(buildTable(node, styleSpec, tableDefaults));
        break;
      case 'image': {
        // First-class image node. Centralizes the correctness concerns that
        // used to fall on every caller of the raw+ImageRun path: explicit PNG
        // type (see the logo .undefined defect), height derived from native
        // aspect, centering, and mandatory non-empty alt text.
        const imgPath = node.path;
        if (typeof imgPath !== 'string' || imgPath.trim() === '') {
          throw new Error("image node requires a non-empty 'path'.");
        }
        const altText = node.altText;
        if (typeof altText !== 'string' || altText.trim() === '') {
          throw new Error(`image node requires non-empty 'altText' (path: ${imgPath}).`);
        }
        const widthPT = typeof node.widthPT === 'number' && node.widthPT > 0 ? node.widthPT : 360;
        const { aspect } = measurePngAspect(imgPath);
        const heightPT = roundToHalf(widthPT / aspect);
        out.push(new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { before: ptToTwip(extraSpaceAbovePT), after: ptToTwip(4) },
          children: [
            new ImageRun({
              type: 'png',
              data: fs.readFileSync(imgPath),
              transformation: {
                width: ptToEmu(widthPT) / 9525,
                height: ptToEmu(heightPT) / 9525,
              },
              altText: { title: altText, description: altText, name: altText },
            }),
          ],
        }));
        if (typeof node.caption === 'string' && node.caption.trim() !== '') {
          out.push(styleParagraph(node.caption, 'Caption', styleSpec));
        }
        break;
      }
      case 'raw':
        if (node.paragraph instanceof Paragraph) out.push(node.paragraph);
        break;
      default:
        throw new Error(`Unknown content node type: ${node.type}`);
    }
    previousNodeType = node.type;
  }
  return out;
}

function buildTable(node, styleSpec, tableDefaults) {
  const tableCfg = styleSpec.table;
  const summaryRows = tableDefaults && Array.isArray(tableDefaults.summaryRows)
    ? tableDefaults.summaryRows
    : [];
  const totalSummaryRow = summaryRows.find((row) => row.label === 'Total' && row.topBorder);
  const totalBorderColor = totalSummaryRow && totalSummaryRow.topBorder && totalSummaryRow.topBorder.color
    ? totalSummaryRow.topBorder.color
    : '0A0A0A';
  const totalBorderWeightPT = totalSummaryRow && totalSummaryRow.topBorder && typeof totalSummaryRow.topBorder.weightPT === 'number'
    ? totalSummaryRow.topBorder.weightPT
    : 1;
  const totalBorderSize = Math.max(1, Math.round(totalBorderWeightPT * 8));
  const rows = [];
  if (node.header) {
    rows.push(new TableRow({
      tableHeader: true,
      children: node.header.map((cellText) => new TableCell({
        shading: { type: ShadingType.CLEAR, fill: tableCfg.headerRow.backgroundColor, color: 'auto' },
        children: [new Paragraph({
          alignment: AlignmentType.START,
          children: [new TextRun({
            text: cellText,
            font: tableCfg.headerRow.font,
            size: Math.round(tableCfg.headerRow.sizePT * 2),
            bold: tableCfg.headerRow.weight >= 600,
            color: tableCfg.headerRow.color,
          })],
        })],
      })),
    }));
  }
  for (let rowIndex = 0; rowIndex < node.rows.length; rowIndex++) {
    const row = node.rows[rowIndex];
    const isLastDataRow = rowIndex === node.rows.length - 1;
    rows.push(new TableRow({
      children: row.map((cellText) => {
        const cellConfig = {
          children: [new Paragraph({
            alignment: AlignmentType.START,
            children: [styleRun(String(cellText), 'TableCell', styleSpec)],
          })],
        };
        if (node.totalRow === true && isLastDataRow) {
          cellConfig.borders = {
            top: {
              style: BorderStyle.SINGLE,
              size: totalBorderSize,
              color: totalBorderColor,
            },
          };
        }
        return new TableCell(cellConfig);
      }),
    }));
  }
  return new Table({
    rows,
    width: { size: 100, type: WidthType.PERCENTAGE },
    borders: {
      top: { style: BorderStyle.SINGLE, size: 4, color: tableCfg.defaultBorder.color },
      bottom: { style: BorderStyle.SINGLE, size: 4, color: tableCfg.defaultBorder.color },
      left: { style: BorderStyle.SINGLE, size: 4, color: tableCfg.defaultBorder.color },
      right: { style: BorderStyle.SINGLE, size: 4, color: tableCfg.defaultBorder.color },
      insideHorizontal: { style: BorderStyle.SINGLE, size: 4, color: tableCfg.defaultBorder.color },
      insideVertical: { style: BorderStyle.SINGLE, size: 4, color: tableCfg.defaultBorder.color },
    },
  });
}

function buildPartyMetadataBlock(partyMetadata, styleSpec) {
  const topBottomBorder = { style: BorderStyle.SINGLE, size: 8, color: '000000' };
  const noBorder = { style: BorderStyle.NONE, size: 0, color: 'auto' };
  const entries = Object.entries(partyMetadata);
  const rows = [];
  for (let rowIndex = 0; rowIndex < entries.length; rowIndex++) {
    const [label, value] = entries[rowIndex];
    const isFirstRow = rowIndex === 0;
    const isLastRow = rowIndex === entries.length - 1;
    const resolvedValue =
      label === 'Date' && (value === undefined || value === null || String(value).trim() === '')
        ? todayMMDDYYYY()
        : value;
    const topBorder = isFirstRow ? topBottomBorder : noBorder;
    const bottomBorder = isLastRow ? topBottomBorder : noBorder;
    rows.push(new TableRow({
      children: [
        new TableCell({
          width: { size: 25, type: WidthType.PERCENTAGE },
          verticalAlign: VerticalAlign.CENTER,
          margins: {
            top: ptToTwip(8),
            bottom: ptToTwip(8),
            left: 0,
            right: 0,
          },
          borders: {
            top: topBorder,
            bottom: bottomBorder,
            left: noBorder,
            right: noBorder,
          },
          children: [new Paragraph({
            alignment: AlignmentType.START,
            spacing: { before: 0, after: 0, line: 240, lineRule: LineRuleType.AUTO },
            children: [
              new TextRun({ text: label + ':', font: 'Geist', size: 22, color: '0A0A0A', bold: true }),
            ],
          })],
        }),
        new TableCell({
          width: { size: 75, type: WidthType.PERCENTAGE },
          verticalAlign: VerticalAlign.CENTER,
          margins: {
            top: ptToTwip(8),
            bottom: ptToTwip(8),
            left: 0,
            right: 0,
          },
          borders: {
            top: topBorder,
            bottom: bottomBorder,
            left: noBorder,
            right: noBorder,
          },
          children: [new Paragraph({
            alignment: AlignmentType.START,
            spacing: { before: 0, after: 0, line: 240, lineRule: LineRuleType.AUTO },
            children: [
              new TextRun({ text: resolvedValue == null ? '' : String(resolvedValue), font: 'Geist', size: 22, color: '0A0A0A', bold: false }),
            ],
          })],
        }),
      ],
    }));
  }
  const table = new Table({
    rows,
    width: { size: 100, type: WidthType.PERCENTAGE },
    borders: {
      top: topBottomBorder,
      bottom: topBottomBorder,
      left: noBorder,
      right: noBorder,
      insideHorizontal: noBorder,
      insideVertical: noBorder,
    },
  });
  const spacer = new Paragraph({
    spacing: { before: 0, after: 0, line: 240, lineRule: LineRuleType.AUTO },
    children: [new TextRun({ text: '', font: 'Geist', size: 22 })],
  });
  return [table, spacer];
}

function buildDocument({ docType, title, subtitle, content, overrides, assetsDir, partyMetadata }) {
  if (!assetsDir) {
    throw new Error('assetsDir is required (absolute path to the skill assets directory).');
  }
  const styleSpec = loadJSON(path.join(assetsDir, 'style-spec.json'));
  const typeDefaults = loadJSON(path.join(assetsDir, 'doc-type-defaults.json'));
  const typeCfg = typeDefaults.documentTypes[docType];
  if (!typeCfg) {
    throw new Error(`Unknown document type: ${docType}. Valid types: ${Object.keys(typeDefaults.documentTypes).join(', ')}`);
  }
  const resolved = Object.assign({}, typeCfg, overrides || {});

  const sectionChildren = [];
  sectionChildren.push(buildLogoParagraph(styleSpec, assetsDir));

  const hasPartyMetadata = resolved.partyMetadataAfterTitle === true && partyMetadata && Object.keys(partyMetadata).length > 0;
  const addPartyMetadataSection = () => {
    if (!hasPartyMetadata) return;
    sectionChildren.push(...buildPartyMetadataBlock(partyMetadata, styleSpec));
  };

  if (resolved.topBlock === 'title-subtitle') {
    if (title) sectionChildren.push(styleParagraph(title, 'TITLE', styleSpec));
    if (subtitle) sectionChildren.push(styleParagraph(subtitle, 'SUBTITLE', styleSpec));
    addPartyMetadataSection();
  }

  if (resolved.topBlock === 'title-only') {
    if (title) sectionChildren.push(styleParagraph(title, 'TITLE', styleSpec));
    addPartyMetadataSection();
  }

  const includeTOC = resolved.tocDefault === true && (!overrides || overrides.tocInclusion !== false);
  if (includeTOC) {
    sectionChildren.push(...buildTOCBlock(content, styleSpec));
  }

  if (Array.isArray(content) && content.length > 0) {
    sectionChildren.push(...renderContent(content, styleSpec, typeDefaults.invoiceLineItemsTable));
  }

  if (resolved.taglineInFooter !== true && resolved.taglineInBody === true) {
    sectionChildren.push(styleParagraph(TAGLINE, 'Body', styleSpec));
  }

  const footerCfg = Object.assign({}, styleSpec.footer, {
    template: resolved.footerTemplate || styleSpec.footer.defaultTemplate,
    alignment: resolved.footerAlignment || styleSpec.footer.alignment,
  });

  // The footer template carries no tagline token, so when taglineInFooter is
  // set the tagline must be appended as its own footer line here. Otherwise
  // the flag maps to no output (the body-tagline branch above is suppressed
  // whenever taglineInFooter is true).
  const footerChildren = [];
  if (resolved.taglineInFooter === true) {
    footerChildren.push(buildTaglineFooterParagraph(footerCfg));
  }
  footerChildren.push(buildFooterParagraph(footerCfg, title || resolved.label));

  const doc = new Document({
    creator: 'Shruggie LLC',
    title: title || resolved.label,
    description: resolved.label,
    styles: {
      default: {
        document: {
          run: { font: 'Geist', size: 22, color: '0A0A0A' },
        },
      },
    },
    numbering: {
      config: [
        {
          reference: 'default-numbered',
          levels: [
            {
              level: 0,
              format: LevelFormat.DECIMAL,
              text: '%1.',
              alignment: AlignmentType.START,
              style: { paragraph: { indent: { left: ptToTwip(18), hanging: ptToTwip(18) } } },
            },
          ],
        },
        {
          reference: 'default-bullet',
          levels: [
            {
              level: 0,
              format: LevelFormat.BULLET,
              text: '\u2022',
              alignment: AlignmentType.START,
              style: {
                run: { font: 'Geist', size: 22, color: '0A0A0A' },
                paragraph: { indent: { left: ptToTwip(18), hanging: ptToTwip(12) } },
              },
            },
          ],
        },
      ],
    },
    sections: [
      {
        properties: {
          page: {
            size: {
              width: ptToTwip(styleSpec.page.sizePT.width),
              height: ptToTwip(styleSpec.page.sizePT.height),
              orientation: PageOrientation.PORTRAIT,
            },
            margin: {
              top: ptToTwip(styleSpec.page.marginsPT.top),
              bottom: ptToTwip(styleSpec.page.marginsPT.bottom),
              left: ptToTwip(styleSpec.page.marginsPT.left),
              right: ptToTwip(styleSpec.page.marginsPT.right),
            },
          },
        },
        footers: {
          default: new Footer({
            children: footerChildren,
          }),
        },
        children: sectionChildren,
      },
    ],
  });
  return doc;
}

module.exports = {
  buildDocument,
  Packer,
  todayMMDDYYYY,
  measurePngAspect,
  roundToHalf,
  ptToTwip,
  ptToEmu,
};
