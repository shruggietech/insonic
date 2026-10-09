# S011 research

## Elected replacement

Decision: extend current admission with a narrow typed replacement election. Rationale: saveLibrary rejects changed source fields and validates the previous Recording against incoming source; globally relaxing it would permit unelected overwrite. Always write new Recording authority with `untranscribed` for cleared audio. Alternatives: generic source mutation weakens contracts; deleting current Recording complicates receipt/current proof and consumer fences.

## Current evidence and retirement

Decision: compare source digest/map as well as document/state during evidence reconciliation; remove prior source-bound mappings on replacement. Rationale: identical retained documents do not validate old audio clips or identity evidence. Update source attachment and conditionally release old media_asset/metadata references before retirement because structural artifact FK references otherwise prevent cleanup forever. Alternatives: retaining alternate current asset rows violates canonical-only authority; blindly deleting shared rows violates current references.

## Roster anchor and identity

Decision: anchor to library_entry and persist empty revisioned header. Rationale: current_recording may be absent before processing; membership must precede transcript/model availability. Exact names/active aliases require a new bounded shared resolver because existing identity operations use IDs only. Bound to existing 1,000-speaker context, preserve Unicode/case, reject ambiguous identity results, and recheck elected active IDs in transaction. Alternatives: derive roster from mappings or Cue JSON confuses intent with evidence; copy names/profile snapshots duplicates mutable identity.

## Queue and replay

Decision: extend FreezeTargets and admission intent/receipt current proofs. Rationale: durable request digest already permits replay before lookup and accepted-item markers scrub input locators. Preserve this path; alias retarget and source/document/roster edits cannot alter queued elections. Alternative: resolve at worker execution changes declared intent.

## Interfaces

Decision: targeted media is distinct from standalone transcript; prepare/dispatch must use request kind rather than repurpose every record+source. Preserve transcript source alias compatibility in transcript requests. Existing parseFlags prohibits repeats, so exempt explicitly repeatable speaker selectors. JSON/CSV/schema fields require coordinated additions to strict allowlists/allOf shapes. `recordings roster` follows existing plural CLI family. Alternatives: new unrelated handlers drift precedence and atomically accepted results.

## Scope and verification

Decision: no bulk pre-v1 audio migration, no acoustic matching/training/model download. Rationale: owner confirms no deployed audio and elects this recording lifecycle slice. Existing fixtures/old formats remain technical interoperability examples, not deployed user data. Additive catalog evolution, current proofs and deterministic native/backend tests remain required.
