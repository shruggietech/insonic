# Feature Specification: v1.0.1 release and documentation publication

**Feature Branch**: `codex/029-v1-0-1-release-publication`

**Created**: 2026-10-14

**Status**: Specified and clarified

**Input**: S029, #97, under autopilot; push the official PR, resolve at most two review rounds, wait for green CI and owner final review/merge, then publish v1.0.1 and matching documentation.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review an accurate release cut (Priority: P1)

A maintainer can review the complete v1.0.1 changes, compatibility implications and publication state before merging release preparation.

**Why this priority**: The next official download must include all accepted changes without presenting an unpublished candidate as shipped.

**Independent Test**: Validate synchronized release identities, a single dated v1.0.1 changelog section containing current changes, concise highlights, and truthful candidate/released archive navigation.

**Acceptance Scenarios**:

1. **Given** merged S028 prerequisites and an unpublished v1.0.1, **When** release preparation is reviewed, **Then** all software, documentation, package and master-schema versions agree at 1.0.1 and current changes appear once in its dated changelog.
2. **Given** the official v1.0.0 release exists, **When** the current archive is rendered, **Then** it is labeled released while frozen historical documents, schemas and assets remain unchanged.
3. **Given** the release-preparation PR has green checks and satisfied reviews, **When** the agent hands it off, **Then** the owner performs the squash merge and publication remains pending.

### User Story 2 - Publish complete exact-source downloads (Priority: P1)

After owner merge, a maintainer receives a fresh qualified v1.0.1 distribution built from the final reviewed main revision, followed by an official complete release.

**Why this priority**: Temporary PR artifacts and older source builds do not fulfill #97.

**Independent Test**: The reviewed main source, immutable tag, complete candidate receipts, draft asset digests and published release identities all agree.

**Acceptance Scenarios**:

1. **Given** the owner confirms merge, **When** qualification runs, **Then** it freshly qualifies four Windows/Linux x86-64 CLI/Desktop+CLI archives, the selectable per-user Windows MSI, public/offline help, inventories, corresponding sources/notices, checksums and unsigned disposition from the exact final main source.
2. **Given** all elected checks pass and remote draft assets match, **When** publication completes, **Then** v1.0.1 is an official release with working downloads and brief highlights ending in its exact-tag master changelog link.
3. **Given** an incomplete candidate, source/tag mismatch or interrupted upload, **When** publication is attempted, **Then** existing integrity checks refuse publication and no incomplete public release or documentation promotion is reported.

### User Story 3 - Read the matching live documentation (Priority: P2)

Readers discover the new release and read matching current documentation with usable mobile navigation, while older links and schemas continue to work.

**Why this priority**: A published binary with stale or mismatched live help is incomplete delivery.

**Independent Test**: Promote the original verified published documentation archive, confirm its deployed source/version and check public routes and screenshots without owner-host assistance.

**Acceptance Scenarios**:

1. **Given** the verified v1.0.1 release, **When** documentation is promoted through the existing Pages destination, **Then** the exact published documentation archive reaches insonic.io with matching identity and complete download links.
2. **Given** a mobile reader or a historical URL, **When** the site is read, **Then** latest-doc routes, compact navigation, company attribution and historical documentation/schema paths remain usable.
3. **Given** publication succeeds but promotion fails, **When** work resumes, **Then** the existing release's original assets can be verified and promoted without rebuilding or replacing them; #97 remains open until deployment is verified.

### Edge Cases

