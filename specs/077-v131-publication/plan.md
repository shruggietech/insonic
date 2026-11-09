# Implementation Plan: v1.3.1 publication and public verification

**Branch**: codex/077-v131-publication | **Date**: 2026-11-08 | **Spec**: [spec.md](spec.md)

## Summary

Publish selected merged S076 source through existing release tooling, keep the accepted original candidate unchanged, promote its original documentation to existing Pages, and retain independent public file/content/render proof. No application implementation changes are planned.

## Technical Context

**Language/Version**: Python 3.12, Node 22+, selected Go 1.27.1 for shipped-binary notice verification; existing PowerShell copy installer.

**Primary Dependencies**: Existing .github/workflows/release.yml, scripts/release.py, package-desktop.py, Pages transport, website.py, check-live-site-ui.mjs and qualify-answers.py; GitHub CLI/API and installed full Chromium.

**Storage**: Retained original archives/receipts under ignored .local/s077; compact internal evidence under this slice; immutable official release assets and existing gh-pages branch.

**Testing**: npm run check, npm test, affected publication/Pages/public-readback/website/HTTP regressions and site tests/build/check; exact-source green main CI; original release qualification; elected seven-case answer campaign; independent public downloads/content and eight browser cases.

**Target Platform**: Windows/Linux amd64 CLI/Desktop and existing insonic.io Pages site.

**Project Type**: Operational publication and evidence slice.

**Performance Goals**: Qualification target 1800 seconds, hard limit 3600, unchanged provisional 60-second reserve. Retain preparation/qualification/combined phases. Release lanes have overlapping preparation; preserve lane timestamps and conservative first-repository-check-to-terminal bound instead of inventing one shared readiness boundary.

**Constraints**: Clean trusted current-main tooling; immutable selected source; no rebuild after acceptance; hidden noninteractive console children and full headless Chrome; exact existing model election; explicit publication authorization; at most two reviews and owner evidence-PR merge.

**Scale/Scope**: Ten original files, four native variants and coordinated installers, all inventoried public/deployed bytes and six product links, eight sampled landing cases.

## Constitution Check

- I: Owner explicitly approved publication, promotion and independent verification. No new eligibility/product policy or repeated routine approval.
- II: Required shared CLI/runtime/native package contracts remain qualified through existing exact-source lanes.
- III: Source, original archives, model/corpus identity, provenance and uncertainty remain immutable and distinct.
- IV: Reuse elected local provider and configured Pages destination; credentials remain outside files/logs.
- V: Blocking analysis, existing integrity/security guards, bounded timing/retention and truthful shipped evidence precede closeout.

Pre-research and post-design gates pass. No constitution exception. Eight custom reviewer-owned checklist markers remain untouched under explicit autopilot authorization.

## Design decisions

- **D1 Source**: Ship 1443198e0dec4738c129bb6dfe48baba9830d6e3 after main CI 42537775325 succeeds and original audit is retained. Evidence branch is not product source.
- **D2 Preparation**: Dispatch existing release.yml from main with full revision and publish=false/promote_docs=false/pages. Require release-target documentation; reject publish=true redispatch after acceptance.
- **D3 Checkout**: Commit planning, switch existing checkout to clean main for operations, return to this branch for evidence. Separate pinned source/trusted tooling only if main moves.
- **D4 Candidate**: Retain terminal run/job/step/artifact metadata, acceptance and original candidate with API digest/size equality, confined regular extraction and maintained complete candidate/source/docs verification.
- **D5 Actual model**: Documented copy-install original Windows CLI, compare complete inventory and exact source, then seven frozen cases with exact verified model, package companions, original provider deadline and owned cleanup. Preserve null/not-applicable scores.
- **D6 Publication**: Recheck source checks/trusted main/tag conflicts, create annotated exact tag and publish accepted bytes through verified draft. No rebuild or new signing election.
- **D7 Pages**: Preflight existing destination, promote original archive with parented non-force update, require exact completed build. Ambiguous outcomes use pages-reconcile against original unique attempt.
- **D8 Independent proof**: Fresh all-file download/name-size-digest comparison, verify-public against exact Pages commit, complete tree verify-deployed, eight full Chromium headless viewport/theme cases and visual inspection.
- **D9 Evidence**: Unique original receipts, hashes and decompression readback outside Actions, compact internal evidence and truthful changelog after success. Public docs do not link numbered planning.
- **D10 Delivery**: Push/open evidence PR under explicit release delivery; count automatic first plus at most one manual second review. Do not reproduce literal review triggers in evidence. Operational exact-head final gates close externally to avoid self-referential commits; owner retains merge.

## Project Structure

This slice owns spec.md, plan.md, research.md, data-model.md, contracts/publication.md, quickstart.md, tasks.md, checklists, verification.md and compact evidence/. Existing scripts/workflows implement publication; .local/s077 contains confined orchestration and original evidence. CHANGELOG.md receives historical publication prose only after proof. Product source/version/dependencies remain unchanged.

**Structure Decision**: Reuse maintained tooling and existing checkout. No runtime subsystem or duplicate product issue.

## Execution sequence

Spec Kit -> local guards/planning commit -> clean main and retained exact main CI -> release preparation -> original candidate/package qualification -> installed actual-model proof -> tag and unchanged-byte publication -> exact Pages promotion -> independent public file/content/render proof -> evidence/changelog/convergence -> official evidence PR, capped review and exact-head CI -> owner handoff.
