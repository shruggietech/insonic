# Validation guide

1. Confirm the current branch is codex/040-webview-lifecycle-repair and the baseline is verified S039 squash c9ed48d. Use only the verified hidden runner for non-Git commands on Windows.
2. Run the native Controller3 BOOL regression against unchanged v1.0.22, retaining the observed by-value ABI failures for false and true in the actual production wrapper.
3. Acquire/check authoritative v1.0.23, preserve maintained lifecycle patches and repeat the same regression. Run the existing message-only ownership suite, including race detection.
4. Run root npm run check, npm test, affected Python/native/product checks and site check/test/build. Verify exact maintained source bytes and notices.
5. Before a hosted experiment, complete its prediction/control/bounds in experiments.md. Reuse the disposable source workflow, stop at the first failure or fixed budget, and classify response completion and process-release evidence. No visible local GUI or ambient input is permitted.
6. Validate final cold/warm, with/without-library and all native actions, then ordinary source/package/candidate checks on the exact repair head within 660 seconds. Update verification.md and #154 with evidence and unverified limits.

A finite successful sample alone does not close #154. Forced cleanup never qualifies a launch. No merge, product installation, release or site publication is part of this guide.
