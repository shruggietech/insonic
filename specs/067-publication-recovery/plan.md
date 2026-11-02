# Implementation Plan: S067 Publication recovery and timing guidance

**Branch**: `codex/067-publication-recovery` | **Date**: 2026-11-01 | **Spec**: [spec.md](spec.md)

## Summary

Resolve #241 with durable promotion-attempt receipts, bounded exact-head reconciliation and a read-only maintainer recovery command. Resolve #240 by auditing current publication timing prose against executed policy. Track both in milestone 52 and Project 7.

## Technical Context

Python 3 standard-library maintainer tooling; JSON receipts; unittest fixtures; Node repository/site checks and Next.js documentation export. Windows and Linux supported tooling. No dependency or product-version changes. Existing GitHub API, destination preflight, candidate/publication validation and file-inventory contracts remain authority.

Promotion retains its 3,600-second aggregate deadline. Ref reconciliation permits at most three observations separated by at most one second, clipped to remaining time. Exact commit/tree validation and successful build remain mandatory. Receipts use atomic same-directory replacement after flush/fsync, with exclusive initial creation to prevent overwriting old attempts. Mutation observations contain validated SHAs and sanitized categories, not remote payloads or exception strings.

## Constitution Check

Pre-design and post-design: PASS. Owner-approved scope and publication boundaries preserved; no product eligibility or attended gate. Source/release bytes remain immutable. Existing authentication and destination restrictions retained. Hidden noninteractive launchers only. Mandatory checks remain. Qualification target/hard limit/reserve are unchanged; wrapper timeouts never establish acceptance. Existing branch/owner squash merge model applies.

## Project Structure

- `scripts/pages.py`: atomic receipt helper, staged identity, reconciliation and original-tree verification.
- `scripts/release.py`: durable promotion integration, read-only `pages-reconcile` and source/archive/publication validation.
- `.github/workflows/release.yml`: always-upload attempted receipt with attempt-qualified name.
- `tests/pages_test.py`, `tests/release_test.py`, workflow tests: injected failure/recovery and authority regressions.
- `docs/v1.1.0/releases.md`: current maintained timing and recovery guidance.
- `CHANGELOG.md`: behavior and dated scoped workflow decision.
- This directory: specification, research, data model, CLI contract, quickstart, tasks and verification.

## Decisions and execution order

1. 2026-11-01: Specify and clarify under autopilot. No hooks registered. The checklist prerequisite unexpectedly requires plan.md; initialize the plan template, generate the checklist, then complete research/design. No implementation preceded analysis.
2. Research existing callback patterns but use atomic receipt persistence instead of truncating an existing file. Keep failed attempts and later read-only results separate.
3. Retry only unavailable/previous-head observations; immediately reject any unrelated valid head or malformed identity. Do not assert the missing S066 response was stale.
4. Persist qualified/staged/update-attempted/updated/build-requested/deployed/public-verified phases plus failed phase/category. Retain previous observations when later recovery succeeds.
5. Read-only recovery revalidates original candidate and official assets, destination, original parent tree, staged parent/tree/files, exact current head and completed build, then public content. No POST/PATCH.
6. Audit documentation against callers (website Git is 600 seconds, despite a separate general Git constant). Preserve 1,800/3,600/60 qualification policy.
7. Verify, commit, push and open one official PR. Honor two-round maximum; stop for owner merge when CI and actionable reviews are satisfied.

## Verification

Write failing deterministic tests before implementation. Run targeted publication/recovery/website suites, repository check and tests, site tests/check/build, then relevant hosted CI. Retain compact acceptance receipts outside Actions before any full rerun. Read-only live-site smoke does not authorize deployment. Custom checklist remains reviewer-owned; proceeding follows the explicit autopilot request, without altering reviewer markers.
