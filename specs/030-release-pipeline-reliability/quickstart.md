# Validation guide

1. Run focused Python bootstrap/public-verification suites. Recovery,
   exhaustion, corruption, path confinement and read-only fixtures must pass.
2. Run npm run check, npm test, complete Python tests, site tests and site build.
3. Acquire both locked actions into new ignored destinations; inspect full
   main/post metadata. Do not run native product installers on the owner host.
4. Use the existing published-candidate command and selected historical source
   to retrieve original assets. Run verify-public against existing v1.0.1 and
   exact Pages commit; verify proof levels and absence of external writes.
5. Push the official PR. Verify supported Windows/Linux and cache post hooks on
   the final head. Record CI turnaround and resolve all review threads within
   at most two review rounds.
6. Stop for owner final review and merge. PR success does not prove the later
   squash commit's main-branch checks.
