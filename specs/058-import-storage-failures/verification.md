# Verification: Local import storage-failure recovery

**Date**: 2026-10-28
**Status**: Implementation, exact-source acceptance and both bounded reviews passed. Delivery record is ready for the final head check and owner review; no merge is performed.

## Changed behavior

Native local capacity/quota failures retain the existing insufficient_space code through wrapped errors. Staging requires successful flush and rewind before returning readable authority; failed preparation retains close/remove errors in its internal chain and returns a redacted error. Filesystem publication propagates failed stage removal without deleting completed object bytes. Publication/work fences, source hashing and product deadlines are unchanged.

## Performed verification

- Failing-first native capacity regressions reproduced generic unavailable results for partial ENOSPC writes; corrected classification passes with source bytes unchanged. Native partial short writes, flush/rewind/cleanup and Windows disk/quota/access-denial distinctions pass.
- Native filesystem final-persistence and cleanup failures preserve completed bytes and pending recovery authority. Four canonical/report pre-publication and uncertain-completion cases preserve an accepted sentinel and original source; cleared failures reconcile and admit one verified new recording with metadata/date provenance and idempotent replay.
- All Go packages pass under the maintained expanded test allowance; go vet ./... passes. Full artifact/library suites pass.
- Full Python suite after Windows alias correction: 700 tests, three declared skips. An initial strict Windows ExifTool stderr check rejected inherited C.UTF-8 locale warnings despite correct exit/version. The focused eight-test regression and complete suite pass with supported C locale; assertions/source were unchanged for that host prerequisite.
- npm run check passed, including schema/link/encoding/brand integrity. npm test: 219/219. Site: 18/18 tests, full public/offline build and 140 checked pages.
- Current local Windows source-image storage campaign passes actual filesystem canonical-publication refusal, source/sentinel preservation, exact blocker removal, distinct ordinary re-import, canonical/report verification, metadata/date provenance, replay and natural runtime/fixture cleanup. Image SHA-256: db8f9a95b826f1a80c8ee2c0c47655bc761be35e5cb1d44b7c4cafb49b9fbaea. This image came from the worktree; Git-bound source/package acceptance is established separately by CI below.
- Initial [CI 40768209479](https://github.com/shruggietech/insonic/actions/runs/40768209479) rejected the new helper's comparison of resolved Windows paths with absolute original spelling. The corrected check anchors the exact derived object path under the resolved library root while retaining marker, inode, hash and link checks. Equivalent-parent and actual-escape regressions pass; a full actual Windows short-path campaign also passes with fixture removal. The rejected attempt's seven artifact ZIPs, three complete Linux contexts, source identity, terminal timing and primary failure are recoverably retained and read-back/hash verified in [the PR audit](https://github.com/shruggietech/insonic/pull/195#issuecomment-6480960682), before the corrected push.
- Strict storage receipt mutations and blocker ownership/confinement cases pass. Current/unknown package receipts require storage evidence; all four exact original v1.0.2 receipts retain their existing historical absence, with original v1.0.1 validation unchanged.
- Spec Kit specify/clarify/plan/tasks/analyze/implement and bounded implementation convergence pass: ten FRs, three buildable local SCs, seven acceptance scenarios and five constitution principles mapped; no unbuilt code gap or exception found. The fourth SC's delivery operations are tracked in completed T014. Post-implementation convergence includes the complete delivery criteria after accepted CI and reviews, with no remaining code gap. No custom reviewer-owned checklist was needed; built-in quality is 12/12. Extension hooks are absent.

## Exact-source CI and independent retention

[CI 40771163696](https://github.com/shruggietech/insonic/actions/runs/40771163696), attempt 1, passed with 30 successful and 18 intentionally skipped jobs. Head 653a77a2ac0c7000e780cfc9644fb9d68907bc16 and tested merge 5a78e841d4c1ed8efbbb22c5e90eee07ee12731f share tracked tree fb66e7c127d8102614e2e5b44095b3e1dcdbd3a5. Required foundation, docs and elected acceptance pass. Preparation: 276 seconds; qualification: 741 seconds; combined: 1017 seconds (within the 1800-second target and 3600-second hard limit).

The independent audit verified all seven original artifact ZIP sizes/SHA-256 digests, exact run/head/attempt, six Windows/Linux source and CLI/desktop package contexts, package source/archive/inventory bindings and all three maintained receipt validators. Counts: 48 terminal, 18 interruption-recovery and six actual filesystem storage-failure cases. All elected results and terminal timing pass. The complete compact audit, receipts, both review rounds and separately attributed local diagnostic are [retained in the PR](https://github.com/shruggietech/insonic/pull/195#issuecomment-6481314380), with read-back/gzip/byte/hash verification (JSON SHA-256 84d4da6194d7aab7f9ce6130fa84fdbdf80afec8dae9701da104423b4d93dd89). The rejected first attempt is retained separately above.

## External reviews

Automatic round one completed without findings at ae6c2fd1443ee620d503c24719e0f3aabd11c93d. The authorized second/final manual round completed without findings at 653a77a2ac0c7000e780cfc9644fb9d68907bc16 at 2026-10-28T21:23:09Z; [the bot receipt](https://github.com/shruggietech/insonic/pull/195#issuecomment-6481017643) and fresh thumbs-up corroborate completion. Individual review threads and formal reviews are empty. No third round is requested. Security review was not requested, so no security-bot approval is claimed.

## Tracking and limits

Milestone 42 tracks this bounded increment; #147 stays open. Owner authorized push/official PR, CI fixes and up to two Codex review rounds, with a stop before owner merge. No merge, tag, release, deployment or owner-library change is performed.

Injected capacity/short-write/report/persistence failures and actual filesystem obstruction are separate evidence. Physical volume exhaustion, physical power loss, backup/remote quota behavior, fresh owner-installed behavior and new acoustic inference remain unverified. A service-level partial import result does not alone define durable work outcome: the actual application marks failed items as failed work. The campaign verifies a distinct ordinary re-import, not same-work public retry execution.
