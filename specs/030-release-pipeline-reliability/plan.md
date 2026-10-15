# Implementation Plan: Release pipeline reliability

**Branch**: `codex/030-release-pipeline-reliability` | **Date**: 2026-10-15
**Spec**: [spec.md](spec.md)
**Tracking**: #102 and #103, milestone S030 - Release pipeline reliability.

## Summary

Acquire the existing pinned setup-go and setup-node archives in bounded job
steps, verify locked SHA256, and invoke unchanged actions through local paths.
Use a sparse checkout at job.workflow_sha for independently selected historical
sources. Reuse original verified candidates for public release and live docs
verification, integrated after exact Pages promotion.

## Technical Context

**Language/Version**: Python 3 standard library; existing Node contract tests.
**Primary Dependencies**: Existing pinned checkout and setup actions, release.py
and pages.py. No new runtime package dependency.
**Storage**: Ignored build staging and a committed action lock.
**Testing**: Python unittest, Node workflow contracts, full product/site CI.
**Target Platform**: Windows 2022 and Ubuntu GitHub-hosted runners; HTTPS docs.
**Project Type**: Maintainer tooling.
**Performance Goals**: <=3 action acquisition attempts in <=60 seconds each;
public HTTP phase <=120 seconds; final PR CI turnaround <=660 seconds.
**Constraints**: No native installer, personal browser or desktop interaction.
No product changes, release/tag movement or publication during this slice.
**Scale/Scope**: Two setup actions, existing workflow consumers, one selected
release and a deterministic bounded public content inventory.

## Constitution Check

Pre-research and post-design: PASS. Owner scope and merge boundary remain clear.
Shared runtime, credential and tenancy contracts are unchanged. No provenance,
source, security or qualification gate is removed. Issues, milestone and Project
track delivery. Public docs contain operational contracts, not slice evidence.
The 660-second budget remains a measured gate. Captured product inputs do not
change, so existing screenshot provenance remains valid. UTF-8 without BOM/LF.

## Project Structure

- scripts/bootstrap-actions.py and scripts/bootstrap-actions.lock.json
- scripts/http_read.py and tests/http_read_test.py for shared bounded HTTPS
- scripts/public-verification.py, scripts/release.py and scripts/pages.py
- tests/bootstrap_actions_test.py and tests/public_verification_test.py
- tests/bootstrap-actions.test.mjs and affected .github/workflows/*.yml
- docs/v1.0.1/releases.md and this internal planning directory

## Decisions and sequencing

1. Keep exact action revisions and runner-managed main/post hooks. A build
   wrapper cannot recover preparation-time 404s. Avoid rewriting upstream setup
   or introducing a new remote retry action.
2. Lock archive SHA256 as well as revision. Compression-only upstream changes
   fail closed and require review. Document manual lock maintenance because
   Dependabot no longer discovers these as remote workflow uses.
3. Stage complete regular action trees atomically, retaining package.json and
   bundled helpers. Retry known-public archive 404, 429, selected 5xx and network
   errors only. Integrity, authentication and archive errors terminate.
4. Reuse original candidate bytes. Public binary proof is API digest metadata;
   live docs content proof downloads and hashes responses. Never describe this
   as repeating every public product-binary download.
5. Derive a deterministic page/schema/navigation/resource inventory from the
   original archive. Verify exact Pages branch/build before and after HTTP.
   Generic HTTPS verification provides content proof without Pages identity.
   Keep live network checks out of ordinary PR CI; use injected fixtures.
6. Write adversarial tests before implementation. Integrate workflows and docs,
   run local parity and read-only live checks, converge, commit and push the PR.
   Resolve at most two review rounds, then stop for owner merge.

## Complexity Tracking

No constitution violations. Two standard-library tools reuse existing release
validation rather than introducing a service or workflow rerun daemon.
