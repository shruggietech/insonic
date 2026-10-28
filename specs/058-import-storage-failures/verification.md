# Verification: Local import storage-failure recovery

**Date**: 2026-10-28
**Status**: Implementation/local acceptance complete; authorized PR, exact-head CI and bounded external reviews remain in T014.

## Changed behavior

Native local capacity/quota failures retain the existing insufficient_space code through wrapped errors. Staging requires successful flush and rewind before returning readable authority; failed preparation retains close/remove errors in its internal chain and returns a redacted error. Filesystem publication propagates failed stage removal without deleting completed object bytes. Publication/work fences, source hashing and product deadlines are unchanged.

## Performed verification

- Failing-first native capacity regressions reproduced generic unavailable results for partial ENOSPC writes; corrected classification passes with source bytes unchanged. Native partial short writes, flush/rewind/cleanup and Windows disk/quota/access-denial distinctions pass.
- Native filesystem final-persistence and cleanup failures preserve completed bytes and pending recovery authority. Four canonical/report pre-publication and uncertain-completion cases preserve an accepted sentinel and original source; cleared failures reconcile and admit one verified new recording with metadata/date provenance and idempotent replay.
- All Go packages pass under the maintained expanded test allowance; go vet ./... passes. Full artifact/library suites pass.
- Full Python suite after Windows alias correction: 700 tests, three declared skips. An initial strict Windows ExifTool stderr check rejected inherited C.UTF-8 locale warnings despite correct exit/version. The focused eight-test regression and complete suite pass with supported C locale; assertions/source were unchanged for that host prerequisite.
- npm run check passed, including schema/link/encoding/brand integrity. npm test: 219/219. Site: 18/18 tests, full public/offline build and 140 checked pages.
- Current local Windows source-image storage campaign passes actual filesystem canonical-publication refusal, source/sentinel preservation, exact blocker removal, distinct ordinary re-import, canonical/report verification, metadata/date provenance, replay and natural runtime/fixture cleanup. Image SHA-256: db8f9a95b826f1a80c8ee2c0c47655bc761be35e5cb1d44b7c4cafb49b9fbaea. This image came from the worktree; Git-bound source/package acceptance remains CI work.
- Initial [CI 40768209479](https://github.com/shruggietech/insonic/actions/runs/40768209479) rejected the new helper's comparison of resolved Windows paths with absolute original spelling. The corrected check anchors the exact derived object path under the resolved library root while retaining marker, inode, hash and link checks. Equivalent-parent and actual-escape regressions pass; a full actual Windows short-path campaign also passes with fixture removal. The rejected attempt's seven artifact ZIPs, three complete Linux contexts, source identity, terminal timing and primary failure are recoverably retained and read-back/hash verified in [the PR audit](https://github.com/shruggietech/insonic/pull/195#issuecomment-6480960682), before the corrected push.
- Strict storage receipt mutations and blocker ownership/confinement cases pass. Current/unknown package receipts require storage evidence; all four exact original v1.0.2 receipts retain their existing historical absence, with original v1.0.1 validation unchanged.
- Spec Kit specify/clarify/plan/tasks/analyze/implement and bounded implementation convergence pass: ten FRs, three buildable local SCs, seven acceptance scenarios and five constitution principles mapped; no unbuilt code gap or exception found. The fourth SC's delivery operations remain explicitly tracked in T014. No custom reviewer-owned checklist was needed; built-in quality is 12/12. Extension hooks are absent.

## Tracking and limits

Milestone 42 tracks this bounded increment; #147 stays open. Owner authorized push/official PR, CI fixes and up to two Codex review rounds, with a stop before owner merge. No merge, tag, release, deployment or owner-library change is performed.

Injected capacity/short-write/report/persistence failures and actual filesystem obstruction are separate evidence. Physical volume exhaustion, physical power loss, backup/remote quota behavior, fresh owner-installed behavior and new acoustic inference remain unverified. A service-level partial import result does not alone define durable work outcome: the actual application marks failed items as failed work. The campaign verifies a distinct ordinary re-import, not same-work public retry execution.
