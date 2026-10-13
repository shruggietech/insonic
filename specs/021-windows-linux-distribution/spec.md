# Feature Specification: Windows/Linux distribution and installation

**Feature Branch**: `codex/021-windows-linux-distribution`

**Created**: 2026-10-13

**Status**: Implemented and qualified; owner review/merge pending

**Input**: S021, GitHub #73 and #63, owner-authorized autopilot through official PR and at most two external review rounds.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Use the supported platform distribution (Priority: P1)

An operator obtains a Windows or Linux CLI-only or Desktop + CLI package. Current support, downloads and qualification consistently describe these two platforms. The retired macOS platform cannot block future candidates or releases.

**Why this priority**: The owner explicitly retires macOS support, superseding the previous three-platform obligation.

**Independent Test**: Assemble a candidate containing exactly four qualified package variants and observe that macOS is rejected as an unsupported target.

**Acceptance Scenarios**:

1. **Given** current development and release automation, **When** it qualifies or assembles a candidate, **Then** Windows/Linux each supply CLI and desktop packages, with exact source revision, complete inventories and every remaining qualification requirement enforced.
2. **Given** a macOS host or requested package target, **When** current owned tooling is entered, **Then** an actionable unsupported-platform result is returned and no owned macOS implementation or package path remains available.
3. **Given** historical published releases, **When** current support is narrowed, **Then** historical bytes, documentation and provenance remain truthful and intact while current support and download entry points identify Windows/Linux only.

### User Story 2 - Install and maintain a Windows ZIP (Priority: P1)

A newcomer chooses between Desktop + CLI and CLI-only, understands architecture/prerequisites, installs the complete ZIP tree, runs the first commands and keeps program files separate from library data.

**Why this priority**: The distribution changes already touch package text and helpers; incomplete instructions and copy failure behavior prevent reliable installation.

**Independent Test**: Follow the published numbered ZIP procedure using an extracted Windows package and a separate disposable library; version/help/init/doctor produce the stated results.

**Acceptance Scenarios**:

1. **Given** a download or installation entry point, **When** an operator selects a package, **Then** Desktop + CLI explicitly includes both clients, CLI-only contains no GUI, and x86-64/prerequisites/signing limitations appear before installation.
2. **Given** an extracted package and a new destination containing spaces or non-ASCII text, **When** the optional copy helper completes, **Then** the complete package is usable there and the success message names only included clients and their actual prerequisites.
3. **Given** a same-tree, descendant, existing or otherwise unsafe destination, **When** the helper is requested, **Then** it fails before copying and preserves existing user files.
4. **Given** an owned new destination and a forced partial-copy failure, **When** installation aborts, **Then** the cause is reported, no success is announced and only the verified helper-owned partial installation can be removed.
5. **Given** a ZIP update or removal, **When** documented steps are followed, **Then** work is allowed to stop through a supported route, rollback remains available, workspace/media/models/credentials are preserved, and no MSI registration, shortcut, PATH or uninstaller effect is invented.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Support Windows x86-64 and Linux x86-64 for CLI-only and Desktop + CLI packages; retire project-owned macOS support.
- **FR-002**: Remove macOS runners, build/native/media/package jobs, matrices, caches, signing/notarization inputs and dependency conditions from current automation; inspect repository rules for obsolete required checks.
- **FR-003**: Remove owned macOS runtime/build integration, including Keychain, application-bundle resources, platform paths, preview acceleration and webview/worker workarounds; reject unsupported host/target requests clearly.
- **FR-004**: Require exactly four future release/candidate variants with exact revision binding, complete package/source inventory, hash checks and required qualification. Historical release records remain valid under their historical contracts.
- **FR-005**: Reconcile constitution principle II, R01, current public/offline documentation, README, support/download matrices, bootstrap metadata and active planning with the two-platform contract.
- **FR-006**: Record the effective candidate version and compatibility effect of macOS retirement in changelog and concise release-note generation; preserve historical published documentation/artifacts. Account for residual macOS references as historical evidence, time-zone data or upstream metadata rather than owned current support.
- **FR-007**: Preserve all Windows/Linux CLI, desktop, credential, media, package, backend and security qualification; no remaining integrity check is weakened to accommodate retirement.
- **FR-008**: Explain the package inclusion relationship consistently in download cards, README, release highlights, generated Windows installation text and current online/offline help.
- **FR-009**: Put supported architecture, feature-specific Visual C++/WebView2 requirements, official prerequisite download routes and unsigned-publisher disposition before installation; separate media companions from separately configured inference environments/models.
- **FR-010**: Provide a numbered complete Windows ZIP procedure covering whole-tree extraction, stable program location, optional copy helper, an open terminal, quoted paths, version/help, separate library initialization and subsequent workspace selection. Use real supported commands.
- **FR-011**: Validate absolute source/destination identity before creation/copy; reject existing targets and source-contained targets, including path aliases that could bypass containment, without changing user files.
- **FR-012**: On copy failure preserve the original error, suppress success and clean only an exact newly created helper-owned destination. Never recursively remove a pre-existing or substituted user directory.
- **FR-013**: Document actual ZIP update, runtime closure/idle exit, rollback and manual removal effects separately from future MSI integration; preserve workspace/media/models/credentials by default. Link applicable installation security guidance and first-run setup status without claiming unfinished flows exist.
- **FR-014**: Execute the installation examples against extracted packages and qualify the copy helper with disposable paths, successful copy, containment rejection and partial failure; enforce current source/online/offline and generated package-text consistency.

