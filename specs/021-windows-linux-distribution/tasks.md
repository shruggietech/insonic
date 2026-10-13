# Tasks: S021 Windows/Linux distribution and installation

**Input**: [spec.md](spec.md), [plan.md](plan.md), research, data model and distribution contracts.

## Phase 1: Setup

- [x] T001 Establish branch/slice/issue identity and owner authorization in specs/021-windows-linux-distribution/spec.md.
- [x] T002 Update the owner-approved platform obligation in .specify/memory/constitution.md; record governance 3.0.0 and policy decisions in specs/021-windows-linux-distribution/plan.md.
- [x] T003 Audit required repository checks and research platform/history/installation boundaries in specs/021-windows-linux-distribution/research.md (FR-002).

## Phase 2: Foundation

- [x] T004 Add failing current four-target/completeness and historical matrix regressions in tests/release_test.py, tests/package_desktop_test.py and tests/release-versions.test.mjs (FR-001/FR-004/FR-006/FR-007).
- [x] T005 Add failing unsupported-host and retained credential/companion regressions in internal/workspace/host_test.go, internal/packageenv/packageenv_test.go and desktop/qualification_test.go (FR-001/FR-003/FR-007).

## Phase 3: US1 supported distribution

**Independent test**: Four qualified Windows/Linux variants assemble; incomplete/extra/current Darwin candidates fail; historical 1.0.0 remains valid.

- [x] T006 [US1] Add shared supported-host diagnostics before runtime/workspace side effects in internal/workspace/host.go, cmd/insonic/main.go, internal/app/app.go and cmd/insonic-desktop/main.go (FR-001/FR-003).
- [x] T007 [US1] Remove owned Darwin credentials/backup/probe files and branches from internal/secrets/, internal/backup/, cmd/insonic-secret-*/, internal/workspace/workspace.go, internal/packageenv/packageenv.go, internal/process/environment.go, internal/app/playback_preview.go and desktop/qualification.go (FR-003/FR-007).
- [x] T008 [US1] Remove current Darwin dependency/companion recipes in internal/qualification/dependencies.json, media-tools.json, scripts/media_source_build.py, scripts/processing-worker.py and scripts/qualify.py (FR-003/FR-007).
- [x] T009 [US1] Remove macOS workflow jobs/matrices/inputs/caches/dependencies in .github/workflows/{ci,release,native-qualification,media-source}.yml (FR-002/FR-007).
- [x] T010 [US1] Enforce current packages=4/content assets=6/published assets=8 while retaining historical 6/8/10 validation in scripts/release.py, scripts/package-desktop.py and schemas/v1.0.1/release-candidate.schema.json (FR-001/FR-004/FR-006/FR-007).
- [x] T011 [US1] Reconcile current support/bootstrap/roadmap and release highlight generation in README.md, .github/bootstrap.json, .github/RELEASE_NOTES_TEMPLATE.md, specs/roadmap.md and docs/v1.0.1/ (FR-005/FR-006/FR-008).

## Phase 4: US2 Windows ZIP installation

**Independent test**: Extracted CLI/desktop first commands and copy helper success/rejection/failure preserve ownership and user data.

- [x] T012 [US2] Add failing executed helper containment/alias/ownership/failure tests and platform/variant INSTALL assertions in tests/package_desktop_test.py and tests/windows_install_test.py (FR-008/FR-009/FR-011/FR-012/FR-014).
- [x] T013 [US2] Implement generated Windows copy-helper physical containment, uniquely owned attempts, atomic promotion and verified cleanup in scripts/package-desktop.py and scripts/install-windows.ps1 (FR-011/FR-012).
- [x] T014 [US2] Generate platform/variant-specific operator INSTALL text with actual clients/prerequisites/effects in scripts/package-desktop.py (FR-008/FR-009/FR-010/FR-013).
- [x] T015 [US2] Add complete Windows ZIP choice/install/update/remove contracts and actual idle shutdown/data separation in docs/v1.0.1/desktop.md, storage.md and README.md (FR-009/FR-010/FR-013).
- [x] T016 [US2] Align current Windows/Linux download cards and links in site/app/page.jsx, site/app/styles.css, site/lib/release-status.mjs and tests/site-release.test.mjs (FR-005/FR-008/FR-009).
- [x] T017 [US2] Execute first-use/install-helper procedures against extracted Windows CLI/desktop packages in scripts/package-desktop.py and scripts/cli_contract_qualification.py (FR-010/FR-014).

## Phase 5: Convergence and publication

- [x] T018 Record owned/historical/transitive/time-zone residual dispositions and preserve frozen 1.0.0 hashes in specs/021-windows-linux-distribution/platform-audit.md (FR-005/FR-006, SC-002/FR-005).
- [x] T019 Run full local runtime/schema/Python/frontend/site and affected extracted-package checks; record evidence in specs/021-windows-linux-distribution/verification.md (FR-007/FR-014, SC-001..005).
- [x] T020 Converge against every requirement/acceptance/decision/principle, fix gaps and update CHANGELOG.md with support withdrawal and dated policy decisions.
- [ ] T021 Commit/push and publish/attach official PR for #73/#63; update Project 7 review tracking and resolve every finding within two rounds; record exact-head green CI in specs/021-windows-linux-distribution/verification.md.

## Dependencies and execution

Setup -> foundation red tests -> supported distribution -> ZIP behavior -> convergence/publication. All edits touching scripts/package-desktop.py are sequential. Independent read-only research ran under speckit-plan; production editing is coordinated. US1 is the first usable increment, but both stories are required for this slice. No merge/tag/release is authorized. Custom checklist review markers remain owner-controlled; explicit autopilot authorization permits implementation without creating an attended gate.
