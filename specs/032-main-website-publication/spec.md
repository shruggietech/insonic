# Feature Specification: Reliable website publishing from main

**Feature Branch**: `codex/032-main-website-publication`

**Created**: 2026-10-15

**Status**: Specified

**Input**: S032 under autopilot: publish website/docs changes from reviewed green main independently of product releases; automate exact live verification, download checks and hosted screenshots; improve CI headroom below 660 seconds. Push and open the official PR, resolve at most two review rounds, then stop for owner final review and merge.

## User Scenarios & Testing

### User Story 1 - Publish reviewed website changes (Priority: P1)

A maintainer selects an exact green main revision and publishes its website and documentation to the existing destination while keeping official software downloads intact.

**Why this priority**: Website corrections currently require manual deployment because the product-release publisher can only restore an original release archive.

**Independent Test**: A controlled publication of a complete reviewed site advances only the configured website destination and records separate site-source and product-release identities.

**Acceptance Scenarios**:

1. **Given** green main and a matching existing published product version, **When** the maintainer requests website publication, **Then** the exact reviewed site's bytes are deployed with current download choices and stable latest-docs links.
2. **Given** missing or inconsistent published assets, an unqualified source or a changed destination, **When** publication is requested, **Then** it fails before changing the live branch.
3. **Given** a lost response or a concurrent deployment, **When** publication resumes, **Then** it reconciles only the exact selected result or reports the conflict without overwriting other work.

### User Story 2 - Prove the live result (Priority: P1)

A maintainer receives deployment identity, content, link and browser evidence from the same unattended run.

**Why this priority**: A successful build or an earlier deployment cannot prove that the elected correction is live.

**Independent Test**: Controlled stale-content, broken-download and browser-layout fixtures fail the verification result, while a matching site produces a bounded receipt and screenshots.

**Acceptance Scenarios**:

1. **Given** a completed deployment, **When** verification runs, **Then** it compares the complete elected static export, latest-docs destinations and all five current official package links with their expected identities.
2. **Given** the deployed page, **When** the unattended browser inspects desktop/mobile and light/dark states, **Then** captures and measured overflow/overlap/target checks describe the actual live site rather than a synthetic release fixture.
3. **Given** a deployment that completes but fails content or browser verification, **When** the run ends, **Then** it reports the deployed revision and failure without claiming success or silently changing product releases.

### User Story 3 - Finish qualification with headroom (Priority: P2)

Maintainers get proportionate complete checks within the existing CI turnaround budget.

**Why this priority**: The last green main workflow took 671 seconds, exceeding the 660-second requirement.

**Independent Test**: Timing evidence identifies the critical path and demonstrates the final elected hosted workflow below 660 seconds, with relevant integrity, security, native and documentation checks retained.

**Acceptance Scenarios**:

1. **Given** the existing workflow, **When** repeated or independent work is optimized, **Then** the final hosted qualification finishes below 660 seconds and records its timing boundary.
2. **Given** a runtime/package change or unavailable change information, **When** qualification is selected, **Then** the full required product checks remain selected.

### Edge Cases

- Product version is prepared but not officially released, or a release points to a different source than the website.
- Historical pages or schema bytes would disappear or change; static assets contain symlinks, path aliases or excessive payloads.
- Main moves after selection; latest CI is failed, cancelled, pending or superseded; required checks are missing.
- Git transport rejects a payload, the destination changes concurrently, or the push succeeds but its response is lost.
- An existing identical deployment is rerun; Pages reports the previous successful build while the new one is queued.
- Download responses redirect, time out or disagree with release metadata; live bytes become stale mid-verification.
- Browser verification fails after deployment; screenshots, failure evidence and owned-resource cleanup still complete.
- Windows maintenance helpers run without visible consoles or interaction with personal browser profiles.

## Requirements

### Functional Requirements

