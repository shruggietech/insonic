# S004 shared runtime contract

Credential setup/status/add/replace/delete/unlock operate through a catalog-independent shared manager. Entered secrets use bounded stdin/current-user IPC; responses never contain values. Runtime resolves selected credentials for authenticated catalog/storage/acquisition.

Media operations: import, manifest batch, list, show, metadata/raw, set-origin, refresh and relocate. Model operations: register, acquire, list, show, verify/materialize. Work operations: show, cancel and retry. All mutating requests carry workspace-scoped idempotent operation IDs; cross-workspace calls fail.

Import/model work publishes verified artifact bytes before catalog references. Current library metadata changes commit atomically; cleanup retires old reports. Status distinguishes pending work, extraction/date diagnostics and complete admission. No timer-only success, fallback provider, hidden alternate output or compulsory review.

CLI and desktop bridge invoke the same runtime/services. JSON Schema documents exact payload shapes, required fields, discriminator layouts, limits and response categories.