- An existing v1.0.1 tag or release conflicts with the selected source: preserve immutable published identity and report the concrete conflict.
- A newer main commit arrives between selection and publication: retain and verify the exact reviewed selected revision and trusted-main ancestry.
- Existing v1.0.1 changelog entries coexist with Unreleased changes: consolidate them without duplicate version headings or loss of older notes.
- Publication status overlays must not change frozen historical source or advertise v1.0.1 as available before publication.
- Native installer/account/UI qualification runs only on disposable hosted Windows runners; local browser evidence uses isolated owned headless contexts.
- A workflow or Pages result belongs to another revision/attempt: do not count it as selected-source success.
- External review posts findings in either authorized round: answer every comment, fix actionable findings and resolve threads; never request a third round.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Prepare v1.0.1 with synchronized software, package, documentation and master-schema identities; consolidate accepted changes into one dated version section, preserve compatibility/migration notes and end highlights of at most 180 words with the exact-tag master changelog link.
- **FR-002**: Correct current navigation's published v1.0.0 status without rewriting frozen historical documentation, schemas, downloads or inventories; retain truthful unpublished v1.0.1 source status until publication-specific rendering.
- **FR-003**: Push and open the official S029 release-preparation PR linked to #97 and its existing milestone/Project, but use a non-closing relationship so merge alone does not close delivery.
- **FR-004**: Resolve every external review comment with at most two requested rounds; verify final-head green CI and all thread states, then stop for owner final review and squash merge.
- **FR-005**: After owner merge, build a fresh complete candidate from the exact final reviewed main revision with released public/offline help and all four Windows/Linux package variants plus MSI.
- **FR-006**: Preserve all elected runtime, security, tenancy, source, notice, hash and package acceptance gates and the approved eleven-minute budget; report exact source/attempt identities and measured elapsed time honestly.
- **FR-007**: Use existing trusted-main tools to verify tag/source ancestry, upload and independently verify every draft asset/hash before publishing the official v1.0.1 release; preserve truthful draft state on interruption.
- **FR-008**: Promote only the verified published documentation archive through the existing Pages/domain configuration, preserving historical routes and verifying the exact destination commit, deployed source/version and public release/download links.
- **FR-009**: Verify public latest-doc aliases, Windows MSI link, company attribution, schemas, compact mobile navigation and light/dark presentation with headless evidence; distinguish automated coverage from unverified screen-reader/other-engine behavior.
- **FR-010**: Close #97 and mark the Project Done only after official publication and matching deployed documentation are verified. PR readiness is an intermediate checkpoint.
- **FR-011**: Prohibit owner-host native UI/account/installer probes, personal browser use, desktop/session control and broad process cleanup; use hidden noninteractive tooling and exact owned-resource cleanup.

### Key Entities

- **Release cut**: version, release date, changelog, highlights, compatibility statements and source publication status.
- **Qualified candidate**: exact reviewed source and workflow attempt, complete package/documentation assets, receipts, inventories and digests.
- **Published release**: immutable version/tag/source, independently verified complete public assets and explicit signing disposition.
- **Documentation deployment**: original published archive, target identity, Pages destination commit and observed public source/version.
- **Delivery checkpoint**: pre-merge readiness versus post-merge publication/deployment, review round count, issue/Project state and evidence.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: One coherent v1.0.1 release cut includes all current accepted changes once, with zero version mismatches, unchanged frozen historical bytes and accurate release/archive labels.
- **SC-002**: Before owner merge, every elected final-head check passes and every review thread is satisfied within at most two requested rounds; elapsed CI turnaround is compared with 660 seconds without hiding queue time.
- **SC-003**: After merge, all four archives, one MSI and both documentation bundles match one exact reviewed main source; published asset digests match qualification evidence.
- **SC-004**: The official v1.0.1 downloads and matching live documentation are independently observed, historical routes remain available, and #97 closes only after that evidence exists.
- **SC-005**: Zero owner-host native installer/UI/account operations or personal browser interactions are needed.

## Assumptions

- Existing v1.0.1 has no official release or remote tag at kickoff; verify again before publication. Published v1.0.0 remains immutable.
- Existing release tooling and Pages configuration are reused. No new host, domain, signing identity or external route is introduced.
- Source may remain candidate-labeled while release-target snapshots show availability only in the qualified publication bundle.
- Owner push/PR authorization is immediate; tag/release/docs publication authorization takes effect after owner merge. The agent does not merge.
- No product features, dependency refresh, branding update or platform expansion are included.

## Clarifications

### Session 2026-10-14

- Q: Is the requested output a prerelease tag or the prepared official version? → A: Publish official v1.0.1 after owner merge, as #97 requests; do not invent an rc suffix or new version.
- Q: Does PR merge complete #97? → A: No. Keep it open through fresh exact-main qualification, publication and live documentation verification.
- Q: May an S028 PR candidate be renamed for publication? → A: No. The squash/main revision requires its own qualified build.
- Q: Is new signing or attended native testing required? → A: No. Retain explicit unsigned disposition and run native qualification only on disposable hosted runners.
