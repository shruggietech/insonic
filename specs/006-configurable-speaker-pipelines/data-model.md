# S006 data model

- Pipeline: ID, name, preset, current revision and typed nonsecret configuration. Create at expected revision0; updates compare current revision; operation receipts bind intent and latest digest.
- Speaker: stable ID, canonical name, active state/revision. Alias: ID, speaker ID, name, language/context, active state/revision and provenance. Text equality never merges IDs.
- Term: ID, canonical spelling, variants, language/context, active/revision and explicit optional speaker/alias links. Validate referenced identities.
- Election: pipeline ID/revision/digest, effective stage settings/quality, compiled bounded hints/digest/omission diagnostics. Saved with work before claim, never stores resolved credential values.
- Selection: paged current source/document/cue/local-speaker references plus computed resolved source timing and quality diagnostics. Computed view is not a second stored assignment array.

Pipeline/speaker/term update is CAS and replayable. Current document replacement and mapping correction continue existing preparation/evidence invalidation and retirement. Term/name edits do not rewrite embedded documents. Snapshot validation covers current entities and latest update proofs; historical migration preserves originals/current result identity and does not revive old assignment data.
