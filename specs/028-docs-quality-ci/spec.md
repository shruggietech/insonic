# Feature Specification: Reliable CI and usable documentation

**Feature Branch**: `codex/028-docs-quality-ci`

**Created**: 2026-10-14

**Status**: Specified, clarification complete

**Input**: Owner requests S028 for #93–#96 under Spec Kit autopilot, automatic push/official PR, at most two review rounds, final exact-head green CI and owner review/merge. Keep #97 for after merge.

## User Scenarios & Testing

### User Story 1 - Reliable bounded delivery (Priority: P1)

Maintainers receive trustworthy Windows/Linux acceptance results and an eleven-minute CI window. The actual main lease-fencing failure is corrected without removing its assertions or extending unrelated product deadlines.

**Independent Test**: Exercise real catalog live-owner refusal, expired takeover, generation fencing and renewal under controlled timing; verify active CI/release job limits and full final workflow elapsed time.

**Acceptance Scenarios**:

1. **Given** a live owned lease, **When** another owner claims it, **Then** admission refuses; after deliberate expiry it succeeds at the next generation and the stale owner cannot renew.
2. **Given** prior ten-minute CI limits, **When** S028 runs, **Then** applicable limits and current governance consistently allow eleven minutes, preserving historical evidence and every elected integrity/security gate.

### User Story 2 - A durable current documentation entry (Priority: P1)

Visitors follow ordinary documentation links into the current manifest-elected documentation, can identify its candidate/released state and can deliberately browse historical versions.

**Independent Test**: Change latest in a synthetic manifest, export public/offline sites and follow default, alias, topical, archive and fragment links with and without JavaScript.

**Acceptance Scenarios**:

1. **Given** a later latest version, **When** a visitor follows the main documentation CTA or stable latest route, **Then** it reaches that version automatically and preserves the requested valid topic/fragment.
2. **Given** an explicit historical URL, **When** followed, **Then** that historical version remains authoritative; missing latest topics show a useful current index/failure rather than wrong content.

### User Story 3 - Readable mobile and accessible desktop documentation (Priority: P1)

Readers reach the article without a mobile link cloud, expand structured navigation when needed and retain useful contents, version context, keyboard access and readable complex content across the entire site.

**Independent Test**: Headless real public/offline exports at320,360/390,768 and desktop widths; keyboard/touch disclosure, theme, focus, text sizing, overflow and progressive fallback checks across each template and representative article type.

**Acceptance Scenarios**:

1. **Given** a narrow screen, **When** opening any documentation article, **Then** navigation starts compact/collapsed and all topics remain available through an accessible control.
2. **Given** keyboard use, enlarged text, unavailable scripting/storage or a dark/light preference, **When** navigating and reading, **Then** content and controls remain reachable and readable with visible focus and no unintended page-wide horizontal overflow.
3. **Given** code, tables, diagrams, schemas, references, changelog and screenshots, **When** reading at narrow widths, **Then** complex content is contained and useful captions/alternatives remain available.

### User Story 4 - Useful company attribution (Priority: P2)

README readers can follow the final ShruggieTech ownership attribution to the company website.

**Independent Test**: Render/inspect the final Markdown attribution and verify its destination while preserving its visible wording.

**Acceptance Scenarios**:

1. **Given** the README final attribution, **When** selecting ShruggieTech, **Then** the link opens https://shruggie.tech/ and the visible text remains A ShruggieTech project.

### Edge Cases

- Candidate documentation is current without becoming a false release/download claim.
- Explicit historical pages and schema resources keep version identity; aliases do not rewrite them.
- Trailing slashes, index.html/file navigation, nested topics, fragments, unknown topics, storage denial and script-disabled rendering remain useful.
- Wide tables/code/Mermaid content may scroll locally; surrounding article/navigation must reflow.
- Lease time boundaries are made explicit rather than depending on wall-clock sleeps or removing exclusion assertions.
- No owner desktop automation, personal browser profile, native installer/account/UI probe or broad process/window cleanup is used.

## Requirements

### Functional Requirements

