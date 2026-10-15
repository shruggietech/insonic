# Tasks: Reliable CI and usable documentation

**Input**: S028 spec.md, plan.md, research.md, data-model.md and contracts/documentation.md.
**Scope**: #93–#96, candidate 1.0.1. #97 remains after merge.
**Tests**: Required by the specification. Add regression assertions before their corresponding implementation, demonstrating the old behavior where practicable. Existing lease failure is already demonstrated by main run 38633239324.

## Phase 1: Setup and foundation

- [x] T001 Establish codex/028-docs-quality-ci and specs/028-docs-quality-ci/ with the approved four-issue scope and explicit publication boundary.
- [x] T002 Complete spec.md, research.md, plan.md, data-model.md, contracts/documentation.md, quickstart.md and the two checklists using installed Spec Kit templates.
- [x] T003 Run the read-only Spec Kit consistency analysis of specs/028-docs-quality-ci/ and record coverage, exceptions and any required remediation in analysis.md before source implementation.
- [x] T004 Set GitHub Project items #93–#96 to S028 / In progress, leaving #97 unchanged.

**Checkpoint**: T003 must have no unresolved critical finding. The reviewer-owned documentation-quality checklist remains an honest unchecked review artifact; existing owner autopilot authorization permits implementation without a new approval gate.

## Phase 2: US1 Reliable bounded delivery (P1)

**Goal**: Trustworthy real-backend lease fencing and consistent 660-second CI bounds.
**Independent test**: Exercise the real SQLite security suite and elected PostgreSQL integration in CI, inspect active workflow/policy limits, retain all integrity gates.

- [x] T005 [P] [US1] Replace wall-clock sleeps in internal/catalog/security_test.go with guarded durable live/expired fixture states, preserving real ClaimLease/RenewLease exclusion, next-generation takeover and stale-generation refusal assertions.
- [x] T006 [US1] Align tests/ci-scope.test.mjs, tests/release-versions.test.mjs, tests/windows_package_parallel_test.py and tests/media_source_build_test.py with the approved 11-minute CI policy before changing their corresponding budget configuration.
- [x] T007 [US1] Apply 11-minute limits in .github/workflows/ci.yml, native-qualification.yml, media-source.yml and release.yml and the relevant outer Windows qualification bound in scripts/qualify-windows-package.py; preserve unrelated operation and compiler limits.
- [x] T008 [US1] Amend current policy in AGENTS.md, .specify/memory/constitution.md, CONTRIBUTING.md, docs/v1.0.1/development.md, docs/v1.0.1/product.md and docs/v1.0.1/references/CONTRIBUTING.md, with consistent active comments in scripts/media_source_build.py and truthful history.
- [x] T009 [US1] Run focused budget/Python/catalog tests through hidden tooling and record actual results in specs/028-docs-quality-ci/verification.md; run native installer/UI checks only on disposable hosted CI.

## Phase 3: US2 Durable current documentation entry (P1)

**Goal**: Manifest-elected latest content and explicit historical archives across public/offline exports.
**Independent test**: Synthetic latest election, fixed-version preservation and static no-script topic/index/fragment navigation.

- [x] T010 [P] [US2] Add routing/export regressions under site/tests/ covering manifest election, stable entries, nested references, removed topics, explicit versions, canonical identity and offline link forms before route implementation.
- [x] T011 [US2] Add latestPath and manifest helpers in site/lib/docs.mjs and implement full static current aliases under site/app/docs/, including /docs/, /docs/latest/, topical/reference aliases and /docs/versions/.
- [x] T012 [US2] Preserve explicit version routes and add honest missing-current-topic/archive fallbacks in site route generators and export/link checking helpers under site/scripts/; preserve query/fragment behavior using native navigation.
- [x] T013 [US2] Document stable current and archive routes in site/README.md and run site routing/export checks, including file index paths and disabled scripting, recording results in verification.md.

## Phase 4: US3 Readable and accessible documentation (P1)

**Goal**: A complete shared-template standards correction with concrete headless evidence.
**Independent test**: Public/offline exports at 320, 360/390, 768 and desktop widths, both themes, keyboard/disclosure and progressive fallback behavior.

