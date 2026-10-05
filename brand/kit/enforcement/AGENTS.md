<!-- BEGIN SHRUGGIE-BRANDBUILDER: CONSUMER CONTRACT -->

## Governed BrandBuilder contract

BrandBuilder is mandatory for brand-system authoring, consumer implementation, and conformance audit. This kit pins Brand Canon `2.0.0`, Interface Canon `1.0.1`, component recipes `1.1.0`, Web/React adapter `1.1.0`, egui adapter `1.0.2`, WordPress adapter `1.0.0`, compiler `3.0.1`, and brand `1.0.0`.

Read `consumer-contract.json`, then `IMPLEMENTATION.md`. The pinned contract outranks screenshots, legacy stylesheets, and inferred local values. Do not reinterpret identity or create a permanent parallel design system.

Affiliation boundary: this is a ShruggieTech-owned child brand with declared parent `ShruggieTech`.

If BrandBuilder `3.0.1` is absent, verify SHA-256 `bef69601c58d6a6575aa5c45c8908069c1fb3b99320c2c6bf6548d303ae7d682` and extract `enforcement/distributions/shruggie-brandbuilder-3.0.1.skill` into the empty directory `enforcement/brandbuilder`. Never substitute another version. Run `python3 enforcement/brandbuilder/templates/verify.py .` and `python3 enforcement/brandbuilder/templates/validate_glyph.py brand.json`; both must report zero failures.
<!-- END SHRUGGIE-BRANDBUILDER: CONSUMER CONTRACT -->
