# Implementation Plan: v1.1.0 publication and independent verification

**Branch**: `codex/066-v110-publication` | **Date**: 2026-11-01 | **Spec**: [spec.md](spec.md)

**Input**: Owner-authorized S066, issue #239, milestone 51, Project 7.

## Summary

Use the existing release workflow to qualify publication-target bytes from reviewed source `82a1bfd7e8f4e389d451789ec7840d874127f39c`. Retain and validate the complete candidate, publish it unchanged, promote its original documentation, independently download every official asset and verify the live site. Commit publication evidence on the separate S066 branch, complete capped reviews and green applicable CI, and leave merge to the owner.

## Technical Context

**Language/Version**: Existing Python 3.12, Node (repository requires >=22.12) and source-pinned Go 1.27.1; no toolchain changes.

**Primary Dependencies**: Maintained release.py, pages.py, public-verification.py, website.py, qualification helpers, GitHub Release/Actions/Pages APIs and installed Playwright 1.64.1/Chromium 1248.

**Storage**: Git source and tracking plus ignored .local/s066 original archives, metadata, validation/download/render receipts and compact audited retention.

**Testing**: npm run check; npm test; maintained release/pages/public-verification Python regressions; exact-main CI, complete Release candidate lanes and final applicable PR CI. Public/download/browser readback is additional execution evidence.

**Target Platform**: Windows/Linux amd64 CLI and desktop packages, Windows MSI/setup, existing insonic.io Pages destination. Local operation tooling remains hidden/noninteractive.

**Project Type**: Operational release and evidence slice using existing product contracts.

**Performance Goals**: Thirty-minute qualification target, sixty-minute hard limit, supported terminal timing authority and provisional finalization reserve. Release lanes overlap and do not expose CI's shared aggregate; retain actual lane intervals and combined campaign wall time without inventing an additive readiness boundary.

**Constraints**: Clean trusted protected-main tooling, immutable accepted product source, complete finite archive/network validation, no rebuild after candidate acceptance, no owner-desktop GUI, no real-model CPU/NVIDIA election, two review rounds maximum, owner merge.

**Scale/Scope**: Ten original release assets, complete public documentation export and resources, eight maintained landing viewport/theme render cases, one official evidence PR.

## Constitution Check

- I: Owner publication/push/PR authorization is explicit; routine steps need no renewed permission. No product eligibility or routing changes.
- II: Existing shared CLI/desktop behavior and Windows/Linux matrix remain unchanged; capability claims derive from the exact candidate.
- III: Immutable source/assets, original captures, evidence lineage and historical documentation remain distinct from new records.
- IV: Existing credentials and Pages route are reused; credential values stay in process memory and out of files/logs. No external provider changes.
- V: Maintained security/integrity gates, foreground local tests, measured timing and truthful evidence boundaries remain required. Reviewer-owned custom checklist markers stay unchanged under owner autopilot authorization.

Pre-research and post-design checks pass; no constitution exception is needed.

## Design decisions

- **D1 Source authority**: Ship the reviewed S065 squash SHA. Main CI 41270520010 passed. S066 planning and evidence commits are not candidate source.
- **D2 Preparation**: Dispatch release.yml from main with the full selected revision, publish=false, promote_docs=false and documentation-transport=pages. It renders release-target docs. Ordinary CI candidate artifacts are not substituted.
- **D3 Checkout reuse**: Commit planning locally, temporarily switch the existing checkout to clean main for trusted tooling and exact-source operations, then return to S066 for records. If remote main changes, preserve candidate identity and use separate clean trusted/current-main and pinned-source checkouts instead of relaxing trusted_history.
- **D4 Retention**: Retain complete attempt-specific run/jobs/artifact inventories, verify the unique release-candidate-ATTEMPT ZIP size/SHA256, extract bounded regular files, load the complete candidate, and explicitly require release-target public/offline docs. Preserve workflow-head and selected product-source identities separately. Retain rejected outcomes before a full rerun.
- **D5 Publication**: Recheck remote main, tag absence/conflict and source checks; create immutable v1.1.0 at accepted source, publish the original candidate with prepared S065 highlights through draft upload/readback, and never rebuild accepted publication assets. Record actual signing and inference dispositions.
- **D6 Documentation**: Existing Pages preflight is read-only. Promote the original published archive with parented non-force update, wait for the exact build and retain its returned commit. A verification failure is investigated using the same bytes, without silent rollback.
- **D7 Independent bytes**: Download all ten assets with published into a fresh directory, then compare every name/size/digest with the retained accepted candidate. Separately run verify-public with the elected Pages commit; its API metadata proof alone is not described as binary download proof.
- **D8 Live rendering**: Materialize exact Pages files as data, use website.py verify-deployed and check-live-site-ui.mjs for eight fresh landing cases. Use explicit installed full Chromium chrome.exe in headless mode through hidden parent tooling; do not launch the console headless-shell binary on the owner host. These sampled render checks do not certify all browsers or accessibility technologies.
- **D9 Documentation observation**: Current release guide prose still names former 120/180-second aggregate verification allowances; actual qualification policy supplies 600/900 seconds. Record a separate proportional documentation follow-up, preserve frozen accepted source, and report actual executed budgets. No publication acceptance failure is demonstrated by this prose mismatch.
- **D10 Delivery**: Add a concise publication entry under Unreleased only after publication succeeds. Keep detailed receipts in this internal feature and retained evidence, then push/open one official evidence PR. Count automatic first and at most one requested second review; resolve every actionable finding, never trigger a third, and hand off only with final applicable CI green.

## Project Structure

### Documentation (this feature)

`specs/066-v110-publication/` contains spec.md, plan.md, research.md, data-model.md, contracts/publication.md, quickstart.md, tasks.md, checklists and verification.md. Compact durable receipts may be retained here after successful independent readback; full original archives stay under ignored .local/s066.

### Existing execution surfaces

- .github/workflows/release.yml and reused native/media/Windows workflows qualify original publication bytes.
- scripts/release.py, pages.py, website.py, public-verification.py and check-live-site-ui.mjs enforce existing contracts.
- tests/release_test.py, pages_test.py and public_verification_test.py cover meaningful publication/integrity failures.
- CHANGELOG.md records publication history; docs/ and schemas/ retain accepted product source during this campaign.

**Structure Decision**: Reuse maintained tools and the current checkout; no new runtime subsystem or package dependency is introduced.

## Execution sequence

Spec Kit and tracking -> local guard tests and planning commit -> clean trusted main/preflight -> fresh Release candidate -> terminal metadata and retained acceptance -> immutable tag/release -> exact Pages promotion -> independent public downloads/byte/render checks -> S066 evidence/changelog -> convergence/local checks -> official PR and bounded review/CI -> external final receipt and owner handoff.
