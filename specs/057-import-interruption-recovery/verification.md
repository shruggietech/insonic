# Verification: Import interruption recovery

**Date**: 2026-10-28
**Status**: Local implementation verified; PR CI, six-context acceptance and external reviews pending.

## Implemented behavior

Current canonical import binds each new capture report to an exact canonical publication ID/digest/size. Complete surviving pairs are verified against original source bytes and reused after process loss. Equal-length changed sources and corrupt report/canonical bytes refuse admission. Partial or older unbound candidates retain recomputation; observed ordinary failures, uncertain accepted commits and current work fences retain their existing behavior.

Shared source/package CLI qualification uses external Windows RW oplocks or Linux file leases and controlled hash-pinned media tools. It independently reads available publications, absent admission, exact runtime/work authority, bytes, metadata/date provenance and prior work receipts. Only the owned runtime is killed, while the observed source-open barrier is held. A distinct replacement must recover or refuse as expected, and final runtime/barrier/path cleanup is required. Current/unknown release receipts require this evidence; only four exact original v1.0.2 receipt digests may lack the new field.

## Performed checks

- Failing-first current-admission regressions reproduced replaced capture candidates and acceptance of changed/corrupt input before the correction. Corrected native recovery, corruption, partial/unbound candidate, stale-work fencing, ordinary-failure retirement and uncertain-commit tests pass.
- Full native library suite passed. All Go packages passed with pinned native media fixtures. An initial concurrent Go package scan raced a Python setup test's removal of a build fixture; serial Go verification after that suite completed passed. No product assertion was waived.
- Full Python suite: 693 tests, three existing declared skips. A subsequent focused six-test recovery suite covers the added exact-date predicate; five actual barrier/negative tests pass with ResourceWarning treated as an error.
- `npm run check`: foundation, encoding/links, 25 schemas, 22 contracts and 30 examples passed.
- `npm test`: 219/219 passed. Site: 18/18 tests and full public/offline export with 140 checked pages passed.
- Local Windows actual-process campaign passed same-source recovery, equal-length changed-source refusal and corrupt-report refusal. Generations advance 1 -> 2 with distinct owned runtime sessions, prior attempt receipts retained, independent bytes/catalog checks and fixture cleanup. Final current-image receipt is retained outside workflow storage at PR delivery.
- Four original v1.0.2 package receipts from published manifest SHA-256 `02a55b6ec51855881a1fc7592b002c5d3ce5d5dce0bf09985094da27beee3449` pass all existing qualification checks. Changed/unknown receipts cannot omit recovery evidence. The compact frozen fixture contains original receipt data, not reconstructed evidence.
- Spec Kit specify/clarify/plan/tasks/analyze/implement/converge: all nine FRs, four SCs and six story cases mapped. Built-in checklist 12/12; no extra code gap or constitution exception found.

## Delivery and limits

S057 milestone 41 and Project linkage exist; #147 remains an open umbrella. Push and official PR creation are explicitly authorized. Exact-head CI and its complete Windows/Linux source/CLI/desktop-package receipt matrix, independent terminal timing/artifact audit, and up to two Codex rounds remain under T014. Stop before merging.

Local source-image execution does not establish Linux/package, owner-installed, released, acoustic or physical power-loss behavior. Existing aggregate timing policy and ordinary production deadlines/leases remain unchanged. No release, tag, deployment or owner data changes are authorized or claimed.
