# Research decisions

2026-10-09, read-only catalog, adapter and surface research followed by design consolidation.

- Decision: Reuse renewable `work_operation` claims and catalog work-authority acceptance. Rationale: cancellation, concurrency and stale-attempt fencing already exist. Alternative rejected: demo `job` rows or an independent training scheduler.
- Decision: Drain stable-epoch corpus pages and revalidate reference-only membership transactionally; default to independent manual attribution for enrollment. Rationale: current selection caps each page and mappings previously lacked origin. Alternative rejected: treating displayed page or automatic guesses as independently confirmed corpus.
- Decision: Separate immutable output metadata from invalidatable dataset/preparation manifests. Rationale: source correction must remove obsolete evidence without deleting completed weights or rewriting origin. Alternative rejected: using invalidated lineage as proof that exact weights are unavailable.
- Decision: Declared pinned local executable and explicitly selected HTTP training adapters execute actual training, with supported resume/checkpoint capabilities and bounded artifact validation. Rationale: no single model architecture covers the public contract. Alternative rejected: replacing training with only a fake fixture or centroid enrollment.
- Decision: A real pinned offline embedding consumer supports versioned enrollment profiles and cosine-based roster matching, with explicit threshold/margin/minimum evidence. Rationale: matching requires compatible acoustic references; names are insufficient and thresholds are not universally calibrated. Alternative rejected: forced top candidate or roster-count diarization.
- Decision: Existing mappings migrate as manual; automatic mapping acceptance is atomic, bounded and current-reference based. Rationale: preserve corrections and prevent feedback. Alternative rejected: duplicated assignment arrays/history in work results or CueJSON.
- Decision: Separate speaker-model operations from base-install inventory while offering documented speaker-filtered convenience forms. Rationale: model/family/version IDs have distinct semantics. Alternative rejected: silently changing existing base model commands.

Primary implementation reference: [pinned pyannote speaker verification component](https://raw.githubusercontent.com/pyannote/pyannote-audio/4.0.7/src/pyannote/audio/pipelines/speaker_verification.py). Deterministic adapter tests establish runtime integrity, not acoustic accuracy.
