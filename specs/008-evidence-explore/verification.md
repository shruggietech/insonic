# S008 verification

Date: 2026-10-08. Owner slice S008, branch `codex/008-evidence-explore`, directory `specs/008-evidence-explore`, issues #11/#12. Base: dc43f00cac915675d570196fc0f25bc641c7a15e. Version remains 0.0.0.

## Spec Kit

Specify, clarify, requirements/evidence checklists, research, plan, data model, contracts, tasks and blocking analyze completed. Research-only agents examined existing graph/catalog/Explore integration; implementation remained sequential. No unresolved constitution or scope conflict.

Convergence assessed all 16 FRs, five SCs, ten story acceptance scenarios, ten plan decisions and five constitution principles. Appended recovery, persistence, delivery, elected-HTTP and disjoint-source/rebuild acceptance tasks (T035-T039). All buildable requirements now have implementation and acceptance coverage. Hosted platform timing/exact-head checks and PR review remain publication gates, not implementation claims.

Task path consolidation: graph/evidence/query/timeline/saved dispatch live in `internal/app/explore_*.go`; common query/calendar hydration lives in `internal/explore/`. Native and backend tests are `internal/app/explore_*_test.go`, `internal/graph/adapter_test.go` and tagged engine tests. The task descriptions' conceptual file names map to these final paths.

## Local verification

- Repository integrity/schema: `npm run check`, 19 schemas/17 contracts/21 examples, UTF-8/LF/link/brand/version checks passed.
- Repository Node tests: 56 passed.
- Full Go acceptance and vet passed; affected catalog/app/explore/evidence/graph race checks passed (catalog about 60 seconds, app about 100 seconds). The final formatting and legacy snapshot-proof correction also passed full Go acceptance and vet.
- Frontend typecheck and 43 rendered/pure tests passed. Covers whole-calendar continuation/keyboard controls, current source playback fences, late-ticket closure, query generation fences, immutable saved versions/CAS, separate layouts and graph/table expansion.
- Python archive, loader, worker and package tests passed. Required scripts enforce no recognition/diarization inference, initialization or weights.
- Public and offline site exports passed: 27 documentation pages, relative links/anchors and versioned JSON reference.
- Real LadybugDB 0.21.2 and ArcadeDB 26.9.1: ordered publication, exact replay, mismatched receipt, stale fence, predecessor gaps, correction/invalidation, workspace binding, uncertain-commit reconciliation, failed atomic rebuild and restored-lineage reset passed.
- Shared real-engine app matrix: all seven normalized operations match catalog reference identities/order/values; current document replacement removes old cue references and rebuild succeeds.
- Real PostgreSQL fixture: saved immutable versions, compatibility replay and separate layouts survive restore with proofs/CAS; forged definitions fail.
- Elected HTTP-to-durable extraction test uses deterministic structured output and validates negation/conditions. Literal extraction retains current cue ownership; replacement invalidates extraction.
- Prepared native dependency checks and source CLI/native desktop qualification passed. The mounted native UI journey includes Explore calendar, current query, saved definition, force graph/table and separate layout.
- Relocated Windows x86-64 package passed real licensed audio/video import, Cue JSON assembly/native export, calendar, embedded graph query/version/rebuild, GUI bridge and native webview under an isolated PATH with checkout Ladybug libraries moved aside and restored. Package smoke took 101 seconds. This local receipt is a dirty working-tree qualification; hosted exact-head clean package receipts are still required.

## Decisions and limitations

Reference graphs contain identities/relationships, not transcript or assignment copies. Shared hydration preserves current text and exact source clocks. Disjoint assertion citations retain individual spans; filters never invent evidence in the gaps. Bounded portable catalog fallback reports its basis/graph error if projection is unavailable or exceeds limits.

Saved versions carry immutable backend/schema compatibility findings. Native validation checks the selected dialect and backend EXPLAIN when reachable; unreachable checks stay pending, and incompatible definitions remain inspectable. Save replay preserves the original compatibility record.

A Windows package regression exposed startup-time DLL resolution and ambient OpenSSL dependence. Graph DLLs now reside beside executables, with checksum-pinned conda-forge OpenSSL 3.5.9 ABI-3 companions and their Apache 2.0 notice. The pinned zstandard archive decoder is installed in the workspace tool directory. This is runtime packaging, with no model initialization.

No official release, signing/notarization, published binary archive, third bot round or owner merge is authorized/performed. Existing companion corresponding-source release work remains #14. Untagged builds explicitly report embedded graph unavailable; prepared packages include it. Arbitrary native functions are not promised portable.

## Publication and review

Official PR #27: https://github.com/shruggietech/insonic/pull/27, published 2026-10-08 from 0252d3b96523d1d8ba87669424c128210d2646ec. Attached to the owner chat; issues #11/#12 moved to In review. Initial automatic Codex review started on that head. At most one explicitly requested second round will be tracked here. Review findings take priority over waiting for CI. Final handoff requires every review replied/resolved and exact-head green checks below ten minutes per hosted job.

