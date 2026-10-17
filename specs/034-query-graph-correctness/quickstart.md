# Validation: S034

1. Use the repository's hidden Windows runner with prepared native tool environment.
2. Run targeted Go regressions for internal/explore, internal/contracts, internal/library, internal/graph, internal/app and cmd/insonic. Exhaust star/dense graph cursors at 1, 2, 100 and 500 and search beyond 500 rows; compare exact node/edge identity unions.
3. Exercise timezone forms and DST membership using bundled data, save the schema-generated query example through dispatch, and reject duplicate native JSON before execution.
4. Run native grammar tests covering quotes/comments/terminators, forbidden effects and functions, then desktop type/tests for result accumulation.
5. Run npm run check, npm test, site tests/build and relevant product checks. Record actual executed checks and boundaries in verification.md.
6. Push, create the linked PR, inspect exact-head CI and handle at most two review rounds. Leave merge to the owner.
