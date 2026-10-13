# Tasks: Package dependency integrity

**Input**: spec.md, plan.md, research.md, data-model.md, quickstart.md. Tests are explicitly required by FR-005/009. Contracts are skipped because changes are internal build qualification.

## Phase 1: Setup

- [x] T001 Complete requirements/design review and link #88/#86 to S024 in specs/024-package-dependency-integrity/checklists/dependencies.md and GitHub Project 7.

## Phase 2: Foundational

- [x] T002 Run the read-only consistency gate and record requirement/task mapping in specs/024-package-dependency-integrity/analysis.md before application edits.

## Phase 3: US1 truthful shipped notices (P1, MVP)

Goal: exact linked variant closure and matching upstream evidence. Independent test: controlled binary/source fixtures reject wrong identity and false inventory membership; actual package assemblies match binary union.

- [x] T003 [US1] Write failing missing Dir, source identity/checksum, replacement/shared/conflict, missing/empty notice, no-go.mod and collision fixtures in tests/go_module_notices_test.py.
- [x] T004 [US1] Implement selected-binary metadata union and selective lookup in scripts/go_module_notices.py; require elected compiler, correct command, versioned effective identity, exact checksum, nonempty absolute existing canonical cache Dir and nonempty upstream license files; reject local replacements and conflicts.
- [x] T005 [US1] Integrate byte-checked notices for CLI versus desktop in scripts/package-desktop.py; retain {module, version, notices} shape and compiler notice, excluding unrelated tooling/downloads.
- [x] T006 [US1] Write and pass independent membership/tampered notice reference fixtures in tests/go_module_notices_test.py and integrate current archive checks in scripts/package-desktop.py; add elected setup-go to .github/workflows/ci.yml candidate metadata reader without requiring platform source caches or executing binaries.

## Phase 4: US2 elected dependency remediation (P1)

Goal: captured x/net and patched docs dependency closure. Independent test: exact lock/Go election, inheritedtrust regression, clean advisory results and package notices.

- [x] T007 [P] [US2] Write a failing inherited trust options regression plus explicit own trust control in site/scripts/katex.test.mjs and elect it in site/scripts/run.mjs.
- [x] T008 [US2] Import captured Dependabot #87 commit576ef85aacbf1bcac1682ec898e56f29624d9869 into go.mod/go.sum, preserving upstream authorship and all other pins.
- [x] T009 [US2] Elect Mermaid12.2.0 and resolve KaTeX0.18.2 in site/package.json/site/package-lock.json; clean install, run patched-options test and audit, verify frontend notice closure includes elected versions.

## Phase 5: US3 portable docs and delivery (P2)

Goal: real SVG/theme compatibility, retained URLs/resources and completed bounded review. Independent test: browser public/offline qualification, historical Pages preparation and exact-head native/candidate CI.

- [x] T010 [US3] Implement hidden bounded Chrome/Chromium/Edge qualification of actual exported diagram sources and theme rerender in scripts/qualify-docs-renderer.py and wire .github/workflows/ci.yml docs job; fail on missing browser/errors rather than skip.
- [x] T011 [US3] Update current notice/qualification contracts in docs/v1.0.1/releases.md and docs/v1.0.1/development.md; append chronological dependency/notice changes in CHANGELOG.md while preserving frozen1.0.0/version/native pins.
- [x] T012 [US3] Run root/Go/Python/frontend/site checks and public/offline browser qualification; record results and advisory/notice evidence in specs/024-package-dependency-integrity/verification.md.
- [ ] T013 [US3] Commit clean source and qualify release-target documentation plus read-only historical Pages preparation, recording preserved paths in specs/024-package-dependency-integrity/verification.md; do not deploy or infer model accuracy.
- [ ] T014 [US3] Push branch and publish/attach official PR with #86/#88 closing links, #87 captured mapping and milestone/Project7 In review; preserve source PR87 until owner merge. Record URL/round ledger in specs/024-package-dependency-integrity/verification.md.
- [ ] T015 [US3] Resolve every comment/thread in at most initial plus one requested second round, recording concrete fixes and final review state in specs/024-package-dependency-integrity/verification.md.
- [ ] T016 [US3] Verify full final-head green Windows/Linux native/backend four-package and candidate CI from one revision/attempt; record exact CI identity in specs/024-package-dependency-integrity/verification.md and stop for owner final review/merge.

## Phase 6: Polish

- [ ] T017 Converge all requirements/tasks and verify clean UTF8/LF/no-mojibake checkout and no frozen/version/pin changes in specs/024-package-dependency-integrity/convergence.md; publish any final evidence commit and wait its exact-head CI/review state before handoff.

## Dependencies and execution order

T001 -> T002 -> US1(T003 -> T004 -> T005 -> T006), US2(T007 -> T008 -> T009) -> US3(T010 -> T011 -> T012 -> T013 -> T014 -> T015 -> T016) -> T017. US1 and US2 share no source until package verification; they can be developed independently after the gate. T007 can run alongside US1 fixtures. Within US3 browser and docs edits touch separate files, but validation waits for both dependency and notice changes.

Parallel examples: US1 membership and source negative fixtures can be grouped in one isolated test run; US2 KaTeX regression and Go election checks are independent; US3 frontend/root/Go checks can run concurrently with dedicated output logs. Mutations and dependencies remain sequential.

Implementation strategy: validate notice MVP first, validate patched dependency behavior next, then portable docs/combined qualification and authorized publication. Do not stop at the MVP or partial CI. All17 tasks use IDs and exact artifact paths; US1 has4 tasks, US2 has3, US3 has7, setup/foundation/polish have3.
