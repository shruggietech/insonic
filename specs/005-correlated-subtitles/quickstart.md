# S005 validation guide

1. From a clean checkout, run repository/schema and maintainer tests, then full Go/vet/affected race tests. None may initialize inference models.
2. Qualify exact current Cueson and pinned media tools on supported native platforms. Import both committed fixtures with supplied reference subtitles and deterministic speaker turns.
3. Inspect current embedded document, external mappings and measured source timing. Export Cue JSON and native subtitles; strict native export refuses annotation loss before writing output.
4. Rerun diarization alone, correct known mapping and verify text/source digests, one current result, invalidated references and physical cleanup. Interrupt/cancel/retry and test snapshot rollback/workspace isolation over catalog/storage combinations.
5. Explicitly elect the separate maintainer inference command with exact local environment/model configuration. Record actual engine outputs, resource use and limits; CI and merge status never depend on this command.
6. After local convergence, publish authorized PR, resolve every comment, issue at most one second-round review request, then verify final checks and stop for owner merge.
