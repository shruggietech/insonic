# S020 data model

SavedPreset: kind=import-preset, schema_version=current supported version, UUID id, revision>=1 after persistence, UTF-8 name (1..128 characters), state active/deleted, nonsecret options. Names match exactly after trimming; IDs take no precedence over a conflicting name. At most 1000 retained identities, registry <=1 MiB. Create expected_revision=0; update/delete require the observed positive revision. Tombstoned IDs cannot be recreated.

Registry: kind=import-preset-registry, schema_version=1.0.1, items. The journal setting key is reserved import-presets. Receipt stores the exact accepted preset, including original revision/configuration.

Import options: existing fields plus optional destination storage profile and optional pipeline_id (empty string explicitly opts out). Request overrides are separate from defaults/items. Per-ordinal frozen selections retain preset identity/revision/effective options and frozen recording payload template; accepted-item scrub preserves this envelope.

Destination: immutable profile_id/profile_revision, filesystem or S3 adapter, supported contract_version and nonsecret configuration, credentials by ID only. Publication profile authority drives subsequent reads even after preset edits/deletion.
