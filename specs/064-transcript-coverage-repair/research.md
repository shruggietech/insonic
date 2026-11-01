# Research: S064

## Persisted acoustic evidence

Decision: retain compact text-free, speaker-free source-clock observation intervals from verified window evidence in recording provenance. Each stage has executed/reused/unavailable status, explicit no-speech observations and positive intervals. Bind evidence to source revision/digest and source-map digest. Reports separately bind document identity.

Rationale: current cue assembly clips speaker assignments to existing cues, losing evidence inside missing-text gaps. Skipped stages initialize no-speech values and must not become false silence observations. Clip each window to its core, keep lanes independent and use interval unions.

Alternatives considered: reconstructing speech from final assignments misses gaps; running fresh inference on inspection violates the read-only contract; retaining full checkpoint transcripts would duplicate accepted authority.

## Coverage and acceptance

Decision: expose selected-audio, accepted-cue, observed-speech, covered/uncovered-speech, explicit no-speech and unknown durations per lane. Diagnose uncovered observed speech, temporally overlapping lexical duplicates and cue/source boundary inconsistency. Keep model disagreement and ambiguous multitrack cues explicit. Pagination affects detail only. Compare repairs against the same evidence/options/source basis, requiring a strict defect reduction with no worsened other defect or lost protected cue coverage.

Rationale: silence/confidence and missing evidence cannot prove transcript failure. Genuine nonoverlapping repetitions remain valid. Same-basis comparison prevents a candidate from declaring its own success by changing the evidence.

Alternatives considered: a single overall coverage score hides denominators; arbitrary transcript density or confidence thresholds reject valid silence/music/quiet speech.

## Candidate construction and transactional acceptance

Decision: extract the existing window builder from its commit wrapper. Keep a complete valid processing plan and filter inference to frozen selected windows. Expand requested ranges to their window cores and complete intersecting cues; expose the bounded scope before execution. Reuse verified checkpoints. Merge raw accepted cues outside scope, permitting only structural ordinal/root-summary updates, and project new cues into the original native format through the existing Cueson driver. Preserve native dependency references and validate the final document.

Rationale: a second processing engine would duplicate source verification, model election and checkpoint logic. Cueson requires ordinal=array position. Supplied ASS/SSA native event/style references require bounded dependency merging, not full-document conversion.

Alternatives considered: committing a temporary candidate first violates current authority; whole-document reassembly can rewrite unaffected attribution; skipping native validation misses scripted-format cross references.

Decision: freeze speaker-mapping digest with source/document/evidence identity and check it transactionally in repair acceptance. Reuse ordinary dependent evidence/dataset invalidation and manual mapping preservation. Repair work uses the existing recording payload with optional repair election, and all portable/backup/model/cleanup kind fences include the new kind.

Rationale: mapping corrections do not increment the recording revision. App-only checks leave a race. Existing acceptance receipts are required for unknown-commit recovery.

## Saved policy and bounded execution

Decision: optional pipeline coverage/repair policies use existing full-policy override/election rules. Ordinary process behavior remains unchanged. Explicit recordings.repair elects ranges or diagnosed targets with default one execution attempt per selected window, maximum three, at most 32 normalized regions, bounded context and output. Attempt counts are durably consumed before failed-window retries and never reset on resume. Stop on the first valid candidate decision, no targets or exhausted budget; do not repeat a valid non-improving candidate.

Rationale: finite unattended repair is owner-approved. A saved policy alone must not silently trigger new operations. Request IDs bind original intent and retries.

## Qualification and release recommendation

Decision: extend the maintained installed-package documented workflow with deterministic loopback fixtures and exact public JSON examples. Cover Windows/Linux, source storage and catalog alternatives through existing CI suites; keep production CI model-inference refusal unchanged. Native Cueson/FFmpeg verification is distinct from acoustic accuracy. Real CPU/NVIDIA inference remains separately elected and unverified unless run.

Rationale: existing package qualification already exercises the actual CLI and native companions under the retained timing contract. Maintainer acoustic campaigns must not become required CI.

Decision: propose v1.1.0 after milestone acceptance, without changing current aligned development release versions or publishing. Additive window processing/coverage/repair exceed patch scope; schema11 and older-client limitations must be described.

## Research sources

Repository inspection by three Spec Kit research agents covered internal/processing, internal/subtitles, internal/app, internal/catalog, internal/pipeline, CLI schemas and scripts/documented_workflows.py. No external API/library choice requires new dependency research. All research questions are resolved.
