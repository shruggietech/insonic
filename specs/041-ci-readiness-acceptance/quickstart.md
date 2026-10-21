# Validation

1. Run `node --test tests/ci-readiness.test.mjs tests/ci-acceptance.test.mjs tests/ci-scope.test.mjs`. Expect a cold producer above 180 seconds to delay consumers without enlarging the transfer helper limit.
2. Run `npm run check`, `npm test` and site tests/build for the development contract change. Verify UTF-8/LF and no mojibake.
3. After authorized publication, retain one deliberately absent exact-key cold run and one warm run, without deleting unrelated caches. Verify all elected checks and current-run transfers, then audit final timing with `node scripts/ci-acceptance.mjs --audit RECEIPT.json` using read-only Actions access and the digest-verified gate receipt downloaded from that exact attempt.
4. Apply the existing bootstrap required-check configuration only after the published aggregate exists. Verify exact post-merge main separately before closing #161. The 2026-10-21 bounded fixture calibration below is authorized before handoff; broader native incident recovery remains under #154.

## Isolated hosted campaign

1. After the aggregate is published, verify main protection requires `foundation`, `docs` and `elected acceptance` from the GitHub Actions app, with strict mode and all other settings preserved. Retain before/after API records.
2. Dispatch `ci.yml` on the final S041 revision with `cold_media_source=true`. Record the selected source commit and unique run ID before interpreting results. The first attempt must show cold mode on both source chains and absent complete/dependency/format compiled cache keys. Original archive caches may be reused only with existing pin rehashing.
3. Monitor the complete attempt. Retain producer logs, source/toolchain artifacts, exact-attempt gate digest and terminal audit. Qualification has a 660-second advisory target and a 900-second hard limit after verified preparation, with the original 15-second provisional finalization reserve.
4. Rerun the entire successful campaign run with `gh run rerun RUN_ID` (not failed jobs only). The same namespace and source are retained, while current-attempt transfers are regenerated. Verify warm source mode on both platforms and repeat the terminal audit.
5. Retain final-head PR merge-source CI separately. PR #162 is historically merged; the timing-policy continuation has explicit squash-merge authorization after required checks pass. Verify exact-main acceptance separately. Cold/warm closure does not block the policy merge or subsequent product slice.

## Historical bounded Windows window-join calibration (PR #162)

1. Use the current instrumented ordinary CI assembly, retaining its run/attempt and tested merge revision. Do not treat this baseline as final acceptance.
2. Dispatch the existing installer-invocation diagnostic workflow in window-calibration mode with that explicit run/attempt/revision. It verifies the immutable assembly and runs the same twelve-case native exit matrix with two seconds, then ten seconds, on one runner using separate equivalent fixture roots.
3. Retain both arms and every successful/failed child observation. Keep the original outer bounds. A baseline failure remains a failure; both sub-two-second passes are inconclusive about the prior incident.
4. Adopt the explicit policy or apply a directly demonstrated bounded repair, run local gates, and freeze the final candidate before ordinary PR CI. Keep all native and package checks, audit terminal timing and stop before merge. Do not repeat compiled-media campaigns solely for fixture observations/allowance; retain exact-final-head campaign closure work separately.