### Key Entities

- **Supported package target**: Windows/Linux x86-64 with CLI-only or desktop variant and exact release/source identity.
- **Release candidate**: Four required qualified variants plus complete immutable package/source evidence.
- **Owned installation attempt**: Resolved source, elected absent destination, ownership proof and success/failure state.
- **Historical release**: Published version-specific artifacts and contracts, independent of current support.

### Edge Cases

- Case variants, trailing separators, relative paths, source descendants, reparse/alias paths, existing targets and concurrent target substitution must not escape copy or cleanup ownership.
- CLI-only text cannot name an absent GUI or require GUI-only WebView2.
- Missing prerequisites, offline prerequisite installation and unsigned packages remain understandable without inspecting inventory JSON.
- Missing/duplicate variants, wrong revisions, corrupt inventory and absent qualification remain candidate failures.
- macOS words in immutable history, upstream licenses/lock metadata and time-zone data are not current support.
- Existing durable work and external storage/catalog/graph choices remain unchanged by program-directory maintenance.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every future candidate accepts exactly the four supported qualified variants and rejects unsupported or incomplete inputs.
- **SC-002**: No current owned support surface or required repository check advertises or requires macOS. Every residual reference has a recorded disposition.
- **SC-003**: All qualified Windows ZIP examples produce their documented first-use result from a relocated complete package.
- **SC-004**: Every containment/ownership/failure regression preserves user files and produces no false success.
- **SC-005**: Current online/offline/generated installation guidance agrees on included clients, prerequisites and maintenance effects; historical release integrity is unchanged.

## Assumptions

- The existing unreleased aligned 1.0.1 candidate is the effective target. No release, merge, tag or public site promotion is authorized by S021.
- #65 MSI delivery, #64 complete security-dialog/screenshots and #66..#69 guided setup/service/worker acquisition remain separate outcomes. This slice links truthful current guidance and states their implementation status.
- Owner authorization for #73 overrides the old three-platform requirement. Shared CLI/runtime behavior and all supported backend alternatives remain mandatory.
- Push and official PR publication are explicitly authorized; all findings are resolved within at most two external review rounds, followed by green exact-head CI and owner merge handoff.

## Clarifications

### Session 2026-10-13

- Q: Does platform retirement rewrite published 1.0.0 records? A: No; current 1.0.1 contracts/tooling change and historical bytes/provenance remain intact.
- Q: Does Windows ZIP work include MSI or managed first-run setup? A: No; actual ZIP effects and current explicit configuration are documented, with the separate issues retained.
