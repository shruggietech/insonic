# Shared catalog contract

A Store binds explicit workspace scope. SQL/connection details stay inside the adapter. Typed mutations use expected workspace revisions and operation UUIDs; identical replay returns the same receipt and changed intent conflicts. Admission, receipt and consecutive graph events commit together.

Durable jobs expose start/show/history/cancel/retry; accepting mutation request_id provides idempotency. Renew/complete check exact owner/attempt/generation, running state and unexpired lease transactionally. Recovery retains abandoned history. Outbox claims expose only checkpoint+1 and acknowledgements validate exact event/current generation.

CLI exposes jobs history and catalog show/export/restore/migrate. File transfers avoid the 1 MiB IPC frame; writes retain owner/catalog boundaries and restore refuses occupied authority. Snapshots preserve exact integers and expire imported authority. The desktop retains the shared IPC bridge.

Opaque SecretProvider resolves PostgreSQL username/password JSON in memory. Errors expose categories only, never driver errors or credentials. Doctor distinguishes configured versus available operations.