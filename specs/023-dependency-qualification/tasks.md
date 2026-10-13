# Tasks: S023 consolidated dependency qualification

**Input**: [spec.md](spec.md), [plan.md](plan.md), research/data-model/quickstart and requirements checklists. Tracking: [#84](https://github.com/shruggietech/insonic/issues/84).

## Phase 1: Setup

- [x] T001 Establish literal S023 branch/directory/issue mapping, captured elections and Spec Kit artifacts in specs/023-dependency-qualification/ (FR-001/009/010).

## Phase 2: Foundation

- [x] T002 Record and inspect all seven captured source commits and review threads in specs/023-dependency-qualification/research.md; preserve inputs after later bot rebases (FR-001/002/009).

## Phase 3: US1 delivery automation

Independent test: exact pinned action inputs preserve archived artifact names/extraction, strict digest failure, cache verification and candidate source/attempt/variant completeness.

- [x] T003 [US1] Add negative artifact-workflow policy tests first in tests/check.test.mjs for mutable artifact pins, archive false/omission, weak digest policy, skipped extraction and missing attempt fencing (FR-002/003/004).
- [x] T004 [US1] Import the five captured action updates into .github/workflows/ci.yml, media-source.yml, native-qualification.yml, release.yml and issue-classification.yml; keep exact requested action SHAs and accurate annotations (FR-001/002).
- [x] T005 [US1] Declare archived uploads, extracted strict-digest downloads and bounded artifact workflow policy validation in scripts/check.mjs and all affected workflows; retain source/compiler/recipe/cache/candidate verification and existing timeout limits (FR-003/004/008).
- [x] T006 [US1] Correct the four-variant workflow label and run artifact/source/cache/candidate negative acceptance in .github/workflows/release.yml and existing tests/media_transfer_test.py, media_source_build_test.py, release_test.py (FR-003/004/007/008).

## Phase 4: US2 runtime libraries

Independent test: real pipe/file terminal detection, ordinary runtime/credential acceptance and live/native backend paths pass with the exact five-module closure.

- [x] T007 [US2] Add a real pipe/file descriptor terminal regression in cmd/insonic/work_wait_test.go, retaining existing progress/machine/quiet/cancellation fixtures (FR-005/008).
- [x] T008 [US2] Import the captured runtime-group update into go.mod/go.sum, retaining Go 1.27.1, sys 0.49 and native pins; text 0.43/sync 0.24 are required closure (FR-001/005/007).
- [x] T009 [US2] Run Go tests/vet and credential/source/notice checks through existing internal/secrets tests and scripts/package-desktop.py qualification; retain no-inference boundaries (FR-005/007/008).

## Phase 5: US3 renderer

Independent test: locked clean installation exports complete public/offline documentation, historical resources and truthful elected candidate/release identity.

- [x] T010 [US3] Import exact renderer election into site/package.json and site/package-lock.json and install with npm ci --ignore-scripts (FR-001/006/007).
- [x] T011 [US3] Qualify public/offline/release-target exports and read-only historical Pages retention using site build/check, tests/site-release.test.mjs and tests/pages_test.py without publishing (FR-006/007/008).
- [x] T012 [US3] Clarify source/release-target status in site/README.md and current four-package dependency guidance in docs/v1.0.1/development.md; keep frozen 1.0.0 bytes intact (FR-006/007).

## Phase 6: Verification and publication

- [x] T013 Append dependency/process decisions to CHANGELOG.md and inspect all owned requested-version references; run full local repository/product/frontend/site/Python checks and converge requirements in specs/023-dependency-qualification/verification.md (FR-007/008/010).
- [x] T014 Commit/push and open the official consolidation PR, linking #84 and originals #77-#83; update GitHub tracking, respond/resolve every finding in at most two rounds, verify exact final-head full CI and stop for owner merge (FR-009/010).

## Dependencies and implementation strategy

T001 -> T002 -> US1 -> US2 -> US3 -> T013/T014. Shared workflow changes occur sequentially. Each story has independent fixtures, but the authorized done-gate includes all seven updates and full qualification, not an isolated MVP. Tests precede their corresponding implementation. Original bot PRs remain open until consolidation merge.

Parallel validation examples: repository checks/tests, runtime tests/vet, frontend tests and site export can run independently through verified hidden foreground child launches. Existing source/cache/candidate negative tests cover original-source identities, archive digests, compiler/recipe receipts and complete source/attempt matrices. Required CI includes actual Windows/Linux packages and backend fixtures; local test passes alone do not constitute qualification.

## Phase 7: Convergence

- [x] T015 Retain historical flat Next segment URLs through exact nested counterparts beside the same verified page in scripts/pages.py, with positive and missing-page/segment negative fixtures in tests/pages_test.py (FR-006, US3/AC2; partial gap discovered against live Pages).

## Phase 8: Review and qualification corrections

- [x] T016 Normalize artifact action owner/repository case and cover mixed-case mutable refs, archive, digest, extraction and attempt checks in scripts/check.mjs and tests/check.test.mjs (FR-002/003; second-round finding).
- [x] T017 Preserve controlled timed-out compiler/test output after process-tree termination in scripts/qualify.py, with real timeout/output/descendant-stop coverage and hidden noninteractive fixture children in tests/qualify_test.py (FR-008/010; incomplete timeout diagnostics discovered during CI).
