# S007 data model

Existing catalog/media/current recording/pipeline/speaker/term/work entities remain authoritative; no catalog migration is planned.

- Workspace selection is mutex-protected native state. An operation captures a workspace pointer/identity; stale frontend generations cannot commit a response to a different workspace.
- Settings section contains validated media_tools or processing_tools and a SHA-256 revision. CAS applies before atomic publication. Persisted input contains credential references only, never values. Selected backend profiles are shown without silently migrating stores.
- Current cue page binds media/recording revision and document digest. Bounded cue text, original-time string milliseconds and validated numeric seek seconds are presentation projections. Voice references use current recording-local UUIDs/mappings.
- Playback handle binds workspace, media/source digest/revision, optional current recording revision/document digest, materialized path and lease, bounded idle expiry and overall lifetime. Path remains native-only. Handle transitions: issued, touched/renewed, expired/closed and cleaned. Desktop tickets never authorize arbitrary supplied paths.
- Package inventory records version/revision/platform and sorted included file hashes/notices. Installed companion manifest uses package-relative paths verified under the installation root. Explicit workspace tool configuration overrides package defaults.
