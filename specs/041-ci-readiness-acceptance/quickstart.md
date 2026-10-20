# Validation

1. Run `node --test tests/ci-readiness.test.mjs tests/ci-acceptance.test.mjs tests/ci-scope.test.mjs`. Expect a cold producer above 180 seconds to delay consumers without enlarging the transfer helper limit.
2. Run `npm run check`, `npm test` and site tests/build for the development contract change. Verify UTF-8/LF and no mojibake.
3. After authorized publication, retain one deliberately absent exact-key cold run and one warm run, without deleting unrelated caches. Verify all elected checks and current-run transfers, then audit final timing with `node scripts/ci-acceptance.mjs --audit RECEIPT.json` using read-only Actions access and the digest-verified gate receipt downloaded from that exact attempt.
4. Apply the existing bootstrap required-check configuration only after the published aggregate exists. Verify exact post-merge main separately before closing #161. Continue native incident baseline/recovery under #154 afterward.

## Isolated hosted campaign

1. After the aggregate is published, verify main protection requires `foundation`, `docs` and `elected acceptance` from the GitHub Actions app, with strict mode and all other settings preserved. Retain before/after API records.
2. Dispatch `ci.yml` on the reviewed feature branch with `cold_media_source=true`. Record the selected source commit and unique run ID before interpreting results. The first attempt must show cold mode on both source chains and absent complete/dependency/format compiled cache keys. Original archive caches may be reused only with existing pin rehashing.
3. Monitor the complete attempt. Retain producer logs, source/toolchain artifacts, exact-attempt gate digest and terminal audit. Qualification still has its original 660-second budget after verified preparation.
4. Rerun the entire successful campaign run with `gh run rerun RUN_ID` (not failed jobs only). The same namespace and source are retained, while current-attempt transfers are regenerated. Verify warm source mode on both platforms and repeat the terminal audit.
5. Retain final-head PR merge-source CI separately. Leave PR #162 open and auto-merge disabled; post-merge main evidence remains pending.
