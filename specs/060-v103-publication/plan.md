# Implementation Plan: v1.0.3 exact-main acceptance and publication

**Branch**: `codex/060-v103-publication` | **Date**: 2026-10-29 | **Spec**: [spec.md](spec.md)

## Summary

Produce one fresh publication-target candidate at reviewed main 72098446d435aa1e3d6d6726a9f07e04e0b227a6, retain its original assets, qualify its exact Windows CLI archive through the maintained real CPU/fresh-restored journey, then publish those same bytes and matching original documentation. An official S060 evidence PR stays open for owner merge after final CI and at most two Codex rounds.

## Technical Context

**Language/Version**: Existing Go 1.27.1 product, Python 3.11+ qualification/publication tooling and Node 22+ documentation checks.
**Primary Dependencies**: Maintained release.yml, scripts/release.py, qualify-journey.py, journey_cpu.py, acoustic_reference.py, exact managed worker/model catalogs and Pages transport.
**Storage**: Owned external candidate retention plus compact internal/public GitHub receipts; no application storage changes.
**Testing**: Root check/test, affected site checks/test/build, complete candidate/platform receipts, strict real CPU/fresh-restored validator and public release/Pages verification.
**Target Platform**: Windows/Linux x86-64 CLI and desktop-with-CLI, plus applicable Windows MSI/setup.
**Project Type**: Operational release and evidence slice; no intended product-code or public-interface change.
**Performance Goals**: Existing 1800-second qualification target and 3600-second hard limit; retain separate preparation/qualification/combined accounting and terminal timestamp authority.
**Constraints**: Hidden local processes, isolated hosted GUI acceptance, exact original byte identity, immutable history, unmerged evidence PR and no Downloads writes.
**Scale/Scope**: Four product archives, two Windows installers, public/offline documentation, maintained original/restored CPU stages and the existing Pages destination.

## Constitution Check

Pre-research and post-design PASS for all five principles. Owner authorization explicitly permits publication from accepted main while reserving evidence-PR merge. Shared runtime, source/provenance, configured routes/credentials and existing substantive/timing gates remain intact. No attended owner-desktop gate or new product restriction.

## Phase 0 Research

The installed plan skill directed one read-only research agent. Maintained code provides the retained-byte publication route and exact-source checks. Candidate production must use publish=false/promote_docs=false; publish=true would rebuild and cannot inherit earlier CPU acceptance. A clean detached main worktree is required for trusted tooling and selected source. Existing exact pinned model/managed prerequisites are verified before qualification.

## Phase 1 Design

1. Preserve the already-reviewed release source and its existing 1.0.3 changelog/highlights. Keep operational planning/evidence on this branch. No new user-facing change or architecture decision is introduced. A demonstrated CI scheduling repair is recorded under Unreleased; frozen release documents and selected release bytes remain unchanged.
2. Use a dedicated clean managed worktree at the selected main revision for qualification and trusted publication. Keep the primary checkout on the evidence branch. If main advances, maintain separate trusted-current-main and exact-candidate source authority.
3. Open the official evidence PR after the Spec Kit analysis gate and initial root/site validation. Describe incomplete operational phases accurately. Its initial automatic review may run while candidate production proceeds; request at most one second round after final evidence.
4. Dispatch release.yml from main with exact revision and publication/promotion disabled. Independently verify original artifact ZIP run/head/attempt, size/digest and confined regular-file extraction. Retain the flat complete candidate (eight assets plus manifest/checksums), original package receipt and hashes outside workflow expiry before CPU execution. Compare the CLI receipt with its manifest package object.
5. Validate the retained candidate with clean trusted-main load_candidate and publication-target documentation checks. Keep capabilities.inference_qualification at its original package-time value; store real CPU acceptance separately, bound to archive/executable identity.
6. Run the maintained real reference-corpus journey from clean source, using only exact selected package/receipt/revision and verified models/managed inputs. Require all original and fresh-restored stages, original-path withdrawal, retained authority and owned cleanup. Independently validate the resulting receipt and rehash unchanged candidate assets.
7. After acceptance, create/push v1.0.3 at the selected reviewed revision, then run trusted-main release.py publish on the retained candidate and S059 highlights. Verify original remote draft assets before publication. Never use a rebuilding publication dispatch after acceptance.
8. Preflight/promote the original released documentation with pages-promote, then independently verify-public with the returned exact Pages commit and an owned saved receipt. Preserve destination configuration/history and inspect actual state after any failure.
9. Retain compact recoverable source/package/CPU/publication/deployment/public/PR-CI/review receipts with readback and hash verification. Finalize records, require final applicable CI and resolved findings, remove owned scratch, and leave the PR unmerged.

## Timing Authority

The release workflow lacks CI's shared readiness/elected-acceptance aggregate jobs. Retain its run/job/step timestamps and select the earliest actual qualifying check as the release-wide qualification boundary; count parallel later preparation inside that interval instead of restarting a clock or omitting early checks. The real CPU receipt separately reports its maintained preparation/qualification/combined fields and enclosing bound. Existing CI audits retain their own aggregate authority.

## Project Structure

- specs/060-v103-publication/{spec,plan,research,data-model,quickstart,tasks,verification}.md and built-in checklists/requirements.md.
- Existing scripts/release.py, qualify-journey.py, journey_cpu.py, acoustic_reference.py, pages.py and release workflow are reused unchanged unless a demonstrated blocking defect requires a separately reviewed main repair.
- Complete candidate assets/original receipts use a uniquely owned retention directory; temporary download/audit payloads use a separate owned transient directory. Neither contains agent thought dumps or owner media.
- No new public contract family is introduced; existing schemas/manifest/receipt validators remain authoritative, so no extra contracts directory is needed.

## Tracking

Issue #198, milestone 44 and Project 7; Slice S060, Release 1.0.3. #147 remains open. Explicit push/PR and release publication authorization are recorded in spec.md. Final owner review is for the evidence PR merge.

## Demonstrated CI repair

2026-10-29: Initial planning-only run 40953107339 passed readiness but skipped foundation/docs-build/docs through intentionally unelected preparation ancestors. Add explicit always() conditions with successful direct-dependency checks (and readiness ready=true where applicable). Preserve every aggregate result requirement, existing clocks and component election. Retain the original failed gate ZIP and independently audited terminal API evidence before pushing the repair. The repair belongs only to the evidence PR; selected product/release source remains reviewed main 72098446d435aa1e3d6d6726a9f07e04e0b227a6.

2026-10-29: The now-elected read-only maintenance smoke rejected the immutable v1.0.2 release manifest (7,481,000 bytes) under its old 1 MiB default HTTP metadata bound. Increase only that verifier default to 16 MiB, preserving explicit smaller limits, bounded read, actual/API size and SHA-256 checks, approved redirects and body-free HEAD checks. A failing-first reader regression covers the real manifest size, oversized/incomplete bodies and explicit small bounds. The publication path reuses unchanged trusted release.py at selected main and does not execute this evidence-branch website repair.
