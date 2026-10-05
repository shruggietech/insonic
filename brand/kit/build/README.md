# insonic Build Notes

The constructed logo helper preserves the approved A1 Glacial blue Full and Reduced geometry. All generated derivatives must pass the source-bound identity continuity checks.

Run `python brands/insonic/build/run_paths.py` from the repository to regenerate the approved path arrays. The launcher discovers the repository's BrandBuilder library independently of the working directory. In a standalone kit, run `python build/run_paths.py --templates /path/to/brandbuilder/templates` with an installed BrandBuilder library. `mk_paths.py` is the immutable, source-bound construction artifact; use the launcher to supply its library dependency. Validate the module with `python /path/to/brandbuilder/templates/validate_glyph.py build/mk_paths.py --module`.

BrandBuilder 3.0.1 fixes `skill/templates/build_specimen.py` upstream to write its outlined SVG as UTF-8 without BOM and with LF line endings on Windows. The fix uses `open(..., newline="\n")` to retain Python 3.8 compatibility. It changes text serialization only; font outlines, logo geometry and social composition remain unchanged.

Owner review also identified that the specimen placed the mark above the display name. The generator now measures the mark and font outline bounds, aligns their vertical ink centers in the header, and places the specimen label above that row. Approved logo path data and generated logo derivatives remain unchanged.

BrandBuilder 3.0.1 also isolates screenshot temporary files per QC run, preventing concurrent kit builds from overwriting each other's desktop and mobile captures.
