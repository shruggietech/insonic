# Verification evidence

**Slice**: S014 | **Date**: 2026-10-09 | **Status**: local implementation and Windows native/package checks passed; all-platform/backend CI and external review pending.

## Implemented behavior

Self-contained and explicit reference-only bundles preserve receipt-backed current catalog authority, exact artifact bytes and durable trained lineage. Empty-target restore relocates publications, resets graph authority and retains independently configured destination credentials. Durable activation proofs and private multipart journals allow exact retries without making failed restores writable. Original local acquisition inputs are excluded through newly admitted portable proofs, without modifying the source catalog.

Native CLI-only and desktop variants contain verified corresponding sources and build inputs, with configured signing before final hashes. Release tools prepare synchronized versions, assemble the complete six-package/documentation candidate and verify draft publication/readback before configured documentation promotion. Opened-only issue classification and bootstrap reconciliation preserve historical issue state and edits. Pending dependency changes from PRs #17/#23 are included.

## Local checks

| Check | Evidence |
| --- | --- |
| Repository integrity/schema | `npm run check` passed: UTF-8/LF, links, JSON/YAML, governed brand bytes, 23 schemas, 21 contracts and 28 examples. |
| Node tests | `npm test` passed, 73 tests including contract, taxonomy, historical-state, version, independent promotion and exact Windows toolchain checks. |
| Go | Full `go test ./...` and `go vet ./...` passed; affected backup/catalog/CLI suites and vet passed again after recovery and manifest-binding changes. |
| Race checks | Backup, catalog, secrets and CLI passed; backup/catalog repeated after durable activation changes, with no detected races. |
| Python | Repository discovery passed, 57 tests; current packaging, source, transfer, release and hidden-child tests include actual prepared-release schema validation, read-only cache readiness, strict private-prefix alias normalization, complete-cache tool exemption, deployment preflight and Windows credential restart. |
| Frontend | Typecheck, 73 rendered tests and production build passed. |
| Documentation | Site/offline export passed, 27 checked documentation pages with valid relative links/anchors. |
| Native graph | Actual Windows LadybugDB accepted-event replay and current-evidence restore/rebuild passed. |
| Source build | Initial Windows recipe cold build passed in 504.391 seconds: dependency stage 316.938 seconds, FFmpeg stage 187.407 seconds, 13 verified media source archives and complete static-library/notices closure. Full warm-cache verification passed in 0.140 seconds. CI fixes changed the recipe identity; corrected all-platform source qualification remains pending. |
| Native media | Actual Windows library/app media qualification passed in 49.171 seconds, including stereo/surround, integer precision, multitrack offsets, MP3 and nonzero MP4/Matroska clocks. |
| Relocated packages | Both Windows variants passed in 233.891 seconds total. CLI smoke took 53.781 seconds; desktop smoke took 110.515 seconds, including native webview. Receipts verify inventory, path-with-spaces relocation, real imports, graph/roster/replacement, Cue assembly/export and isolated development libraries. Local receipts truthfully record uncommitted source; clean exact-head CI is still required. |
| Source transfer | Actual isolated complete transfer roundtrip passed in 18.187 seconds, 43 files and 13 source notice trees. Original native/source directories and caches were preserved. |
| Query budget fixture | The observed main-branch Windows failure came from first-open native graph initialization consuming the query fixture budget. The fixture now initializes the graph before the measured query phases; five repetitions passed. |
| Tracking | Live `npm run github:check` reported complete configuration; no bootstrap mutation was needed. Open issues remain #14/#29, with no new issue since scope assessment. |

Backup fixtures cover source-store loss, exact large integer preservation, current Cue JSON/mappings/rosters/durable trained outputs, obsolete-byte exclusion, original-input removal, corruption/missing bytes, traversal, occupied targets, generation fencing, retention release, failed configuration activation, successive restore and recomputed-manifest publication tampering.

Required adapter CI includes the SQLite/PostgreSQL and filesystem/S3 matrix, actual ArcadeDB replay/rebuild and an interrupted 10.5 MiB multipart restore that must reuse the original upload. Native CI qualifies both variants on all three platforms and retains receipts. Those live results remain pending at this record revision.

## Publication and review boundary

Official [PR #40](https://github.com/shruggietech/insonic/pull/40) is published. Initial automatic review completed with three findings: originating-path semantics, publisher check-read scope and missing deployment configuration. Fixes and targeted tests are implemented; individual responses/resolutions follow the correction push. At most one explicit second review request is allowed. Every finding requires a response and resolution; exact final-head CI must pass before owner handoff.

The [initial CI run](https://github.com/shruggietech/insonic/actions/runs/37999665623) passed foundation/docs and Go core checks but failed Unix dependency closure, Windows source prerequisites and Python release-schema fixtures. Unix private-prefix aliases are now completely validated before materialization to regular files; wrong or incomplete Windows toolchains fail before cold builds, and every Windows workflow selects the setup action's actual installation. Core lanes install the Node/schema dependencies required by release fixtures. Initial failed-run timing does not establish successful CI turnaround.

An independent retention audit also fixed manual-reference collisions, backup lifecycle ID reservation, native ownership proof recognition and atomic rejection of partial release. Cross-platform source-root fixtures and the affected backup/catalog/CLI suites and race checks passed after these changes.

Actual publisher signing/notarization, product tag/release publication, documentation deployment and acoustic inference are not executed by this slice. Signing and publication failures/readback are qualified with controlled fixtures. Version remains 0.0.0; no shipped-product or acoustic-quality claim is made.
