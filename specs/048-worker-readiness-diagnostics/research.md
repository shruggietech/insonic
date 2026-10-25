# Research: S048

Research completed 2026-10-25 through three read-only Spec Kit plan agents.

| Decision | Evidence and rationale | Alternative |
| --- | --- | --- |
| Fixed diagnostic/checkpoint | Python catches version/import/inventory failures together; Go maps run/JSON failure to engine_unavailable. A terminal-only response cannot identify a killed import/hash. | Raw stderr parsing risks noisy libraries and secret text; a separate logging service is unnecessary. |
| Preserve enclosing budget | Inspection starts 60 seconds before interpreter hashing. Readiness shares another 60 seconds across both roles and receipt/model hashes. Interrupted Hash becomes checksum_mismatch; reuse suppresses the error. | Increasing all product clocks or concurrent heavy imports would obscure the demonstrated attribution defect. |
| Preserve failed setup facts | Prepare discards failed Inspection; app work checkpoints already support bounded safe result facts. | Returning a partial Prepared value alone loses facts when callers discard error results. |
| Optional observation schema | Inspection schema disallows unknown fields and receipt verification decodes strictly. | Rewriting accepted receipts would break their frozen identity. |
| Ephemeral cancellation | IPC already clips caller deadlines, but disconnect cancellation covers only assistance calls. | Canceling enqueued setup on disconnect would violate durable ownership. |
| Actual CPU acceptance | S046 records ambiguous final probe failure and a separate 51.202-second successful snapshot of 27,571 files. Existing real journey isolates and removes profiles. | Passing repetitions cannot identify the historical failure cause; hosted CI forbids acoustic inference. |

No unresolved technology choice remains. Existing local model inputs are read-only, transient campaign arguments; private host paths are not product or public documentation.
