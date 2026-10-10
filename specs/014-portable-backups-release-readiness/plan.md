# Implementation plan: Portable backups and release readiness

**Slice**: S014 | **Issues**: #14, #29 | **Date**: 2026-10-09

**Branch**: `codex/014-portable-backups-release-readiness`

## Technical context

Go shared runtime and CLI, Wails desktop, Node maintainer tooling, Python packaging, static Next.js documentation. Reuse receipt-backed catalog schema 9, streamed artifact stores, graph evidence rebuild and native relocation qualification. Defaults filesystem/SQLite/LadybugDB; S3/PostgreSQL/ArcadeDB have equal contracts. Version stays 0.0.0 until explicit release preparation.

## Constitution check

I: Complete authorized scope, no eligibility/review gates. II: Shared CLI contracts; advanced maintenance operations can precede GUI controls with explicit help. III: Current canonical authority, exact provenance/lineage and raw metadata survive; exclude transient and redundant bytes. IV: All supported backends, destination credentials separate, fenced writable authority. V: Meaningful targeted automated verification, no acoustic inference, complete required-CI turnaround below ten minutes. Design addresses all principles; actual turnaround remains a qualification requirement.

## Design and execution phases

1. Specify/clarify/checklist and research complete portability, native sources and release/tracking gaps.
2. Build catalog-native backup capture/portable restore, lifetime retention pins and write fencing. Add streamed directory bundle service and offline CLI operations, using workspace lock without starting processing workers. Graph checkpoints reset for evidence rebuild. Validate all source relations and bytes before target activation.
3. Build source-complete native media companions on each platform, preserving all supported built-in decoder/demuxer behavior and explicitly retaining external input capabilities. Package exact corresponding sources/build recipe and capability evidence. Add CLI-only variant and configured signing before hashes.
4. Add version preparation and release candidate/publication workflows; validate exact clean source revision, synchronized versions, all six native package variants, sources, hashes and signatures. Upload complete assets to draft, verify remote assets, publish, then allow configured matching documentation promotion.
5. Align canonical taxonomy and forms while preserving historical issue edits/states/project values. Audit final public contracts and add schema/help/glossary/changelog changes.
6. Analyze before implementation, run required local checks, publish the official PR to establish native/backend CI evidence, converge, satisfy all findings within two external review rounds and stop for owner merge.

## Code ownership and research strategy

The plan skill explicitly dispatches research agents. Backup agent owns internal/backup, catalog transfer/retention and focused tests. Package agent owns source builders/media pins/package variants/signing and packaging tests. Release agent owns release/version tooling/workflows, bootstrap/forms and focused tests. Root owns CLI/shared surface integration, schemas, public docs/changelog, Spec Kit artifacts, CI coordination, GitHub lifecycle and final convergence. Shared-file edits are coordinated explicitly.

## Constraints and decisions

No source snapshot JSON mutation bypasses admission receipts. Cross-profile restore admits relocation authority inside a validated transaction. Reference-only pins survive interruption and release explicitly. Completion manifest publishes last; private immutable digest-addressed files avoid general archive extraction. Existing local catalog export remains catalog-only and documented separately.

No upstream binary is called source-complete based on FFmpeg core sources alone. Replace incomplete upstream media distributions with owned exact source builds and explicit input-capability evidence. No arbitrary codec restriction is introduced.

No release version/tag/destination/signing identity is invented. Publication is implemented, tested and gated on owner invocation; S014 itself publishes only its branch and PR. Documentation promotion depends on verified published exact-source release, not workflow job order alone.

## Verification

Backup current-authority/lineage roundtrip, corruption/missing-byte/path rejection, no-overwrite, retention-release, stale transfer owners and interruption. Same authority checks on alternative backend fixtures and graph rebuild. Native real media/canonical operations and relocated package checks on all three operating systems and both variants. Release mismatch/failure/readback/signing tests. Canonical tracking history preservation. npm run check, npm test, full Go/vet, affected race tests, Python tests, frontend checks/build and site build. Exact final PR head CI and every external review thread verified.

## Measured timing disposition, 2026-10-10

Complete fresh source/bootstrap qualification passed in 29m04 after the exact source recipe invalidated compiled caches. All individual jobs met unchanged ten-minute deadlines. Independent dependency groups, verified format/FFmpeg stages, immediate complete-cache transfer and parallel qualification preserve all capabilities and acceptance checks. Complete cached PR qualification subsequently passed in 9m46 on the same recipe, with every runtime/platform/package/backend lane retained.

Invoke Constitution Governance's concrete, justified and reported exception for this measured S014 cold source bootstrap only. The complete cold turnaround remains a disclosed deviation from the original target; the cached full workflow meets it. This is not per-job reinterpretation, a product scope reduction, a waiver of future cached regressions or a blanket exception for future source recipes. Verification records the exact revisions, source/cache identity, cold and cached durations and retained checks. No attended approval stage is added to the autonomous workflow.