### Round 1 findings and corrections

Initial automatic code review completed on 0252d3b with four findings. Removed duplicate graph-maintenance startup from the one-second recovery tick. Separated immutable projection snapshots (128 MiB) from work inputs (8 MiB), with consistent paged backend reads and automatic smaller-page retry for transport limits. Routed Arcade native results through the shared 500-row/512 KiB check. The desktop retains complete saved definitions and changes only explicitly edited controls, preserving multi-ID/concept/entity/date filters, ordering and traversal direction.

Focused regressions cover three actual recovery ticks and joined shutdown, over-8-MiB immutable snapshot replay/claim/restore, paged reads/backoff, both native output routes, and CLI-authored advanced query run/save/edit. Full Go acceptance/vet, real Ladybug/Arcade graph and app matrices, 42 frontend tests/typecheck, repository integrity/schema and documentation/offline export pass. All four threads were replied to and resolved after correction commit 1d8b1c4 was pushed. Focused regression race checks also pass. The second and final round was requested once in issue comment 6056595728, asking for code and security review on the corrected head. No third round will be triggered.
### Round 2 result

The second manual code review completed on 1d8b1c4 with one comment about comment-separated native function calls. The reviewed head already contains comment-aware lookahead in native_space.go, so the reported bypass does not reproduce. Two older SQL test cases used the Cypher dialect and could reject for the wrong reason; they now use SQL. Added 54 disallowed-call cases across all three dialects, block/line/chained comments and quoted/unquoted identifiers, proving rejection before engine access. Eighteen corresponding allowed COUNT cases and malformed comments also pass. Graph acceptance/race checks pass. No third round is requested. No independent security-review result was returned by the connector for the combined second-round request.

### Hosted workflow correction

The first hosted runs failed before creating jobs because the Windows decoder command used an unquoted colon/space in a YAML plain scalar. Converted it to a block scalar. Repository integrity now parses project YAML with checksum-locked yaml 2.9.1, and a regression catches that exact command plus duplicate keys. This adds syntax integrity without relaxing or extending the ten-minute job budgets. Reviews are complete; hosted exact-head acceptance remains pending.

Hosted adapter acceptance exposed a historical test setup still retaining schema-6 tables while recreating schema 3. The fixture now drops graph_layout, saved_query and current_extraction before their referenced older tables. Full local PostgreSQL integration acceptance (including populated historical migration, proof forgery, saved/layout restore and all catalog tests) passes in 26 seconds. Production migration behavior and integrity assertions are unchanged.

The macOS mounted journey reached video playback and then the whole-window deadline, without a stage failure response. Media playback now scrolls the element into the viewport and bounds the real play() promise separately; it still requires decoded time advancement. A never-settling-promise regression ensures diagnostics return before the global window deadline. Hosted macOS requalification is required to establish the result.


Windows hosted relocated-package acceptance passed (9m56s) on ba3b878. Linux exhausted its ten-minute budget during Azure HTTP mirror downloads before any product checks ran. The Linux dependency step preserves every declared package and APT signature verification, switches Azure Ubuntu sources to the official HTTPS archive, bounds connection waits/retries and omits unrelated recommended packages. Hosted checks must rerun on the correction head; no job budget was raised.


The next adapter run passed catalog migration but timed out in ten-second Arcade database creation while Go packages shared the fixture concurrently. Fixture packages now run serially and both administrative setup clients have a finite 30-second timeout; native query limits remain unchanged. Linux passed dependency preparation with the official HTTPS mirror and advanced into native packaging. macOS again timed out with its hidden WKWebView, so qualification alone starts an onscreen window on GitHub-hosted macOS, retaining hidden local/Windows smoke runs and ordinary application behavior. Platform-policy regression tests cover those boundaries. Hosted requalification remains required.

On 7e5e456, complete hosted Linux native/package acceptance passed in 6m21s and Windows passed in 9m58s. Both remained below ten minutes. The latest serial real Arcade graph/app checks pass locally, as do desktop policy tests and repository checks. The remaining correction head requires hosted fixture and macOS acceptance before owner handoff.
On 258304e, hosted adapter fixtures passed in 2m08s, Linux native/package qualification in 6m32s and macOS native/package qualification in 5m22s. The hosted-only visible macOS qualification establishes both source and relocated mounted Explore journeys. Windows exhausted the job budget during package inventory after core acceptance took 3m08s and Go setup/cache restore took 1m21s. Windows core acceptance now runs as an independent parallel job with the identical Go acceptance/vet and Python suite. Native qualification retains every original engine, CLI, credential, UI and relocated-package check; both Windows jobs retain ten-minute limits. Final exact-head checks remain required.