- **FR-001**: Maintainers MUST have a dedicated unattended website/docs publication entry point that selects an exact reviewed green main source and the existing destination.
- **FR-002**: Publication MUST validate the current product version against an already published release and preserve the five exact Windows/Linux MSI/CLI/Desktop download choices. Website source and software-release source MUST remain distinct, explicit identities.
- **FR-003**: Website publication MUST NOT create, move or replace product tags, releases or release assets, including their original documentation archives.
- **FR-004**: Publication MUST qualify and bind a complete clean-source static export before writes, retain historical documentation paths and schema bytes, and preserve domain/HTTPS/destination configuration.
- **FR-005**: Destination advancement MUST preserve concurrent changes, avoid force updates, and reconcile lost responses and identical reruns by exact content/commit identity.
- **FR-006**: Success MUST require the elected completed Pages deployment, complete exported live-byte comparison, correct latest-docs routing and valid official package-link responses with expected sizes.
- **FR-007**: The same run MUST record actual deployed landing screenshots and layout checks at 320, 390, 768 and 1440 CSS pixels in light/dark themes. Evidence MUST distinguish pre-publication exports, live content and any fixtures.
- **FR-008**: Networking and browser work MUST have aggregate deadlines, bounded payloads and retries, redacted diagnostics, and cleanup confined to owned processes/profiles/files. Console children on Windows MUST be hidden and noninteractive.
- **FR-009**: Publication failures MUST retain evidence and distinguish no destination change, successful deployment with failed verification, conflict and verified completion. Routine replay MUST NOT manufacture duplicate releases or silently roll back a deployed site.
- **FR-010**: Existing release publication/promotion MUST retain exact original-release integrity and authorization boundaries. New website automation MUST remain unavailable to untrusted source with write credentials.
- **FR-011**: CI improvements MUST remove measured redundant work, apply accurate scope or parallelize independent work, retaining required acceptance/security/artifact checks and eleven-minute job deadlines.
- **FR-012**: Maintainer documentation MUST explain selection, identities, publication, verification and recovery. Internal slice/tracking records MUST remain outside the public documentation.

### Key Entities

- **Website publication**: Exact main revision, qualified export digest/inventory, published product version/source and expected download inventory.
- **Destination identity**: Existing domain/source branch, observed parent commit, staged content/commit and elected completed build.
- **Verification receipt**: Phase outcomes, checked URLs/digests/sizes, actual live browser cases/screenshots, elapsed times and remaining unverified behavior.

## Success Criteria

### Measurable Outcomes

- **SC-001**: One maintainer invocation produces a verified website publication from green main without rebuilding or republishing product software.
- **SC-002**: Controlled invalid-source/release/export/destination scenarios cause zero live-branch or release mutations; concurrent updates are retained.
- **SC-003**: All elected static files, latest-docs entry points and five package links are verified; all eight live viewport/theme cases produce evidence and fail on measured overflow, overlap or inadequate target size.
- **SC-004**: The final elected hosted CI workflow completes within 660 seconds; timing evidence identifies queue and active-work boundaries and preserves every required qualification outcome.
- **SC-005**: Failure/replay coverage proves bounded networking and cleanup, truthful phase receipts and zero unintended product publication.

## Clarifications

### Session 2026-10-15

- Require current main again before writing and the newest exact-source push/main CI success. Older success cannot override a pending or failed rerun.
- Replay qualified archives; rebuilding the same source does not imply identical Next.js bytes.
- Use isolated Git with a parent fence and non-force push. Preserve historical schema bytes. Original release promotion retains integrity checks.
- Native CI reuses source/run/attempt/hash-bound documentation with no fallback. Keep dependencies, notices, native and MSI checks. Release callers build independently by default.

## Tracking

- [#106 website publication](https://github.com/shruggietech/insonic/issues/106)
- [#107 CI turnaround](https://github.com/shruggietech/insonic/issues/107)
- [Milestone S032](https://github.com/shruggietech/insonic/milestone/15)

## Assumptions

- Scope is the existing `insonic.io` Pages destination; no host migration, new product release, application feature or brand-kit update is requested.
- The current public documentation version must already have an official product release before website-only publication; future unpublished product versions remain release-workflow concerns.
- This slice delivers reviewed automation and controlled end-to-end evidence. The owner retains the final merge ritual; production use of newly merged publication automation follows a subsequent authorized invocation.
- Autopilot resolves routine clarification choices and permits push/PR publication. Review requests are capped at two actual code/security rounds for this PR.