- [x] T014 [P] [US3] Add presentation regressions under site/tests/ and bounded browser checks in scripts/check-docs-ui.mjs for disclosure semantics, shared article/reference structure, focus, reflow and fallback behavior before the associated presentation fixes.
- [x] T015 [US3] Implement a shared DocumentationFrame in site/components/ and use it in article/reference route JSX; preserve version context, active navigation, heading/contents structure, previous/next navigation and skip targets.
- [x] T016 [US3] Update site/app/layout.jsx, site/app/page.jsx and site/scripts/site-runtime.js for latestPath ordinary entry links, compact accessible header/navigation, native responsive disclosures and progressive theme behavior.
- [x] T017 [US3] Correct governed typography, spacing, targets, focus, themes, motion and 320–desktop reflow in site/app/styles.css without changing brand/kit/ bytes.
- [x] T018 [US3] Contain real tables, code and diagrams with labelled focusable local scroll regions in site/lib/docs.mjs, coordinated with the shared styles; keep archived Markdown content intact.
- [x] T019 [US3] Run scripts/check-docs-ui.mjs across all generated templates and representative complex content, including public/offline, no JavaScript, denied storage, keyboard, enlarged text, forced colors and reduced motion; save JSON and screenshots under build/s028-docs-ui/.
- [x] T020 [US3] Integrate the bounded browser regression check and evidence upload in .github/workflows/ci.yml, keeping the final workflow within 660 seconds and recording specific audit findings/corrections and unverified assistive-technology behavior in verification.md.

## Phase 5: US4 Useful company attribution (P2)

**Goal**: README endorsement links to the company website.
**Independent test**: Inspect rendered Markdown and destination during configured repository/site checks.

- [x] T021 [P] [US4] Link ShruggieTech in the final README.md endorsement to https://shruggie.tech/, preserving visible wording and routing ordinary docs links through the stable current entry where applicable.

## Phase 6: Integration, provenance and delivery

- [x] T022 Integrate all stories, update CHANGELOG.md and specs/028-docs-quality-ci/ evidence chronologically, inspect encoding/LF/brand-byte integrity and run npm run check, npm test plus affected site/product/Python/catalog checks from quickstart.md.
- [x] T023 Commit source implementation, build exact clean-source application capture inputs through hidden launchers and refresh affected governed application screenshots/receipts through scripts/capture-application-ui.mjs using only a fresh owned headless browser context; verify public/offline asset integrity and retain candidate version 1.0.1.
- [x] T024 Run Spec Kit convergence, document its result in convergence.md, complete tasks only with evidence, commit final artifacts, push codex/028-docs-quality-ci and open/attach the official PR closing #93–#96 with milestone 12 and Project In review.
- [ ] T025 Inspect all automatic first-round reviews/reactions, answer every comment and resolve each actionable thread with verified changes; optionally request exactly one second review round, never a third, recording rounds and final head in verification.md.
- [ ] T026 Verify elected CI checks green on the exact final PR head and elapsed turnaround against 660 seconds, verify all review threads satisfied, and ping the owner for final review/merge. Keep #97 after merge and perform no merge/tag/release/site publication.

## Dependencies and parallel execution

T001–T003 establish the implementation gate. T004 can follow that gate independently. Afterward US1, US2 and US3 proceed in parallel with disjoint file ownership, while the root handles US4, tracking and integration. US2 owns renderer helpers and route generation; US3 owns presentation frames/JSX/styles/runtime, coordinating the latestPath import and local overflow semantics. US1 owns workflows and integrates the browser command only after US3 supplies it. T022–T026 depend on all four story checkpoints; capture refresh follows a clean source commit, then exact-head checks follow the final commit.

## Implementation strategy and boundaries

All four stories are required for completion. No partial MVP is declared delivered. Changes to frozen historical source, governed kit bytes, production lease TTL policy or unrelated operation timeouts are excluded. Use hidden noninteractive tooling, direct Git/GitHub commands and exact owned-process cleanup. Every review comment receives an appropriate response, even where no code change is warranted. The owner performs final merge; release publication is #97 afterward.
