# Data model: S064

## Coverage evidence

A versioned text-free observation set binds source digest/revision and source-map digest. Each observation carries window/output identity, lane ordinal, bounded source-clock core, explicit recognition/diarization status (executed, reused or unavailable), positive speech intervals and explicit no-speech flags. Observation count is at most 32,768 (two stage histories across 16,384 windows); combined interval count is at most 1,000,000. No transcript text or speaker identity is retained here. Missing or mismatched evidence yields explicit unavailable status rather than silence.

## Coverage report

Report identity binds recording/source/document revisions and digests plus evidence and effective-option digests. Each lane retains stream/channel, native sample selection and source-clock interval. Named metrics are interval-union durations: selected audio, cue coverage, observed speech, covered/uncovered speech, observed no-speech and unknown activity. Defect metrics include overlapping duplicate duration and boundary inconsistency count. Fractions name numerator/denominator and are null when unavailable. Regions carry bounded lane/range, reason, supporting cue/window IDs and uncertainty. Details default to 100 and cap at 1,000 per page; totals cover the entire snapshot. At most 65,536 cues and 16,384 windows are accepted; dense duplicate comparison has an explicit work/output cap.

## Repair policy and election

Optional saved coverage/repair policies are full-policy overrides. Repair request selects explicit native sample ranges or automatic diagnosed targets. Frozen election binds exact source/document/mapping/evidence identity, normalized source-clock reconciliation scopes, full window options, model/tool configuration and finite budgets. At most 32 normalized regions, a bounded selected-window count/duration, and max_attempts in 1..3 (default 1) apply. max_attempts is the maximum execution attempts per selected window across restarts, consumed durably before execution. Successful verified checkpoints do not consume another attempt. A valid non-improving candidate stops immediately rather than repeating identical work.

## Candidate and decision

The shared builder produces a temporary candidate without committing. Only selected windows execute. A raw-cue merge preserves unaffected semantic fields and native dependencies, with structural ordinal/root-summary changes allowed. Native Cueson validates the result. Comparison uses the original evidence and tolerances, requires strict target defect improvement and forbids worsened defects or lost protected cue coverage.

The decision records accepted/unchanged status, reasons, actual attempts, source/document identity, declared reconciliation/inference scopes and before/after metrics. Accepted and unchanged decisions use durable idempotent receipts. Accepted replacement and mapping fence share one catalog transaction. Unchanged acceptance retires only this work's checkpoints, without manufacturing a document replacement. Failed/cancelled work retains compatible ready checkpoints for bounded resume; stale source/document invalidation retires them through existing authority reconciliation.

## State transitions

Queued -> claimed -> selected-window attempts/checkpoints -> candidate -> accepted or unchanged. Transient failure may retry within remaining per-window budget; cancellation/owner loss stops execution. A resumed work first reconciles an existing decision, then verified checkpoints and attempt counts. Any source/document/mapping mismatch prevents acceptance. Existing work ownership, source revision, current receipt and portable backup proofs remain authoritative.

## Concrete defaults and limits

Coverage gap_tolerance_us defaults to 0, minimum_gap_us to 250000, boundary_tolerance_us to 1000 and duplicate_overlap_fraction to 0.5. Tolerances are bounded to 60000000 microseconds, overlap fraction to (0,1], and detail after/limit to nonnegative/1..1000. Repair max_windows defaults to 64 with ceiling 16384; max_duration_us defaults to 300000000 with ceiling 604800000000 (the existing seven-day source bound). max_attempts defaults to 1 with ceiling 3, and normalized target regions cap at 32. Budget expansion never silently drops a region: exceeding the elected budget returns an actionable limit error.
