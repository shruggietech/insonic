#!/usr/bin/env python3
"""embed-fonts.py: post-generation font embedding for shruggie-docs.

Takes a .docx produced by document-template.js and embeds the six bundled TTFs
into the file. docx-js does not expose font embedding directly; this script
unpacks the .docx, drops the TTFs into word/fonts/, patches word/fontTable.xml
to reference each embed, sets the embedTrueTypeFonts flag in word/settings.xml,
and repacks.

Usage:
    python embed-fonts.py <path-to-docx>

The script is idempotent: running it twice on the same file will not duplicate
font entries.

Runtime requirements:
    - Python 3.9+
    - lxml (for OOXML manipulation)
    - The bundled assets/fonts/ directory adjacent to this script

The script does not require the public docx skill at runtime; it implements
unpack/pack inline using the Python standard library zipfile module. The
public skill's scripts/office/unpack.py and pack.py produce equivalent output
and may be substituted by callers that prefer the public toolchain.
"""

from __future__ import annotations

import argparse
import hashlib
import os
import shutil
import sys
import tempfile
import zipfile
from pathlib import Path

try:
    from lxml import etree
except ImportError as exc:
    raise SystemExit(
        "lxml is required. Install it in the skill execution environment "
        "with `pip install lxml`."
    ) from exc


SCRIPT_DIR = Path(__file__).resolve().parent
FONTS_DIR = SCRIPT_DIR / "fonts"

NS_W = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
NS_R = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
NS_REL = "http://schemas.openxmlformats.org/package/2006/relationships"
NSMAP_W = {"w": NS_W, "r": NS_R}

FONT_BINDINGS = [
    {
        "file": "SpaceGrotesk-Bold.ttf",
        "family": "Space Grotesk",
        "slot": "embedBold",
    },
    {
        "file": "SpaceGrotesk-Medium.ttf",
        "family": "Space Grotesk",
        "slot": "embedRegular",
    },
    {
        "file": "Geist-Regular.ttf",
        "family": "Geist",
        "slot": "embedRegular",
    },
    {
        "file": "Geist-Medium.ttf",
        "family": "Geist",
        "slot": "embedBold",
    },
    {
        "file": "Geist-Italic.ttf",
        "family": "Geist",
        "slot": "embedItalic",
    },
    {
        "file": "GeistMono-Regular.ttf",
        "family": "Geist Mono",
        "slot": "embedRegular",
    },
]


