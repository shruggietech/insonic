# S061 validation

1. Select `specs/061-backup-capacity-recovery` through the local Spec Kit feature selector. Confirm the branch and #200 mapping.
2. Use the pinned Go 1.27.1 compiler with CGO and the selected host compiler. Run `go test ./internal/contracts ./internal/artifact ./internal/backup ./cmd/insonic` and affected `go vet` checks. Native tests inject bounded errors only in owned fixtures; they launch no GUI.
3. Run `python -m unittest discover -s tests -p '*backup*test.py'`, affected CLI qualification and release-validator tests, `npm run check` and `npm test`.
4. Run affected documentation/site checks. Source/package CI executes the actual obstruction campaign through the selected CLI under the existing original qualification deadline.
5. Require strict complete receipts and green exact-head CI, resolve all actionable external review findings within two Codex rounds, retain/read back compact evidence, and remove only verified owned staging.
6. Leave the official PR open with auto-merge disabled for owner review. No release/tag is authorized.

## Local verification (2026-10-31)

Failing-first regressions reproduced lost native capacity classification and an actual confined SQLite page-limit restore failure. Repairs pass five affected Go packages and vet; the later portable page-limit test also proves rollback, identical retry and retained fencing after lease release. All 219 root Node tests, 50 CLI Python tests, 98 release/history Python tests, 12 timing tests and 18 site tests pass. Public/offline documentation builds and export checks pass for 169 pages. Root text/schema checks and Git whitespace checks pass.

The focused headless development CLI campaign passed all three actual filesystem-obstruction cases, source withdrawal, exact retry and restored-byte checks, with natural runtime exit and fixture removal. This development image was built before commit; exact-source Windows/Linux source/package acceptance remains the CI gate. Native injected faults, confined database page limits and actual filesystem obstructions do not establish physical exhaustion or live provider quota behavior. Local GUI, GPU/acoustic and attended installed-product acceptance were not run.

The initial local root run exposed the new timer owner missing from the maintained inventory and a non-home scratch prerequisite for this checkout. Both are resolved; the complete root suite passes with a verified owned temporary parent outside the user profile. No product deadline was changed.

## Initial hosted failure and correction

[Implementation CI 41173748041](https://github.com/shruggietech/insonic/actions/runs/41173748041) failed (19 success, 19 skipped, 10 failed including aggregate failures). All six source/package CLI contexts reached the new backup campaign but failed restore retry. Existing core, backend and native desktop checks passed. The [failed attempt retention](https://github.com/shruggietech/insonic/pull/201#issuecomment-6539874594) was independently read back, decompressed and hash-checked before replacement CI.

Fixture-created private roots were the gap: intermediate Unix permissions and elevated Windows ownership do not establish product admission. The corrected fixture creates each owned ancestor privately, uses the already product-owned fresh workspace control directory as its explicitly configured destination filesystem root, and lets restore create protected progress through a preceding actual publication refusal. It avoids starting a destination application owner, which can seed a catalog that must remain empty for restore. A resolved-parent regression also covers equivalent Windows short-path spelling. The retained receipt records the selected destination root; product private-directory checks and deadlines are unchanged. Corrected hosted acceptance remains pending.

The corrected headless development campaign passes all three cases with natural runtime exit and owned fixture removal. CLI Python checks now run 52 tests (51 pass and the Unix-mode assertion is explicitly skipped on Windows); root text/schema checks pass. Hosted Windows/Linux confirmation is still required.
