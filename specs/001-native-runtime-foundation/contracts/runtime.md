# Runtime contract

## Wire protocol

One newline-delimited request/response per connection; maximum one MiB. Version matches VERSION. Envelopes carry kind runtime-request or runtime-response and are registered in the master JSON Schema. Current-user Windows pipe or private Unix socket; no application network listener. Validate workspace, UUID request identity and operation arguments before dispatch.

| Operation | Arguments | Result |
| --- | --- | --- |
| workspace.show | None | Workspace ID, selected profiles, session |
| doctor | None | Platform paths and capabilities |
| jobs.start | duration_ms, bounded | Session job/attempt identity |
| jobs.show | job_id | Current attempt |
| jobs.cancel | job_id | Idempotent cancellation |
| jobs.retry | job_id | New generation after failure/cancellation |

The bounded attempt is a foundation qualification operation. runtime serve is internal. workspace init creates defaults without replacement. --workspace selects a root; otherwise discover ancestors. --json emits versioned machine records, diagnostics use stderr, failures exit nonzero and no invocation prompts.

## Errors

Fixed messages for invalid_request, incompatible_version, workspace_mismatch, not_found, conflict, unavailable, cancelled, output_limit and operation_failed. No child output, backend exception text or secrets.

## Child boundary

Absolute executable, literal arguments, explicit working directory, context cancellation, finite output and closed input unless protected stdin is explicitly supplied. Windows hidden creation flags. Qualification secrets never enter arguments or receipts.

Supervised children own an isolated Unix process group or a Windows kill-on-close job. Windows assignment occurs before a suspended child resumes. Cancellation, output exhaustion and qualification timeouts terminate descendants as well as the immediate child. A detached runtime deliberately outlives its launching client.
