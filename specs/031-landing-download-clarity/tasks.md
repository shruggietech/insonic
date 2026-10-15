# Tasks: Landing-page download clarity

**Input**: [plan.md](plan.md), [spec.md](spec.md), research, data model and landing contract.

## Phase 1: Setup

- [x] T001 Align S031 branch, selector, issue #101, milestone and Project in `specs/031-landing-download-clarity/`.
- [x] T002 Record brand, primary guidance, release-manifest identity and isolated-browser research in `research.md`.

## Phase 2: Foundation

- [x] T003 Create a trusted-source publication-state renderer in `site/scripts/landing-fixture.mjs` and extract the unchanged pure source to `site/app/components/LandingPage.jsx`.

## Phase 3: US1 - Choose the right package

**Independent test**: Both rendered publication states and exact five release URLs, package labels and component-selection copy.

- [x] T004 [US1] Add failing rendered release/candidate, five-package and grouping cases in `site/tests/landing.test.mjs`.
- [x] T005 [US1] Implement ordered Windows installer/ZIP and Linux tar.gz choices, prerequisite copy and standalone labels in `site/app/components/LandingPage.jsx`.

## Phase 4: US2 - Navigate predictable links

**Independent test**: Every landing/chrome link has a matching label, resolved href, intent and version; no inaccurate Linux lifecycle promise or generic duplicate.

- [x] T006 [US2] Add rendered whole-page link-contract cases in `site/tests/landing.test.mjs`.
- [x] T007 [US2] Correct hero/architecture hierarchy and platform guidance destinations in `site/app/components/LandingPage.jsx`; record the complete inventory in `specs/031-landing-download-clarity/link-inventory.md`.

## Phase 5: US3 - Reflow and input

**Independent test**: Whole landing in light/dark at 320/390/768/1440, enlargement/reflow, actual focus and native activation without package downloads.

- [x] T008 [US3] Add geometry failure regressions and activation-guard isolation cases in `tests/docs-ui.test.mjs`.
- [x] T009 [US3] Add token-based action layout, spacing, wrapping and subordinate emphasis in `site/app/styles.css`.
- [x] T010 [US3] Extend `scripts/check-docs-ui.mjs` with snapshot/released-fixture inventory, control/label geometry, full-page screenshots, input/focus, 200% text and explicit 400%-equivalent reflow proof.

## Phase 6: Verification and delivery

- [x] T011 Update `CHANGELOG.md` and record local repository/site/build and encoding checks in `specs/031-landing-download-clarity/verification.md`.
- [ ] T012 Inspect hosted Linux browser reports/screenshots and all five official asset identities; record evidence and truthful limitations in `verification.md`.
- [ ] T013 Commit, push and open the official PR with `Closes #101`; resolve every finding in at most two rounds and verify CI on the final head. Record final acceptance externally on the PR after the last commit to avoid evidence-only CI churn.

## Dependencies and implementation strategy

T001/T002 -> T003 -> T004 -> T005 -> T006 -> T007 -> T008 -> T009 -> T010 -> T011 -> T012/T013. Browser acceptance uses the authorized pushed PR's hosted CI. No code tasks are parallelized because layout, renderer and browser contract are shared. US1 supplies package clarity first; US2 audits navigation; US3 validates final geometry and inputs. Each story retains its stated independent test. No partial story is a delivery completion claim.