- **FR-001**: Correct the demonstrated main Windows lease-fencing regression and retain real exclusion, expiry takeover, stale-generation refusal and valid renewal coverage.
- **FR-002**: Apply the owner's additional60seconds consistently to relevant CI/release ten-minute limits and current CI policy, producing an eleven-minute/660second window; preserve unrelated application-operation bounds and historical records.
- **FR-003**: Provide a stable manifest-driven latest-documentation entry and route ordinary public landing/header/README entry links through it.
- **FR-004**: Provide a clear version archive/chooser, truthful status and immutable explicit historical routes/content/schema identity.
- **FR-005**: Latest/topic navigation must support static hosting, offline files, trailing slash/index paths and fragments without loops or wrong-version fallback; script-disabled users receive an accessible direct destination.
- **FR-006**: Narrow-screen documentation navigation must be collapsed by default, clearly labelled and operable by touch and keyboard with correct state and visible focus.
- **FR-007**: Audit and correct navigation hierarchy, current-page indication, headings/landmarks, contents, previous/next and skip-to-content across all shared templates.
- **FR-008**: Correct reading typography, spacing, contrast, link/target/focus states, themes, reduced motion and enlarged text using the governed brand vocabulary.
- **FR-009**: Reflow at320,360/390,768 and desktop widths; contain genuine two-dimensional code/table/schema/diagram content without page-wide overflow.
- **FR-010**: Cover current/archived articles, references, contracts, changelog, long pages and installation figures while preserving frozen historical source content.
- **FR-011**: Public/offline navigation and essential content remain usable with scripting or preference storage unavailable and without remote assets.
- **FR-012**: Record a concrete standards audit, corrections, automated keyboard/accessibility/overflow checks, isolated headless screenshots and remaining unverified assistive-technology behavior; do not claim blanket certification.
- **FR-013**: README must end with A [ShruggieTech](https://shruggie.tech/) project, preserving visible endorsement and upstream governed kit bytes.
- **FR-014**: Preserve software/documentation/schema release1.0.1 candidate identity and exact-source package/capture provenance. Refresh affected application captures when input inventory changes.
- **FR-015**: Run configured integrity/tests and affected product/site checks, push an official PR closing #93–#96, answer/resolve every finding in at most two requested review rounds, and verify final exact-head green CI before owner handoff.
- **FR-016**: Exclude #97 release/tag/publication/deployment and owner merge from S028; implementation produces reviewable code/artifacts and the official PR only.

### Key Entities

- Documentation manifest: latest version, version status and version-specific pages.
- Documentation entry/alias: stable URL, elected version/topic, canonical target and direct fallback.
- Navigation state: viewport context, disclosure state, active topic and version.
- CI budget:660seconds, elected job/workflow bounds and elapsed execution evidence.
- Lease fixture: owner, generation and deliberate logical expiry; unchanged production authority.
- Standards evidence: covered template/content/viewport/theme/keyboard cases, findings, screenshots and limitations.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Real catalog exclusion/takeover/fencing/renewal regressions pass deterministically, and final elected Windows/Linux core/native/package/candidate checks are green.
- **SC-002**: Active applicable CI/release limits and policy are eleven minutes; final exact-head CI turnaround is no more than660seconds, reported honestly.
- **SC-003**: Every ordinary documentation entry follows the manifest's elected latest version; archive/version/topic/fragment cases and public/offline/script-disabled checks pass.
- **SC-004**: Tested320,360/390,768 and desktop layouts expose compact usable navigation, visible keyboard focus and zero unintended page-wide horizontal overflow across audited templates and themes.
- **SC-005**: Applicable standards findings are corrected, with current headless visual evidence and explicit limits of automated accessibility verification.
- **SC-006**: README final attribution is a correct company link; all relevant source/PNG/export integrity checks pass and governed kit bytes are unchanged.
- **SC-007**: Official PR links all four issues and milestone12, every external finding is answered/resolved within two requested rounds, and no #97 publication or merge action occurs.

## Assumptions

- Existing static Next.js export, local runtime scripts and Pages/offline delivery remain the architecture.
- Current source latest is1.0.1 candidate; publication/promotion remains #97 after merge.
- The owner explicitly authorizes push/PR and the eleven-minute policy amendment; no additional routine approval gate applies.
- All four journeys are required for S028; no partial MVP is declared complete.

## Clarifications

### Session 2026-10-14

- Q: Does the earlier ten-minute policy block the new budget? → A: No. The owner approved eleven minutes in #93; update active governance and retain historical evidence.
- Q: Does this slice publish the current candidate? → A: No. #97 stays after merge; public deployment observations are documented separately from source verification.
- Q: Is the documentation fix only a mobile toggle? → A: No. Complete the whole shared-template/content standards audit and meaningful verification specified in #95.
- Q: Must historical version links follow latest? → A: No. Only stable current entry routes follow latest; explicit archived routes retain their own contracts.
