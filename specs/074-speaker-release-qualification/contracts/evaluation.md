# S074 evaluation contracts

The maintainer entrypoint is scripts/qualify-speakers.py. Deterministic mode must not initialize or invoke real models. Real mode requires explicit election and local configuration; it remains separate from required deterministic CI.

Contextual evaluation invokes installed speakers run/discover/resolve with supplied synthetic cue/voice context and no expected-answer payload. It reads current hypotheses/decisions and reports all cases. An owned loopback provider may serve deterministic expected protocol replies solely in deterministic mode; those outputs never claim model quality.

Acoustic evaluation invokes installed dataset creation, real pyannote-profile enrollment, profile selection and recordings match. cpu and cuda are separate explicit rows and require matching actual device provenance. Source timing is supplied reference truth, not inferred acoustic alignment.

Receipts include mode, source/corpus identity, per-case terminal states, separate contextual/acoustic metrics, model/configuration/device provenance and timing. Undefined ratios are null. Model drift, incomplete cases, bad source identity, wrong-device execution and protocol failures are explicit errors, not omitted rows or guessed success. Synthetic and licensed measurements retain their distinct limitations.

Existing installed lifecycle/recovery/backend receipts remain mandatory. No new public runtime endpoint is required by this slice. Existing discovery protocol version remains integer 1 after release preparation.
