# S064 verification

## Scope and authority

S064 implements #217-#219 on `codex/064-transcript-coverage-repair`. The owner authorized push, an official PR, every review response and at most two external review rounds. Final merge, tagging and release publication remain owner-controlled. The proposed release is documented in [release-recommendation.md](release-recommendation.md).

## Spec Kit

Specify, clarify, checklist, plan/research, tasks, blocking analyze and implement completed. Convergence assessed 16 functional requirements, 7 success criteria, 13 acceptance scenarios, 8 design decisions and all 5 constitution principles. T029 and T030 closed the expanded-scope ambiguous-lane and bounded-component-scan gaps. Their regressions pass; no remaining buildable gap was found. Final verification and review gates remain tracked separately.

## Local verification

- Full `go test ./...` with pinned native Cueson passed (89.683 seconds). Native repair covers accepted/unchanged decisions, exact target-only calls, replay, finite attempts across retries, cancellation/restart, corrupt checkpoint refusal, source/document/mapping/owner fencing and temporary checkpoint retirement.
- Native-format merge fixtures pass for SubRip, WebVTT, ASS and SSA, preserving unaffected semantic cue fields and native source data. Structural cue ordinals remain aligned with accepted provenance.
- The native CLI journey passed admission, 14-window interrupted processing/resume, two-gap coverage, two-window repair, fixed-basis improvement, retrieval, request replay, explicit non-improving repair and actual publication retirement. Its retained receipt is `.local/s064/recording-journey-receipt.json`.
- Strict request/output schema and human outcome tests pass. Dense normalization exercises 60,000 independent cues; unresolved lanes reached by whole-cue expansion are rejected.
- Full foundation/Node/Python checks initially exposed the new timer inventory entries, strict schema object typing and a missing verification link. Those integration omissions are corrected. The setup source-binding fixture additionally requires selecting the pinned compiler directly instead of the older host launcher.

Foreground gate logs and per-phase UTC timestamps/durations are retained under `.local/s064/`. A final local gate summary and exact-head CI/review receipts will be appended before owner handoff. Full platform/backend acceptance is not claimed by these local results.

## Evidence limits

Deterministic fixture correctness does not establish acoustic recognition/speaker accuracy or real-model CPU/NVIDIA performance. Real-model CPU/NVIDIA qualification is not elected. Local execution remains headless; installed GUI/package acceptance belongs to isolated CI desktops. Catalog schema remains 11; software/docs/master schema remain aligned at 1.0.3. No release or merge has occurred in this slice.

## Local gate closeout (2026-10-31)

All local gates are green: `npm run check` (1,982 project text files; 29 schemas, 26 contracts, 34 examples), `npm test` (222/222), `go test ./...` with native Cueson, `go vet ./...`, Python discovery (735 tests, 5 platform/election skips), frontend typecheck/build/tests (109/109), site tests (18/18) and production site build (169 checked export pages). The final request/response schema tests also pass after the strict schema correction. The final native CLI journey took 11.172 seconds, or 16.675 seconds including its isolated native build/setup, and read back human accepted/unchanged summaries.

The verification campaign began after already-present prerequisite readiness at 22:35:41 UTC and its final affected checks completed at 22:40:22 UTC (280.8 seconds including the diagnosed failures, corrections and reruns). Prerequisite acquisition was not performed in this campaign, so it is not represented as a measured zero-second preparation. These local intervals do not substitute for the separately measured CI preparation/qualification/combined receipt.

The timing inventory now classifies the new successful waits under maintained expanded harness allowances; injected catalog leases retain their contract semantics. No production deadline or CI inference refusal was weakened. The schema registry expectation advances from 24 to 26 additive contracts and from 32 to 34 examples. UTF-8/LF, Markdown links, version alignment and upstream kit integrity passed the foundation check.

## External review round 1

The automatic Codex review at `1b9da18b` identified one P2 finding: two absent voice assignments compared equal, making corroborated overlapping repeated text incorrectly certain. The regression reproduced that defect. Coverage now requires nonempty matching voice assignments for certainty; both-unresolved, one-unresolved and different-voice repetitions remain visible but uncertain, while the same resolved local voice remains eligible. The complete processing package passes after the fix. No separate security review receipt has arrived.

## Initial package qualification correction

Initial CI package runs reached the installed recording journey, then rejected duplicate `--request-id` flags. The maintained invocation harness owns request correlation; the journey now supplies its replay UUID through the existing `request_id` keyword instead of duplicating an argv flag. The local native journey was rerun with the same automatic correlation convention and passed in 10.625 seconds (15.213 seconds including build/setup). Product duplicate-flag rejection remains intact. Failed package logs are retained under `.local/s064/`; terminal run audits will be retained before the next full CI cycle.

## Reviewed implementation acceptance

[PR CI 41250496793](https://github.com/shruggietech/insonic/actions/runs/41250496793) passed all elected checks for reviewed head `912d1e0acc57f0cfdf7c2f9971e697041dfd4636` (GitHub temporary merge revision `0820cf425043fb3e90eb8af358dafc72fad130c3`). This includes Windows/Linux core and native runtime, deterministic installed CLI/documentation journeys, Windows setup/interactive/archive/maintenance qualification and exact-byte join, Linux CLI/desktop packages, PostgreSQL/S3/ArcadeDB fixtures, documentation rendering and complete candidate collection.

The attempt-1 acceptance archive is retained outside Actions at `.local/s064/review-pr-ci/`, alongside run/jobs/artifact metadata and the terminal audit. Artifact 13061725687 has archive SHA256 `0c0ea69708dbc14882cb70336eff72b7c130f6d1dc1eabd1797176055679aa60`. Terminal acceptance is true: preparation 264 seconds, qualification 760 seconds, combined 1,024 seconds; no target or hard-limit overrun. Earlier failed PR/direct-head receipts remain in `initial-pr-ci/` and `initial-head-ci/` with their original failure conclusions, separate timing and verified archive digests.

[Codex round 2](https://github.com/shruggietech/insonic/pull/236#issuecomment-6550298389) completed without findings at the reviewed head. The initial P2 finding was fixed, answered and resolved. Exactly two review rounds were used; no third round is authorized or requested. No separate security review receipt was returned. The retained review ledger and thread readback show zero unresolved findings.

## Final owner handoff gate

The closeout commit changes only internal Spec Kit status, tasks and this evidence record. Runtime, public contracts, documentation, qualification helpers and tests remain the bytes reviewed at `912d1e0acc57f0cfdf7c2f9971e697041dfd4636`. A fresh PR run and a direct branch-head run validate the final closeout commit; their exact identities, terminal results, digest-verified retained receipts and timings are recorded in the [PR conversation](https://github.com/shruggietech/insonic/pull/236) before owner handoff. That final external receipt avoids embedding a commit's own not-yet-known hash in its contents.

Issues #217-#219 are In review/S064 under milestone 47 and close through PR #236 on owner merge. The candidate is ready for the owner's final review and squash-merge ritual only after those final runs are green. No merge, tag or release publication is performed. The v1.1.0 recommendation and separately elected acoustic/CPU/NVIDIA evidence limits remain as documented.
