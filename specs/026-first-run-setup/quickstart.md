# S026 validation guide

1. Build a complete current CLI/desktop package, including native companions.
2. Run `insonic setup location --json`; verify the proposed location is editable and no destination is created.
3. Save a bootstrap draft using `setup draft save --input FILE --request-id UUID --json`, then read `setup draft show --json`. Retry the same immutable input/UUID and compare normalized workspace/profile IDs.
4. Run `setup provision` to elect those backends, then `--workspace LIBRARY setup show --json`. No model download is required for defaults.
5. Save a typed journey using the observed revision, repeat the request after restart, then try a stale competing input and inspect conflict preservation.
6. In desktop, use optional Setup, skip processing/services and open normal import. Reopen from Settings; elected preparation stays visible in Jobs after defer.
7. Run configured check/test/product/site commands. Verification.md records exact results and unverified platform behavior.
