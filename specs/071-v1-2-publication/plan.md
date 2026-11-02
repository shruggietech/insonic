# Implementation Plan: v1.2.0 publication and independent verification

**Branch**: `codex/071-v1-2-publication` | **Date**: 2026-11-02 | **Spec**: [spec.md](spec.md)

**Input**: Owner-authorized S071, accepted S070 squash 61a5ba35e2b388965ac316412e5a380e6e99b258, no new tracking issue.

## Summary

Qualify a publication-target candidate from accepted merged source using the existing Release workflow, retain and independently validate its original bytes, publish v1.2.0, promote its original documentation to insonic.io, independently verify downloads/content/rendering, and deliver a separate evidence PR for owner merge.

## Technical Context

**Language/Version**: Existing Python 3.12, Node >=22.12 and source-pinned Go 1.27.1; no dependency changes.
**Primary Dependencies**: scripts/release.py, pages.py, public-verification.py, website.py, check-live-site-ui.mjs, existing GitHub Actions/Release/Pages contracts and Playwright 1.64.1.
**Storage**: Ignored .local/s071 original archives and diagnostics; compact source-bound audited receipts in this feature and final PR conversation.
**Testing**: npm run check, npm test, maintained release/Pages/public-verification/website regressions, site tests/build, accepted main CI, complete Release lanes and final applicable PR CI.
**Target Platform**: Windows/Linux amd64 CLI and desktop, Windows MSI/setup and existing Pages site.
**Project Type**: Operational publication and evidence slice.
**Performance Goals**: Existing 1800-second qualification target, 3600-second hard limit and provisional 60-second reserve. Release lanes overlap and lack a shared CI readiness gate; retain lane preparation/qualification timestamps plus a conservative first-repository-check-to-terminal upper bound rather than inventing a shared readiness boundary.
**Constraints**: Clean trusted current-main tooling, immutable accepted product source, no rebuild after candidate acceptance, hidden/noninteractive console processes, headless browser, no real-model/signing election, at most two reviews, owner PR merge.
**Scale/Scope**: Ten original release files, complete public/offline documentation and eight landing viewport/theme cases.

## Constitution Check

- I: Owner authorized the recommended publication and evidence scope; routine operation needs no repeated approval. No product policy changes.
- II: Existing shared CLI/runtime and Windows/Linux contracts retain required native/package proof.
- III: Original source, captures, immutable assets and explicit provenance remain bound and unchanged.
- IV: Existing credentials and Pages destination are reused without provider rerouting; secrets stay out of logs and committed artifacts.
- V: Meaningful existing integrity/security regressions precede publication, terminal evidence controls timing and claims, and final checks/reviews precede owner handoff.

Pre-research and post-design gates pass. No constitution exception. Reviewer-owned custom checklist markers remain untouched under explicit autopilot authorization.

## Design decisions

- **D1 Source**: Ship S070 squash 61a5ba35e2b388965ac316412e5a380e6e99b258; main CI 41490507754 passes. S071 records are not product source. Reject substituting the pre-merge S070 candidate or temporary merge evidence.
- **D2 Candidate**: Dispatch release.yml from main with full revision, publish=false, promote_docs=false and documentation-transport=pages. This produces release-target public/offline docs. Reject ordinary candidate-status docs and publish=true redispatch after acceptance because it rebuilds.
- **D3 Checkout**: Commit planning locally and switch the existing checkout to clean main for trusted tooling. Return to S071 for evidence. If remote main moves, preserve selected source and use separate clean trusted/current-main and pinned-source checkouts rather than relaxing trusted history.
- **D4 Retention**: Retain run/job/step/artifact metadata and unique release-candidate-ATTEMPT ZIP with saved-byte digest/size verification; extract only bounded regular confined files. Bind artifact to run/attempt/input source, load complete candidate and explicitly verify both docs archives with require_release=True. Preserve failed outcomes before full reruns.
- **D5 Publication**: Recheck main, successful source checks and absent/conflict-free tag; create immutable v1.2.0 at selected source and publish accepted bytes with S070 highlights through draft upload/readback. No rebuild after acceptance.
- **D6 Pages**: Read-only preflight before release. Promote the original published archive with parented non-force update, retain exact staged/pushed/deployed identity and require its completed build. Uncertain updates use maintained exact-identity reconciliation. Do not infer success or silently roll back.
- **D7 Independent proof**: Download all ten official files into a fresh directory with published, compare name/size/digest to accepted candidate, run verify-public against exact Pages commit, then verify-deployed and eight live landing cases. Keep API metadata proof distinct from downloaded-byte proof.
- **D8 Windows UI**: Use the installed full Chromium chrome.exe with headless=true through hidden parent tooling; avoid console headless-shell and all owner-desktop GUI acceptance. Required native acceptance stays on isolated hosted runners.
- **D9 Delivery**: Record successful publication under Unreleased only after proof exists, retain detailed evidence internally, converge and push/open one official PR. Automatic review counts as round1; request at most one additional round. Operational final-head gates close externally without a self-referential acceptance commit.

## Project Structure

This feature contains spec.md, plan.md, research.md, data-model.md, contracts/publication.md, quickstart.md, tasks.md, checklists, verification.md and compact evidence/. Existing .github/workflows/release.yml and maintained release/Pages/public-verification/website scripts remain implementation authority; tests/ contains their regression suites. CHANGELOG.md records publication history. Product implementation and published version authorities are unchanged.

**Structure Decision**: Reuse existing checkout, release tools and provider contracts. No runtime subsystem, package dependency or new issue.

## Execution sequence

Spec Kit -> local guard checks and planning commit -> clean trusted main and retained main acceptance -> Release preparation -> retained terminal/candidate acceptance -> tag and unchanged-byte publication -> exact Pages promotion -> independent asset/content/render verification -> evidence/changelog -> convergence and local checks -> official PR, capped reviews and final CI -> retained handoff and owner merge boundary.