# Ordered child sequence of CT_Settings (the type of <w:settings>), per
# ECMA-376 Part 1. A stable sort against this list places known elements
# in their schema slot; unknown elements preserve their relative order and
# trail the known elements. Microsoft Word is lenient about ordering, but
# strict validators (including scripts/office/validate.py in the public
# docx skill) reject out-of-sequence children, so the script normalizes
# the order before writing the part.
CT_SETTINGS_ORDER = [
    "writeProtection", "view", "zoom", "removePersonalInformation",
    "removeDateAndTime", "doNotDisplayPageBoundaries",
    "displayBackgroundShape", "printPostScriptOverText",
    "printFractionalCharacterWidth", "printFormsData", "embedTrueTypeFonts",
    "embedSystemFonts", "saveSubsetFonts", "saveFormsData", "mirrorMargins",
    "alignBordersAndEdges", "bordersDoNotSurroundHeader",
    "bordersDoNotSurroundFooter", "gutterAtTop", "hideSpellingErrors",
    "hideGrammaticalErrors", "activeWritingStyle", "proofState",
    "formsDesign", "attachedTemplate", "linkStyles", "stylePaneFormatFilter",
    "stylePaneSortMethod", "documentType", "mailMerge", "revisionView",
    "trackChanges", "doNotTrackMoves", "doNotTrackFormatting",
    "documentProtection", "autoFormatOverride", "styleLockTheme",
    "styleLockQFSet", "defaultTabStop", "autoHyphenation",
    "consecutiveHyphenLimit", "hyphenationZone", "doNotHyphenateCaps",
    "showEnvelope", "summaryLength", "clickAndTypeStyle",
    "defaultTableStyle", "evenAndOddHeaders", "bookFoldRevPrinting",
    "bookFoldPrinting", "bookFoldPrintingSheets",
    "drawingGridHorizontalSpacing", "drawingGridVerticalSpacing",
    "displayHorizontalDrawingGridEvery", "displayVerticalDrawingGridEvery",
    "doNotUseMarginsForDrawingGridOrigin", "drawingGridHorizontalOrigin",
    "drawingGridVerticalOrigin", "doNotShadeFormData", "noPunctuationKerning",
    "characterSpacingControl", "printTwoOnOne", "strictFirstAndLastChars",
    "noLineBreaksAfter", "noLineBreaksBefore", "savePreviewPicture",
    "doNotValidateAgainstSchema", "saveInvalidXml", "ignoreMixedContent",
    "alwaysShowPlaceholderText", "doNotDemarcateInvalidXml",
    "saveXmlDataOnly", "useXSLTWhenSaving", "saveThroughXslt", "showXMLTags",
    "alwaysMergeEmptyNamespace", "updateFields", "hdrShapeDefaults",
    "footnotePr", "endnotePr", "compat", "docVars", "rsids", "mathPr",
    "attachedSchema", "themeFontLang", "clrSchemeMapping",
    "doNotIncludeSubdocsInStats", "doNotAutoCompressPictures", "forceUpgrade",
    "captions", "readModeInkLockDown", "smartTagType", "schemaLibrary",
    "shapeDefaults", "doNotEmbedSmartTags", "decimalSymbol", "listSeparator",
]

# Ordered child sequence of CT_Font (the type of each <w:font>), per
# ECMA-376 Part 1. The four embed* children must appear in this order
# and must follow any altName/panose1/charset/family/notTrueType/pitch/sig
# children that may already be present on the element.
CT_FONT_ORDER = [
    "altName", "panose1", "charset", "family", "notTrueType", "pitch", "sig",
    "embedRegular", "embedBold", "embedItalic", "embedBoldItalic",
]

# Image extensions a word/media/* part may legitimately carry. A media part
# whose extension is absent from this set (e.g. the '.undefined' produced when
# an ImageRun omits its type) has no inferable content type and can silently
# fail to render in strict OOXML viewers and server-side PDF/e-sign converters.
KNOWN_IMAGE_EXTS = {
    "png", "jpg", "jpeg", "gif", "bmp", "tif", "tiff", "emf", "wmf", "svg",
}


def reorder_children(parent, order: list[str], ns: str) -> None:
    """Reorder element children of `parent` into canonical sequence.

    Children whose namespace is `ns` and whose local name is in `order` are
    placed in `order` position. Children outside that set keep their original
    relative order and trail the known elements. The sort is stable and
    idempotent.
    """
    order_map = {name: i for i, name in enumerate(order)}
    children = list(parent)
    if not children:
        return

    def key(item):
        idx, child = item
        qname = etree.QName(child)
        if qname.namespace == ns and qname.localname in order_map:
            return (0, order_map[qname.localname], idx)
        return (1, 0, idx)

    indexed = sorted(enumerate(children), key=key)
    if [c for _, c in indexed] == children:
        return
    for child in children:
        parent.remove(child)
    for _, child in indexed:
        parent.append(child)


def unpack_docx(docx_path: Path, dest: Path) -> None:
    with zipfile.ZipFile(docx_path, "r") as zf:
        zf.extractall(dest)


