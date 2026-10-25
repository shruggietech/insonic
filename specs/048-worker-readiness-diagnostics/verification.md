# S048 verification

Scope: bounded child #174 of #147, [PR #175](https://github.com/shruggietech/insonic/pull/175). Implementation is qualified and unmerged; the owner performs final review and merge.

## Implementation and local checks

- Spec Kit specify/clarify, research/design, tasks/analyze, implement and convergence completed. Ten FRs, four success criteria, eight acceptance scenarios, eleven design decisions and five constitution principles were assessed. Final convergence found no remaining code gaps.
- Fixed bounded stage/reason/count/timing diagnostics reach shared JSON, CLI, desktop, failed setup and partial readiness. Original 60-second worker/readiness and 65-second IPC limits, integrity checks, legacy receipts, routes and durable-work ownership remain authoritative.
- Root integrity/schema checks passed (25 schemas, 22 contracts, 30 examples), with 194 Node tests. Full Go tests/vet passed; the final receipt-consistency repair passed complete workers/app suites (0.219 s / 26.904 s) and vet.
- Full Python suite: 603 cases, two declared skips. Final campaign harness regressions: 31 passed. Desktop: 109 tests/types/build. Site: 17 tests/build and 111-page validation. Text checks cover UTF-8 without BOM, LF and mojibake.
- Controlled negative tests reproduced catalog-history collision, Arcade fixture cutoff, missing runtime-startup join and contradictory receipt success before their fixes. Other cases cover malformed/duplicate/oversized/reordered checkpoints, actual child disconnect, cancellation during hashing/reuse, earlier deadline, legacy receipt compatibility, changed bytes and redaction.

## Source-bound real CPU acceptance

[Retained public receipt and terminal CI audit](https://github.com/shruggietech/insonic/pull/175#issuecomment-6398236660). Complete package/source/harness revision: `278288805eabc1334315ff62be82ecc3ba42500f` (test-merge of implementation `04247d44f4ca8ce90e7fdb05d91fd6ab4023930d`). Harness remained clean and unchanged.

- All 15 required stages passed, including actual transcription/diarization on six cases, two profile enrollments, four matching cases, query/timeline/playback, retrieval, accepted-data restore and owned cleanup. Both correlated native startup reports passed. Receipt independently revalidated against the frozen corpus.
- Preparation: 186.578 s; qualification: 515.141 s; combined: 701.719 s. Runtime released and temporary directory removed.
- CLI archive SHA-256: `516dbe96540f232fb90800bc32ae898dc5e6fdf3b6a982edc9e247e1856515b0`; inventory: `a0d80c71cfd7dea0f1aa49eb40ca349cdbb4feac393ceeca18b8630ce4dbafc3`; complete receipt: `ff0d0200ca4cf9e9529eed03adbb4de7fa2cba752b094b300f7f99880debb15e`. Downloaded artifact digests, byte counts and confined extraction were checked before execution.

| Role | Fresh snapshot native/child ms | Inspect 1 native/child ms | Inspect 2 native/child ms |
| --- | --- | --- | --- |
| Recognition | 3105 / 2594 | 990 / 515 | 956 / 516 |
| Diarization | 31260 / 29891 | 14529 / 13015 | 15116 / 13750 |

Each original probe budget was 60000 ms. Repeated inspections use the same immutable environment; snapshot and inspect modes do different work. No OS-cache flush is claimed.

## CI, reviews and limitations

- Implementation CI [40195538653](https://github.com/shruggietech/insonic/actions/runs/40195538653), attempt 1: terminal success, 30 successful jobs and 18 expected skips. Independent audit accepted every elected result: 241 s preparation + 683 s qualification = 924 s combined. Final record-only completion receives its own applicable CI before handoff; product and harness inputs remain unchanged.
- Both authorized Codex rounds completed without findings, on `086eb92` and `905f88b`. No individual review threads remain. Subsequent repairs have regression, actual CPU and CI evidence and await owner final review. No third round or separate security-review completion is claimed.
- Earlier failed local package preflight and real attempts remain retained in PR comments. CI `40188824899` failed the Arcade fixture and elected acceptance (252 + 708 = 960 s); that failure is not replaced by later success. Superseded run `40194446732` was cancelled after the final repair push and is not terminal acceptance evidence.
- The small public corpus produced 12/252 scenario-weighted word errors (4.7619%); five speaker decisions were two correct matches, one unknown, one correct unknown and one mixed reference, with zero incorrect. This does not establish general accuracy or DER.
- Restore verified accepted authority, queries, timeline, playback and fetched model bytes. Restored transcription/diarization/training prerequisites reported needs_attention/unusable; fresh restored inference was not tested. This is retained on broad #147, which stays open. Historical S046 causality remains unproven.
- Campaign receipts are retained on GitHub. The owned qualification checkout is archived; owned campaign scratch and rejected local package outputs are removed after identity/path checks. No owner profile/library, installed product, release or merge was changed.
