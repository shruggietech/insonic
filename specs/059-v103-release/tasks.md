# Tasks: v1.0.3 patch release preparation

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md).
**Tests**: Required meaningful preparation/history regressions, plus maintained complete qualification.

## Phase 1: Setup

- [x] T001 Establish matching S059 branch, selector, specification/clarification and built-in quality in specs/059-v103-release/.
- [x] T002 Link issue #196, milestone 43 and Project 7 to specs/059-v103-release/spec.md.

## Phase 2: Foundation

- [x] T003 Reproduce frozen-history migration and duplicate final-highlight-link failures through actual preparation regressions in tests/release_test.py (FR-002, FR-003, FR-004).
- [x] T004 Correct scripts/release.py frozen historical test/fixture handling, owned download tag migration and idempotent final link; use dynamic current-version negatives while preserving explicit history in tests/release_history_test.py (FR-002, FR-003, FR-004).

## Phase 3: US1 - Coherent patch candidate

Independent test: all current owned authorities become 1.0.3, candidate status remains explicit, highlights contain one final tagged changelog link and upstream identities do not change.

- [x] T005 [US1] Finalize specs/059-v103-release/highlights.md, record proportional tooling decisions in CHANGELOG.md, and run scripts/release.py prepare for 1.0.3 dated 2026-10-29 (FR-001, FR-002, FR-004; SC-001).
- [x] T006 [US1] Validate current VERSION/manifests/runtime/schema/fixture bindings and independent tool/model identities after maintained migration (FR-001, FR-004; SC-001).

## Phase 4: US2 - Historical authority

Independent test: original versioned docs/schemas and frozen receipt fixtures retain their exact authority; current and unknown receipts still require complete modern evidence.

- [x] T007 [US2] After copying corrections forward, restore the four demonstrated changed docs/v1.0.2 pages from exact published tag authority, mark verified 1.0.2 released, and update README.md, CONTRIBUTING.md, docs/README.md and site/README.md current entry prose (FR-003, FR-006; SC-002).
- [x] T008 [US2] Run historical/preparation/schema regressions and verify historical schema/receipt bytes and narrow original exemptions in tests/release_history_test.py and tests/recovery_release_history_test.py (FR-003, FR-004; SC-001, SC-002).

## Phase 5: US3 - Qualified owner handoff

Independent test: genuine current-version image authority and exact final-source platform checks are accepted; all findings are resolved with an unmerged PR.

- [x] T009 [US3] Temporarily omit only current screenshot sections in docs/v1.0.3, commit/push clean capture bootstrap and dispatch .github/workflows/ci.yml on its exact branch SHA (FR-005, FR-006, FR-007; SC-002, SC-003).
- [x] T010 [US3] Independently verify original capture artifact digests/source authority, import original enumerated PNGs and four receipts into docs/screenshots/v1.0.3 and restore current documentation embeds (FR-006; SC-002).
- [x] T011 [US3] Complete root check/test and site check/test/public/offline builds, source/version/hash image validation, visual inspection and placeholder/encoding scan (FR-001, FR-002, FR-006; SC-001, SC-002).
- [x] T012 [US3] Complete full Python and affected Go/vet checks without weakening history, security, substantive assertions or product deadlines (FR-003, FR-004, FR-007; SC-001, SC-003).
- [x] T013 [US3] Finalize executable post-merge exact-main candidate/CPU/restore/retained-byte publication/public-verification guide in specs/059-v103-release/quickstart.md (FR-009; SC-004).

## Phase 6: Convergence and delivery

- [x] T014 Converge actual changes against every requirement, scenario, task and constitution principle in specs/059-v103-release/ (FR-001 through FR-010; SC-001 through SC-004).
- [x] T015 Commit/push completed source and genuine captures, open the official ready PR, attach it and update #196/Project tracking (FR-008; SC-004).
- [x] T016 Monitor/fix final exact-source CI and retain independently audited platform/package/image/receipt identities and preparation/qualification/combined timing outside workflow expiry before any full rerun (FR-005, FR-007, FR-008; SC-003).
- [x] T017 Resolve every actionable review and individual thread within the initial automatic Codex round plus at most one manual second round; retain receipts (FR-008; SC-004).
- [x] T018 Finalize concise verification.md and tracking, verify clean final head and owned transient cleanup, and hand off the open unmerged PR for owner final review (FR-008, FR-009; SC-004).

## Dependencies and execution

T001 -> T002 -> T003 -> T004 -> US1 -> US2 -> capture bootstrap -> original captures -> final local/platform qualification -> convergence/delivery. The maintained guide and independent read-only checks can proceed while hosted CI runs. Shared source edits are serial.

## Parallel examples

Independent Python/root/site checks can run after prepared source readiness. The plan-directed research agent is read-only; no implementation agents edit shared files. Capture artifacts and local headless checks have separate owned lifetimes.

## Implementation strategy

Use maintained preparation after failing-first proportional repairs, then preserve historical authority, then genuine hosted captures and final qualified delivery. The first clean bootstrap is an evidence producer, not an official release. All twenty tasks belong to this PR phase; later merge/publication operations are documented continuation only.

## Phase 7: Convergence

- [x] T019 Repair redundant Windows private-directory permission normalization in internal/workspace/private_windows.go with fresh exact owner/ACL validation; add deterministic no-write and changed-ACL normalization/error regressions in internal/workspace/private_windows_test.go, then revalidate concurrent startup and all Go packages under unchanged clocks (FR-005, FR-007, FR-010; Constitution IV/V; complete). The successful full-source revalidation unblocks the T009 capture push.

## Phase 8: Convergence

- [x] T020 Apply maintained request headroom to non-expiry acquisition fixtures in internal/library/acquisition_test.go, retain intentionally injected 100 ms timeout clocks, and prove delayed truncated-response classification plus persisted transcript correction/authority through three focused repetitions (FR-007; Constitution V; complete). Final hosted full-source acceptance remains T016.
