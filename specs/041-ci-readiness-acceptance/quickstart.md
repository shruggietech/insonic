# Validation

1. Run `node --test tests/ci-readiness.test.mjs tests/ci-acceptance.test.mjs tests/ci-scope.test.mjs`. Expect a cold producer above 180 seconds to delay consumers without enlarging the transfer helper limit.
2. Run `npm run check`, `npm test` and site tests/build for the development contract change. Verify UTF-8/LF and no mojibake.
3. After authorized publication, retain one deliberately absent exact-key cold run and one warm run, without deleting unrelated caches. Verify all elected checks and current-run transfers, then audit final timing with `node scripts/ci-acceptance.mjs --audit RECEIPT.json` using read-only Actions access and the digest-verified gate receipt downloaded from that exact attempt.
4. Apply the existing bootstrap required-check configuration only after the published aggregate exists. Verify exact post-merge main separately before closing #161. Continue native incident baseline/recovery under #154 afterward.
