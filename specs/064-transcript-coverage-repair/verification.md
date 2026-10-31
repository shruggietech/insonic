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
