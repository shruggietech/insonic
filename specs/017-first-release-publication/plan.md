# S017 implementation plan

Branch: `codex/017-first-release-publication`. Date: 2026-10-11. [Specification](spec.md). Tracking: [issue #46](https://github.com/shruggietech/insonic/issues/46), milestone `v1.0.0 publication`, Project insonic.

## Summary and technical context

Finalize the existing 1.0.0 release metadata without another version preparation or product change. Reuse Python release/Pages tooling, Node 22 repository checks, Next.js public/offline exports and existing native Go/package qualification for Windows x86-64, Linux x86-64 and macOS ARM64. All configured acceptance and integrity checks remain. Jobs retain existing bounds/caches; no new release architecture is needed.

## Constitution gates

Pre-research and post-design: PASS. Spec Kit and primary GitHub tracking are present; software/docs/schema remain synchronized; the first-major glossary rescan is included; public authority stays self-contained; six variants and complete corresponding sources are required; explicit publication authorization exists. No publisher-signing or acoustic-accuracy gate is added. Existing platform loader integrity remains. No constitution exception is required.

## Chronological execution

1. Specify and clarify the S017 outcome; research the existing guarded launch path and analyze requirement/task coverage.
2. Consolidate `CHANGELOG.md` and `docs/v1.0.0/references/CHANGELOG.md` with the actual release date; rescan `docs/v1.0.0/glossary.md` and public terminology. Author concise `highlights.md` within this internal slice.
3. Run repository, existing release/Pages tests and ordinary/release-target site checks; converge the prepared source against the specification. Push a preparation PR, satisfy CI and all review comments, then squash merge and delete the branch under owner authorization.
4. Select the clean final protected-main SHA, wait for its external checks, qualify an exact-source publication-disabled candidate, create/push `v1.0.0` at that SHA, then dispatch the existing publication workflow with Pages promotion and highlights.
5. Verify the public release and original assets plus live documentation and the exact Pages deployment. Close issue #46/milestone and clean temporary launch material. Record actual launch results on the issue, rather than committing another archive of logs/receipts.

## Touch points and implementation boundary

Only master/frozen changelogs, any demonstrated glossary correction and compact `specs/017-first-release-publication/` planning/highlights/runbook need source edits. Existing interfaces are documented in [data-model.md](data-model.md); no new API contract is introduced. Existing guards in `scripts/release.py`, `scripts/pages.py`, `.github/workflows/release.yml` and `site/lib/release-status.mjs` remain unchanged unless a demonstrated launch defect requires a proportional fix. Ordinary source builds retain candidate status; elected immutable release artifacts render released status.

Repository task checkboxes describe source preparation and executable launch instructions. Actual publication acceptance is tracked separately on issue #46 and cannot be claimed by a preparation merge alone. Scratch files/downloads stay in one external temporary directory and are removed at completion. Useful native caches remain.

## Preparation verification

2026-10-11: Repository integrity/schema checks passed; all 83 Node and 137 existing Python tests passed. Ordinary and elected release-target public/offline builds each validated 52 pages and their links/anchors. The terminology rescan covered 23 current documentation pages and 157 existing glossary entries; added precise ad-hoc-signature and Pages CNAME-file definitions with primary learning/local usage links. Master/frozen changelogs agree, software/docs/schema remain 1.0.0 and highlights satisfy the 180-word/link contract. Spec Kit analysis found zero conflicts or uncovered requirements. Source convergence found no unbuilt source preparation; actual release/deployment remains the separate live acceptance on issue #46.
