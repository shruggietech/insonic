# S029 post-merge publication recovery

## Demonstrated failure (2026-10-15 UTC)

PR #99 was owner-squash-merged as 136f5c0307a6fce00ba8240ef69c907a944cd8c0. Main CI 38683979572 passed 18 elected jobs in 593 seconds. Immutable v1.0.1 points to that exact reviewed source.

Six release attempts failed before checkout when GitHub could not download the valid pinned setup-go action. Run 38686403094 attempt 2 subsequently passed every native runtime/package gate, release-target docs/UI gate and complete candidate collector. Its publisher refused the 93-check inventory because cancelled/failed suite 104829245084 from the older run remains attached to the same source. No draft upload or official publication occurred. The live site remains v1.0.0 at source 61cd2c71ff15a5606dca1597fc8782e2b8f64dc1.

Starting a fresh run exposed the verifier limitation. Existing source checks are latest per suite, rather than one latest suite for each workflow. The earlier update saying publication had cleared exact-source checks was premature; the publisher entered that command and then correctly failed before uploads.

## Decision and contract

The existing plan permits a proportional scripts/release.py correction after demonstrated failure. Correct verified supersession in trusted-main tooling. Preserve failed check history, original v1.0.1 source/tag, all runtime/security/tenancy/package gates, complete candidate validation, source/tag ancestry, remote draft hashes and original-archive promotion. No product or frozen-version documentation changes are needed.

Only a verified current main-branch workflow_dispatch of this repository release.yml establishes supersession. An older run must have a canonical GitHub job details URL, lower run ID, matching independently fetched repository/main branch/release workflow ID, exact candidate revision, matching check suite and completed terminal outcome. Current/historical candidate dispatches both work. Unknown, active, newer, other-branch/repository/source/workflow and security/CI checks remain gates. Cache one read per prior run. Outside a verified workflow context no suite is omitted.

The publisher invokes this selection only after trusted-main, complete candidate/package/documentation, version/source/tag and inventory validation. Required foundation/docs and all remaining external checks must still be green before any publication write.

## Spec Kit analysis

Converge appended T020-T022 against FR-004/FR-006/FR-007 and SC-003. Read-only analyze found complete mapping to the existing release outcome, no constitution conflict, no tag change and no missing security acceptance. Implementation is authorized by the demonstrated-failure exception in plan.md. Original two review rounds are exhausted; authorizing a recovery PR review cycle is a separate pending operator decision. No additional review is triggered.

## Verification

- Red first: the two successful-supersession scenarios failed against the original publisher with exact-source automated checks are not green.
- Focused publisher suite: 49 tests pass, including current and historical source, terminal/identity/URL rejection, unrelated CI/security failure, missing required checks, pagination and no-publication-write assertions.
- Read-only replay of actual 93 checks: retains all 32 main-CI checks, with both required names green; omits only verified release suites 104829245084 and 104833265806. Zero network writes.
- Release-source hosted UI: 225 HTML files, 70 width/theme combinations, 18 scenarios, 80.4 seconds, zero external traffic/errors and verified owned cleanup. Mobile light/dark screenshots inspected. Native MSI/account/UI checks ran only on disposable hosted Windows.
- Full local checks passed: npm run check (1225 text files, 25 schemas, 22 contracts and 30 examples), 103 root tests, 10 site tests and 250 Python tests. The final focused 49-test suite and actual-check replay also passed after pagination/current-context refinements.
- Source candidate artifact: run 38686403094 attempt 2, release-candidate-2 (artifact 11950449051, ZIP digest a0e62731db0d5ba387046b8c33d34b841e85dfb6cfc8b69ba27fb1617809ffcb). All four native archives, MSI, both documentation bundles and candidate metadata passed collection. No official release asset exists.

## Delivery boundary

This patch is a recovery proposal, not an official release. T017-T019 and #97 remain open. The original source/tag and qualified candidate are retained. No failed run/check is deleted or relabeled. Publishing requires reviewed trusted-main recovery and the existing publication/deployment gates. Obtain a new PR review-cycle authorization before opening a recovery PR; never request a third review on #99.
