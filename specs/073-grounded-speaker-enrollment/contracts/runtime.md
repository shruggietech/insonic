# Runtime contracts

- speakers.apply: recording item ID, expected discovery revision. Applies only accepted independently supported registered identities; preserves owner mappings and returns applied/preserved/unknown/ambiguous/conflicting counts and operation evidence. Repeated request ID reconciles its accepted result.
- speakers.retract: recording item ID, observed recording revision and targeted contextual mapping revisions/application identity. Removes only matching contextual authority; concurrent corrections cause conflict or remain preserved, never overwritten.
- speakers.run: recording item ID, explicit inline speaker workflow configuration or exact pipeline ID/revision. Configuration selects provider/limits/resolution, application and optional dataset/enrollment/profile selection. Returns durable work ID; work show/wait/cancel/retry reuse standard contracts.
- speakers.discover/hypotheses/resolve and manual recordings.map remain supported. Outputs distinguish hypotheses, supported-contextual, acoustic matches, manual-preserved and unknown/ambiguous/conflicting outcomes.
- Saved pipeline speaker_workflow contains explicit normalized discovery and application elections, optional enrollment configuration, and no credentials except encrypted references. Published schemas and strict Go inputs must agree.

All operations use shared runtime/desktop dispatch. No per-item approval, implicit provider route or mutation of source transcript. Limit failures and current-authority conflicts return actionable structured errors.
