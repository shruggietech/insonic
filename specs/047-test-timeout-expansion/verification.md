# Verification: S047

## Local checks (2026-10-24)

- Root integrity/schema checks passed: 1,605 text files, 25 schemas, 22 contracts, 30 examples, version 1.0.1. Root Node tests passed 194/194.
- Initial full Python discovery ran 583 cases with one existing platform skip in 412.522 seconds, using pinned Go 1.27.1 and the Windows-supported C locale. After repairs, qualification ran 22 cases (one POSIX-only Windows skip), package tests passed 28/28 and the runtime replacement fixture passed 20 Windows repetitions.
- Complete Go tests and vet passed. Final controlled-clock process fixtures passed their full package suite in 4.971 seconds; production clocks and cancellation/generation fences remain intact.
- Frontend: 108 tests, TypeScript and build passed. Site: 17 tests, production build and 111-page link/anchor checks passed.
- Current hosted core Python discovery ran 587 cases per platform: Windows 584 passed / 3 skipped, Linux 554 passed / 33 skipped. Platform/environment skips are explicit; Linux executed the new physical-descriptor regression.

## Repairs and review

The initial full Go run exposed short healthy fixture clocks. Runtime owner fixtures now prepare catalog prerequisites before the unchanged product startup clock. Direct model fixtures retain lease headroom and generation/cancellation fencing. Process-containment fixtures use actual readiness barriers and controlled expiry clocks.

Codex round one found two P1 gaps. All 12 sparse workflow layouts now include the timing loader and policy, with isolated execution regressions. Windows producer waits now allow 3,600 seconds under the original phase deadline; controlled joins accept exact valid proofs at 3,005 seconds with reserve remaining. Both individual threads are resolved. The second and final authorized round reported no major issues on 3a079e4. Later CI repairs are included for owner review; no third trigger was sent.

PR CI 40127126725 and isolated cold run 40127360673 on 3a079e4 failed functional gates despite valid timing accounting (1,495 + 707 = 2,202 seconds and 1,656 + 792 = 2,448 seconds). Linux source/package help rejected ambiguous receipts after successful child exits. The cold Windows runtime fixture also failed directory initialization. No shared cause or historical HRESULT is inferred.

Diagnostic 6c45102 exposed 17 progress JSON records plus the valid receipt on stdout. Ubuntu xvfb-run merged client stderr into stdout. A fixed argument-preserving wrapper now restores the original stderr before all six source/package help, GUI and Exit clients. Strict cardinality remains enforced. Empty, duplicate, mismatching and oversized outputs still reject; receipt encoding errors now fail the child.

The runtime replacement fixture now owns an isolated profile and observes early Serve failure immediately. Its first long test-named profile hit Linux's existing 100-byte endpoint guard; 46d9265 uses a short owned temporary profile with cleanup. Existing-directory permissions, product initialization and endpoint limits are unchanged. The shared elevated-profile creation window is consistent with the original cold error but remains unproven as its cause.

## Accepted stable implementation

All rows use head 46d926577fb37777a108fc98aedf46aef1f9bf93. The PR run tested merge revision fbb9cf8b202376f1b8835b9f525a73bc5f6149d3. Terminal metadata was fetched independently; artifact size and SHA-256 were verified before bounded ZIP parsing. Required functional and timing evaluations passed.

| Run / attempt | Source paths | Preparation seconds | Qualification seconds | Combined seconds | Windows help owned seconds |
| --- | --- | ---: | ---: | ---: | ---: |
| PR 40132111337 / 1 | warm / warm | 445 | 643 | 1088 | 8.187 |
| Campaign 40132110240 / 1 | cold / cold | 1705 | 739 | 2444 | 18.015 |
| Campaign 40132110240 / 2 | warm / warm | 263 | 694 | 957 | 8.140 |

The campaign reran every job on the same run ID, source and isolated compiled-cache namespace. Cold had 41 successful / 11 intentionally skipped jobs; warm had 30 successful / 18 intentionally skipped jobs. Help retained all 17 monotonic progress records, natural owned-process release and application-return classification within the former 35-second bound. This is finite acceptance evidence, not a causal closure of #154.

Acceptance artifact identities:

- PR: 12576061396, 986 bytes, SHA-256 fefbd54fa0fa5b9a496a99587f4f91814d1770f59da6f3baf2de8c3318583149.
- Cold: 12576960746, 958 bytes, SHA-256 40acfdb241904130bef5314b116bb1dc6c2c5e9496da5e2d3778cb416b8cf590.
- Warm: 12576554820, 952 bytes, SHA-256 2bb3ef92acfc2b72f480d7bd574ca53a62ddeb46426c71e441704b98ae4e429d.

## Evidence retention and limits

After warm rerun began, cold artifact IDs returned HTTP 404 and the run artifact list contained attempt 2 only. The successful independent cold audit had completed beforehand. Its complete decoded help receipt, all progress, source/run/attempt identity, artifact sizes/digests and terminal timings are retained in the [cold receipt](https://github.com/shruggietech/insonic/pull/173#issuecomment-6389944708). Original cold ZIP bytes are unavailable and are not represented as retained.

The completed cold candidate job also retains its [package manifest and input identities](https://github.com/shruggietech/insonic/pull/173#issuecomment-6389966814): release-manifest.json is 7,429,982 bytes, SHA-256 3d607ccdf6f736d135b151183f1f1e2362f0dea6dc7dae1f9da62f4aa691cf24, binding four packages and eight assets. These are job-reported identities, distinct from independently downloaded proof bytes.

The [warm receipt](https://github.com/shruggietech/insonic/pull/173#issuecomment-6390049943) retains its separate audit and three-asset byte identity. Its archive proof 12576594460 is 2,895,112 bytes, SHA-256 73a01ede78bafa7ee9fe2d996734e94824dad4602193eedeeeac77dfcc138254. The internal proof file inventory and archive/MSI/setup bindings were independently checked. Agent/quickstart guidance now places compact receipt retention before rerun; the naming regression claims collision avoidance, not platform retention.

## Convergence and handoff

Final Spec Kit convergence checked 13 functional requirements, five success criteria, ten acceptance scenarios, eight design decisions and the five core constitution principles plus relevant repository constraints. No remaining implementation finding was identified; the assessment appended no empty phase. All repair tasks are complete.

The verification-record commit updates internal planning, agent retention guidance and a test description/comment only. Stable implementation qualification above remains identified by its actual source. That final PR head receives its own full CI and terminal audit before owner handoff; its receipt is retained in the PR discussion. No merge, release, installation or owner-desktop GUI/input operation occurred. Merged-main acceptance remains a later owner-approved step; #154 and #147 retain their separate scopes.
