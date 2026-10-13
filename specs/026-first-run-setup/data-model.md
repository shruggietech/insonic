# Setup data model

BootstrapDraft: kind/version, revision, path, normalized workspace configuration, stage workspace/backends, active/deferred status, bounded replay receipts (request UUID, SHA-256 intent, accepted revision). Protected values are rejected by strict backend configuration validation and never stored. Atomic saves use an interprocess lock; stable request identity reconciles unknown results.

Journey: kind/version/workspace ID; active/deferred/completed status; current stage; five typed stage records (pending/completed/skipped plus observation digest); choices limited to processing mode and selected pipeline ID/revision. Effect authority remains normal catalog/work/settings objects. Input expected revision is nonnegative. Unknown fields and invalid IDs/enums are rejected.

Show returns persisted setting revision, current configuration proofs and needs-attention overlays. Save returns accepted receipt plus current reconciled state. A completed stage whose observation changed becomes needs attention. Reopening uses the same library identity. Deferring changes metadata only. Completion requires no pending optional stage; ready import and unresolved optional dependencies remain independently visible.
