# Internal recovery qualification receipt

Maintainer evidence, not a new public runtime schema.

`insonic-import-recovery-qualification-v1` binds executable SHA-256 and Windows/Linux platform to exactly three cases: same-source, changed-source and corrupt-report. Each case records source digest/size, work identity, old/new runtime sessions and process identities, interrupted/replacement attempts, available canonical/report publication identities and digests, absent pre-kill admission, actual source-open break, abrupt old-runtime exit, post-restart admission/refusal and independently read authority. The same-source case preserves raw metadata and date provenance and proves idempotent reconciliation. Negative cases preserve unrelated accepted state.

Success requires the complete case inventory, correct image/platform correlation, distinct sessions/attempts, verified publication/byte bindings and expected outcomes. Final runtime natural exit, barrier/helper release and owned fixture removal are mandatory. Missing/mismatched/stale data, a graceful old-runtime shutdown, a client-only interrupt, wrong process, premature/automatic barrier release or forced final success fails validation.

Package receipts nest this under `cli_contract.recovery_contracts`, alongside existing coverage identity and terminal receipts. Release validation requires it except for exact immutable original v1.0.2 receipt hashes and already authorized historical profiles. All other existing checks remain required. Source/package archive and inventory binding is inherited from the existing coverage identity and rechecked during independent CI audit.

Nested operations use the original enclosing qualification deadline. Kernel lease auto-break is an additional fixture bound, never a changed product lease/deadline.
