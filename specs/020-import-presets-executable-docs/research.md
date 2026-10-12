# S020 research

## Registry and destinations

Existing setting_revision is workspace-scoped, revisioned, nonsecret and already portable on SQLite/PostgreSQL. PutNamedSetting alone cannot safely replay an app-side reconstructed registry after another mutation; use a typed mutation with digest independent of current state. Validate reserved registry values during generic writes and restore. Keep deleted identities as tombstones; reject duplicate active names.

Artifact service currently binds all publications to the single default storage profile. Genuine preset destinations require immutable profile registration and routing by recorded profile identity/revision for materialization, playback, processing, cleanup and backup. Merely storing destination configuration does not satisfy #70.

## Admission and processing

ReadManifest/CLI currently call PrepareImport before preset lookup, capturing an implicit timezone too early. CLI flags currently merge into defaults and lose to per-item options. Keep explicit overrides in a separate layer. Catalog accepted-item scrub removes all item configuration; freeze elections in ordinal envelope metadata. Reuse electRecordingOptions, frozen model election and recordingPayload execution, with deterministic child IDs after admission.

## Documentation

Current desktop examples contain unsupported workspace create, media attach-subtitle, jobs list and model-fetch syntax. Current ingestion prose claims pipeline work remains unimplemented and uses local as a nonexistent built-in saved preset. Fix current candidate only; immutable archives remain unchanged. Mapping revision is the current recording revision; mapping_revision independently selects creation/correction.

## Alternatives rejected

Inert labels, default fallback, alternate processor, generic setting reconstruction and per-item mutable lookup all violate the requested observable behavior. A separate preset DDL migration duplicates existing versioned settings. Documentation-only promises cannot substitute for package execution.
