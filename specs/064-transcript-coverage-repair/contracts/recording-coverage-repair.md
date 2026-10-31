# Recording coverage and repair contract

## Coverage

The CLI exposes recordings coverage MEDIA_ID with optional --input FILE and ordinary human/JSON envelope modes. Shared operation is recordings.coverage. Input controls gap tolerance, minimum reported gap, boundary tolerance, duplicate overlap fraction and detail after/limit. No input means documented defaults. Unknown properties and invalid/overflowing bounds are rejected.

Output is a versioned recording-coverage report with exact snapshot identity, independent lane metrics, named fractions, complete diagnostic totals, bounded region page and evidence limitations. No inference executor, document mutation, timestamp repair or identity assignment runs during inspection. Missing source authority/audio, explicit no-speech observations and absent acoustic observations are separate states. Byte availability is only claimed if actually checked.

## Repair

The CLI exposes recordings repair MEDIA_ID --input FILE. Shared operation is recordings.repair and enqueues durable work. Input includes exact expected document/source revisions and document digest, optional saved pipeline election/overrides, recognition/diarization routing, explicit ranges or automatic diagnosed-region selection, and bounded repair policy. Explicit ranges use existing native stream/channel/start_sample/end_sample selection semantics; disjoint ranges are supported without duplicating plan track lanes.

Effective policy uses default one and maximum three execution attempts per selected window across restarts, at most 32 normalized regions, a bounded duration/window selection and permitted context. A non-improving valid candidate terminates unchanged; extra attempts are for failed selected-window execution. The work payload freezes every semantic election and source/document/mapping/evidence digest. Reusing a request UUID with different intent conflicts.

Before inference, expose normalized reconciliation scope and actual core/context window ranges. Extend to whole affected cues within the finite budget; reject ambiguous lane ownership instead of guessing. No unrelated windows execute. Candidate merge keeps one accepted Cueson document, original native clock/source envelope and all out-of-scope semantic cue fields. Structural ordinal and root summary changes are required by Cueson.

Acceptance requires final native validation, exact unaffected-field preservation, stable evidence/source/options, strict reduction of uncovered observed speech, duplicate overlap or boundary inconsistency, and no worsening of other defect metrics or protected cue coverage. Source/document/owner/mapping conflicts cannot commit. Normal mapping/evidence/dataset invalidation and recoverable cleanup follow an accepted replacement.

Work results expose a compact recording-repair decision with accepted/unchanged status, reasons, before/after metrics, current/candidate identities, scopes and actual attempt counts. Failure responses retain actionable codes and bounded progress. Accepted/unchanged decisions reconcile idempotently after unknown commit outcomes. Candidate transcript/checkpoint bytes are retired after a terminal decision; original sources remain immutable.

## Compatibility and GUI

The operations and policy fields are additive. Older recordings can return unknown acoustic evidence; they are not silently reprocessed. The CLI/shared runtime ships first and the documentation identifies the absence of dedicated GUI controls. Existing normal processing remains unchanged. Release version remains aligned across software/docs/master schema until a separately authorized release workflow applies the proposed version.

## Concrete defaults and limits

Coverage gap_tolerance_us defaults to 0, minimum_gap_us to 250000, boundary_tolerance_us to 1000 and duplicate_overlap_fraction to 0.5. Tolerances are bounded to 60000000 microseconds, overlap fraction to (0,1], and detail after/limit to nonnegative/1..1000. Repair max_windows defaults to 64 with ceiling 16384; max_duration_us defaults to 300000000 with ceiling 604800000000 (the existing seven-day source bound). max_attempts defaults to 1 with ceiling 3, and normalized target regions cap at 32. Budget expansion never silently drops a region: exceeding the elected budget returns an actionable limit error.
