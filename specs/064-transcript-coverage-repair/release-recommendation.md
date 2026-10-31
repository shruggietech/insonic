# S064 milestone release recommendation

## Recommendation

Propose **v1.1.0** for the next coordinated release after owner merge and separate release authorization. S063 and S064 add public long-recording window, coverage and targeted-repair operations; these are backward-compatible capabilities beyond a patch-only change. S061/S062 maintenance fixes can ship in the same release. This document is a recommendation, not a tag or publication receipt.

## Compatibility

- Existing processing requests and saved pipelines remain valid. Coverage and repair policies are optional; ordinary processing does not implicitly repair.
- Repair is an explicit durable operation with exact source/document/mapping fences, finite per-window attempt budgets and one current accepted document. Existing catalog schema 11 and publication/checkpoint mechanisms are reused without a catalog migration.
- Current software, public documentation and master schema remain aligned at 1.0.3 during this work slice. The separately authorized release must advance all coordinated version surfaces and preserve supported historical contracts together.
- New reports and repair receipts are versioned additive contracts. Coverage is derived from persisted observations; older recordings without compatible evidence explicitly report unknown activity. Inspection does not run inference.
- CLI operations use the shared runtime. Dedicated GUI controls are not added here; public documentation identifies the advanced CLI workflow.
- Structural cue ordinals and document summaries may change during a native-valid merge. Unaffected cue identities, timing, text, native fields and source/voice associations remain invariant.

## Proposed concise highlights

- Process long recordings in bounded windows and resume verified completed work after interruption.
- Inspect transcript coverage by source track and repair selected gaps without retranscribing unrelated windows.
- Retain native subtitle semantics, canonical media and one accepted current document, with bounded repair decisions and temporary-checkpoint cleanup.
- Include capacity recovery and local device/runtime corrections from the preceding maintenance work.

[Full changelog](../../CHANGELOG.md).

## Qualification and evidence limits

The maintained deterministic journey admits a 34-second stereo source, interrupts and resumes 14 processing windows, diagnoses two known gaps, repairs only those windows, verifies unaffected native cues and source authority, extracts/retrieves evidence, exercises UUID replay and unchanged explicit repair, and checks temporary publication retirement. Native merge tests cover SubRip, WebVTT, ASS and SSA. Catalog fault tests run against SQLite and the maintained PostgreSQL integration fixture; the existing CI backend suite retains filesystem/S3 and LadybugDB/ArcadeDB coverage.

Final Windows/Linux installed-package and alternative-backend results must be read from the exact candidate CI receipt described in [verification.md](verification.md). Deterministic fixtures establish workflow/contract correctness, not recognition accuracy, speaker accuracy, real-model CPU performance or NVIDIA execution. Real-model CPU/NVIDIA qualification was not elected. No local attended GUI acceptance or physical hardware claim is made.

The release decision requires the final green mandatory checks and reviewed PR. Preparation, qualification and combined durations remain separate; 1,800 seconds is advisory and 3,600 seconds is the qualification hard limit. Audited acceptance evidence is retained outside the workflow before a full rerun. Owner final review and merge, version preparation, release qualification and publication remain subsequent actions.