def pack_docx(src: Path, docx_path: Path) -> None:
    if docx_path.exists():
        docx_path.unlink()
    with zipfile.ZipFile(docx_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for root, _dirs, files in os.walk(src):
            for f in files:
                full = Path(root) / f
                rel = full.relative_to(src)
                zf.write(full, arcname=str(rel).replace(os.sep, "/"))


def ensure_font_payload(extracted: Path) -> None:
    fonts_dir = extracted / "word" / "fonts"
    fonts_dir.mkdir(parents=True, exist_ok=True)
    for binding in FONT_BINDINGS:
        src = FONTS_DIR / binding["file"]
        if not src.is_file():
            raise SystemExit(f"Missing bundled font: {src}")
        dst = fonts_dir / binding["file"]
        shutil.copyfile(src, dst)


def update_content_types(extracted: Path) -> None:
    ct_path = extracted / "[Content_Types].xml"
    tree = etree.parse(str(ct_path))
    root = tree.getroot()
    existing_defaults = {
        d.get("Extension", "").lower(): d
        for d in root.findall("{http://schemas.openxmlformats.org/package/2006/content-types}Default")
    }
    if "ttf" not in existing_defaults:
        default_el = etree.SubElement(
            root,
            "{http://schemas.openxmlformats.org/package/2006/content-types}Default",
        )
        default_el.set("Extension", "ttf")
        default_el.set("ContentType", "application/x-font-ttf")
    tree.write(str(ct_path), xml_declaration=True, encoding="UTF-8", standalone=True)


def update_font_table_rels(extracted: Path) -> dict[str, str]:
    """Add a relationship per font binding to word/_rels/fontTable.xml.rels
    and return a mapping from font filename to relationship Id.

    The r:id references on <w:embed*> elements live inside word/fontTable.xml,
    so the relationships they resolve against must live in that part's
    relationships file (word/_rels/fontTable.xml.rels) per the Open Packaging
    Conventions. Writing them to document.xml.rels would leave the r:id
    references unresolvable from fontTable.xml and silently break font
    embedding in Microsoft Word.
    """
    rels_path = extracted / "word" / "_rels" / "fontTable.xml.rels"
    rels_path.parent.mkdir(parents=True, exist_ok=True)
    rels_ns = "http://schemas.openxmlformats.org/package/2006/relationships"
    if rels_path.exists():
        tree = etree.parse(str(rels_path))
        root = tree.getroot()
    else:
        root = etree.Element(f"{{{rels_ns}}}Relationships", nsmap={None: rels_ns})
        tree = etree.ElementTree(root)

    existing_ids = set()
    existing_targets = {}
    for rel in root.findall(f"{{{rels_ns}}}Relationship"):
        existing_ids.add(rel.get("Id"))
        existing_targets[rel.get("Target")] = rel.get("Id")

    def next_id() -> str:
        n = 1
        while f"rIdFont{n}" in existing_ids:
            n += 1
        new = f"rIdFont{n}"
        existing_ids.add(new)
        return new

    mapping: dict[str, str] = {}
    font_rel_type = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/font"
    for binding in FONT_BINDINGS:
        target = f"fonts/{binding['file']}"
        if target in existing_targets:
            mapping[binding["file"]] = existing_targets[target]
            continue
        rid = next_id()
        rel_el = etree.SubElement(root, f"{{{rels_ns}}}Relationship")
        rel_el.set("Id", rid)
        rel_el.set("Type", font_rel_type)
        rel_el.set("Target", target)
        mapping[binding["file"]] = rid

    tree.write(str(rels_path), xml_declaration=True, encoding="UTF-8", standalone=True)
    return mapping


def update_font_table(extracted: Path, font_to_rid: dict[str, str]) -> None:
    ft_path = extracted / "word" / "fontTable.xml"
    w_ns = NS_W
    r_ns = NS_R
    nsmap = {"w": w_ns, "r": r_ns}
    if ft_path.exists():
        tree = etree.parse(str(ft_path))
        root = tree.getroot()
    else:
        root = etree.Element(f"{{{w_ns}}}fonts", nsmap=nsmap)
        tree = etree.ElementTree(root)

    fonts_by_name: dict[str, etree._Element] = {}
    for f in root.findall(f"{{{w_ns}}}font"):
        fonts_by_name[f.get(f"{{{w_ns}}}name") or ""] = f

    for binding in FONT_BINDINGS:
        family = binding["family"]
        font_el = fonts_by_name.get(family)
        if font_el is None:
            font_el = etree.SubElement(root, f"{{{w_ns}}}font")
            font_el.set(f"{{{w_ns}}}name", family)
            fonts_by_name[family] = font_el

        slot = binding["slot"]
        existing = font_el.find(f"{{{w_ns}}}{slot}")
        if existing is not None:
            font_el.remove(existing)
        embed_el = etree.SubElement(font_el, f"{{{w_ns}}}{slot}")
        embed_el.set(f"{{{r_ns}}}id", font_to_rid[binding["file"]])
        embed_el.set(f"{{{w_ns}}}fontKey", "{00000000-0000-0000-0000-000000000000}")

    # Bindings can be visited in any order (Bold before Regular for Space
    # Grotesk, in the current binding list). CT_Font is an ordered
    # xsd:sequence: the four embed* children must appear in the canonical
    # order, after any altName/panose1/charset/family/notTrueType/pitch/sig
    # children. Normalize each <w:font>'s children after all bindings have
    # been applied.
    for font_el in root.findall(f"{{{w_ns}}}font"):
        reorder_children(font_el, CT_FONT_ORDER, w_ns)

    tree.write(str(ft_path), xml_declaration=True, encoding="UTF-8", standalone=True)


def update_settings(extracted: Path) -> None:
    s_path = extracted / "word" / "settings.xml"
    w_ns = NS_W
    if s_path.exists():
        tree = etree.parse(str(s_path))
        root = tree.getroot()
    else:
        nsmap = {"w": w_ns}
        root = etree.Element(f"{{{w_ns}}}settings", nsmap=nsmap)
        tree = etree.ElementTree(root)

    embed_flag = root.find(f"{{{w_ns}}}embedTrueTypeFonts")
    if embed_flag is None:
        embed_flag = etree.SubElement(root, f"{{{w_ns}}}embedTrueTypeFonts")
        embed_flag.set(f"{{{w_ns}}}val", "true")

    save_subset = root.find(f"{{{w_ns}}}saveSubsetFonts")
    if save_subset is None:
        save_subset = etree.SubElement(root, f"{{{w_ns}}}saveSubsetFonts")
        save_subset.set(f"{{{w_ns}}}val", "false")

    # SubElement appends always go to the end of the parent. CT_Settings is
    # an ordered xsd:sequence, and embedTrueTypeFonts/saveSubsetFonts sit
    # early in that sequence (before evenAndOddHeaders and compat). Normalize
    # the order so strict validators do not reject the part.
    reorder_children(root, CT_SETTINGS_ORDER, w_ns)

    tree.write(str(s_path), xml_declaration=True, encoding="UTF-8", standalone=True)


def embed(docx_path: Path) -> None:
    if not docx_path.is_file():
        raise SystemExit(f"Not a file: {docx_path}")
    with tempfile.TemporaryDirectory(prefix="shruggie-docs-embed-") as tmp:
        tmpdir = Path(tmp)
        unpack_docx(docx_path, tmpdir)
        ensure_font_payload(tmpdir)
        update_content_types(tmpdir)
        rid_map = update_font_table_rels(tmpdir)
        update_font_table(tmpdir, rid_map)
        update_settings(tmpdir)
        pack_docx(tmpdir, docx_path)


def verify(docx_path: Path) -> None:
    """Verify the embed result is functionally correct.

    Beyond presence checks, this confirms that the relationship graph that
    actually makes the embedded fonts usable in Microsoft Word is intact:
    every r:id on a <w:embed*> element in word/fontTable.xml must resolve to
    a Relationship in word/_rels/fontTable.xml.rels, and that relationship's
    Target must be a part present in the archive. It also confirms no font
    relationships have leaked into word/_rels/document.xml.rels (a prior
    defect that left embedding silently broken) and that the CT_Settings
    and CT_Font child sequences are normalized.

    Raises SystemExit on any failure. A passing run is necessary but not
    sufficient: the functional proof is that the document opens in Microsoft
    Word and renders the brand typefaces on a machine without the six TTFs
    installed.
    """
    rels_ns = "http://schemas.openxmlformats.org/package/2006/relationships"
    font_rel_type = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/font"
    expected = {b["file"] for b in FONT_BINDINGS}
    expected_families = {b["family"] for b in FONT_BINDINGS}

    with zipfile.ZipFile(docx_path, "r") as zf:
        names = set(zf.namelist())

        # 1. All six TTFs are physically present at the expected paths.
        for f in expected:
            arc = f"word/fonts/{f}"
            if arc not in names:
                raise SystemExit(f"Verification failed: {arc} not in archive.")

        # 2. fontTable.xml exists and references every expected family.
        if "word/fontTable.xml" not in names:
            raise SystemExit("Verification failed: word/fontTable.xml missing.")
        with zf.open("word/fontTable.xml") as fh:
            ft_root = etree.fromstring(fh.read())
        seen_families = {
            (f.get(f"{{{NS_W}}}name") or "")
            for f in ft_root.findall(f"{{{NS_W}}}font")
        }
        missing = expected_families - seen_families
        if missing:
            raise SystemExit(
                f"Verification failed: families not referenced in fontTable.xml: {sorted(missing)}"
            )

        # 3. Collect every r:id used on <w:embed*> elements.
        rid_refs: list[tuple[str, str, str]] = []  # (family, slot, rid)
        for font_el in ft_root.findall(f"{{{NS_W}}}font"):
            family = font_el.get(f"{{{NS_W}}}name") or ""
            for child in font_el:
                qname = etree.QName(child)
                if qname.namespace != NS_W:
                    continue
                if qname.localname not in {
                    "embedRegular", "embedBold", "embedItalic", "embedBoldItalic",
                }:
                    continue
                rid = child.get(f"{{{NS_R}}}id")
                if not rid:
                    raise SystemExit(
                        f"Verification failed: <w:{qname.localname}> on font "
                        f"{family!r} has no r:id."
                    )
                rid_refs.append((family, qname.localname, rid))

        # 4. fontTable.xml.rels exists and provides the relationships those
        #    r:id references resolve against.
        ft_rels_path = "word/_rels/fontTable.xml.rels"
        if ft_rels_path not in names:
            raise SystemExit(
                f"Verification failed: {ft_rels_path} missing. Font r:id "
                f"references in fontTable.xml will not resolve."
            )
        with zf.open(ft_rels_path) as fh:
            ft_rels_root = etree.fromstring(fh.read())
        rid_to_target: dict[str, str] = {}
        for rel in ft_rels_root.findall(f"{{{rels_ns}}}Relationship"):
            rid_to_target[rel.get("Id") or ""] = rel.get("Target") or ""

        for family, slot, rid in rid_refs:
            if rid not in rid_to_target:
                raise SystemExit(
                    f"Verification failed: <w:{slot}> on font {family!r} "
                    f"references r:id {rid!r}, which is not in {ft_rels_path}. "
                    f"Valid IDs: {sorted(rid_to_target)}"
                )
            target = rid_to_target[rid]
            # Targets in fontTable.xml.rels are relative to word/.
            target_path = f"word/{target}" if not target.startswith("/") else target.lstrip("/")
            if target_path not in names:
                raise SystemExit(
                    f"Verification failed: relationship {rid!r} targets "
                    f"{target!r}, which is not present in the archive."
                )

        # 5. document.xml.rels must not contain any font-type relationships
        #    (legacy bug: the script used to write them there, where nothing
        #    consumed them).
        doc_rels_path = "word/_rels/document.xml.rels"
        if doc_rels_path in names:
            with zf.open(doc_rels_path) as fh:
                doc_rels_root = etree.fromstring(fh.read())
            stray = [
                r.get("Id")
                for r in doc_rels_root.findall(f"{{{rels_ns}}}Relationship")
                if (r.get("Type") or "") == font_rel_type
            ]
            if stray:
                raise SystemExit(
                    f"Verification failed: {doc_rels_path} contains font "
                    f"relationships {stray}. Font relationships belong in "
                    f"{ft_rels_path}, not document.xml.rels."
                )

        # 6. CT_Font child order: regression guard for D2.
        font_order_map = {n: i for i, n in enumerate(CT_FONT_ORDER)}
        for font_el in ft_root.findall(f"{{{NS_W}}}font"):
            family = font_el.get(f"{{{NS_W}}}name") or ""
            last = -1
            for child in font_el:
                qname = etree.QName(child)
                if qname.namespace != NS_W:
                    continue
                pos = font_order_map.get(qname.localname)
                if pos is None:
                    continue
                if pos < last:
                    raise SystemExit(
                        f"Verification failed: <w:font name={family!r}> "
                        f"children are out of CT_Font sequence "
                        f"(<w:{qname.localname}> appears after a later element)."
                    )
                last = pos

        # 7. CT_Settings child order: regression guard for D1.
        if "word/settings.xml" in names:
            with zf.open("word/settings.xml") as fh:
                settings_root = etree.fromstring(fh.read())
            settings_order_map = {n: i for i, n in enumerate(CT_SETTINGS_ORDER)}
            last = -1
            for child in settings_root:
                qname = etree.QName(child)
                if qname.namespace != NS_W:
                    continue
                pos = settings_order_map.get(qname.localname)
                if pos is None:
                    continue
                if pos < last:
                    raise SystemExit(
                        f"Verification failed: word/settings.xml children "
                        f"are out of CT_Settings sequence "
                        f"(<w:{qname.localname}> appears after a later element)."
                    )
                last = pos

        # 8. Media integrity: every word/media/* part must have a recognized
        #    image extension and a resolvable content type (a <Default> for its
        #    extension or an <Override> for its part name). Guards against the
        #    '.undefined' media class of defect, which passed the public docx
        #    validator despite having no valid extension or content type.
        ct_ns = "http://schemas.openxmlformats.org/package/2006/content-types"
        media_parts = [n for n in names if n.startswith("word/media/") and not n.endswith("/")]
        if media_parts:
            if "[Content_Types].xml" not in names:
                raise SystemExit(
                    "Verification failed: [Content_Types].xml missing while "
                    "word/media/* parts are present."
                )
            with zf.open("[Content_Types].xml") as fh:
                ct_root = etree.fromstring(fh.read())
            default_exts = {
                (d.get("Extension") or "").lower()
                for d in ct_root.findall(f"{{{ct_ns}}}Default")
            }
            override_parts = {
                o.get("PartName") or ""
                for o in ct_root.findall(f"{{{ct_ns}}}Override")
            }
            for part in media_parts:
                ext = part.rsplit(".", 1)[-1].lower() if "." in part.rsplit("/", 1)[-1] else ""
                if ext not in KNOWN_IMAGE_EXTS:
                    raise SystemExit(
                        f"Verification failed: media part {part!r} has no "
                        f"recognized image extension (got {ext!r}). Known "
                        f"extensions: {sorted(KNOWN_IMAGE_EXTS)}."
                    )
                part_name = "/" + part
                if ext not in default_exts and part_name not in override_parts:
                    raise SystemExit(
                        f"Verification failed: media part {part!r} has no "
                        f"content type (no <Default Extension={ext!r}> and no "
                        f"<Override PartName={part_name!r}> in [Content_Types].xml)."
                    )


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description="Embed bundled fonts into a .docx")
    parser.add_argument("docx", type=Path, help="Path to the .docx file to modify in place")
    parser.add_argument("--verify-only", action="store_true", help="Skip embedding; only verify an already-embedded file")
    args = parser.parse_args(argv)

    if not args.verify_only:
        embed(args.docx)
    verify(args.docx)
    print(f"OK: {args.docx}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
