# S020 shared interface

CLI import-presets create --input FILE; list; show UUID_OR_EXACT_NAME; update UUID --input FILE; delete UUID --input FILE. Runtime operations import-presets.create/list/show/update/delete. Create/update input carries preset; update/delete also carry expected_revision. Shared schemas reject unknown fields, credential values and incompatible versions. List is paged through existing page input.

media.import accepts defaults/items/overrides. Runtime resolves effective preset first, merges preset < defaults < item < overrides, captures timezone, freezes destination and optional processing election, then queues durable work. media.refresh resolves the same options and frozen selection. Public callers cannot supply frozen elections/model digests/accepted receipts. Invalid selections and dependencies produce actionable structured errors before queueing.

Work results expose preset identity/revision, effective choices and deterministic queued_job_ids. Retries retain the envelope election and reconcile accepted receipts. Desktop lifecycle and selection use these exact operations.
