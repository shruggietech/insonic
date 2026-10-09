# Speakers and terminology

## Local voices and known speakers

A diarization voice belongs to one recording. Its local token occurs in the current embedded Cue JSON document; it is not a global person ID. The same person in another recording receives another local speaker token. Imported tokens retain their exact upstream-valid values. Native subtitle labels and mentioned names remain source observations, independent of that identity. A catalog speaker is a stable user-facing identity with a canonical name, active state and revisioned aliases.

```mermaid
flowchart TB
  Audio[Mapped recording audio] --> Turns[Temporary acoustic voice results]
  Turns --> Document[Current Cue JSON local speaker token assignments]
  Document --> Mapping[External recording and UUID mapping]
  Mapping --> Known[Known speaker or unresolved identity]
  Document --> References[Current cue and local speaker token references]
  References --> Playback[Playback and evidence]
  References --> Corpus[Library-wide current speaker corpus]
  Corpus --> Models[Optional trained model lineage]
```

Acoustic diarization describes which voice spoke when. Known-speaker mapping describes who that voice may represent. A mentioned person, quoted statement, scene participant and speaking voice are different observations. Unsupported identity remains unresolved and does not block valid source-level operations. Model confidence is a diagnostic, not proof of identity.

Known-person corrections update the external recording/token mapping without rewriting subtitle text or document bytes. Mapping rows bind current document identity and expected revision. A removed local speaker token cannot retain an active mapping. Other records cannot establish another list of assignments under a voice, segment or attribution-revision name.

## Reference-only speaker audio

A current segment record refers to recording ID, document digest, cue ID and local speaker token, with an optional managed clip reference. It stores no copied interval, channel, known-person assignment or attribution array. Query and playback resolve the interval from current embedded assignments and the recording's source map. Untimed participation does not provide an exact clip interval, and uncovered acoustic speech does not gain invented subtitle cues.

The speaker corpus spans all current library recordings. A source can contribute overlapping voices, and deduplication uses original source/cue evidence rather than counting repeated processing as independent speech. Lazy extraction uses verified source bytes and explicit transform/time mapping. A clip is derived current data and cannot retain an obsolete assignment as an alternative authority.

Replacing the current document removes stale segment/membership references and invalidates dependent evidence and prepared corpus data. Mapping correction preserves the current document/segments while invalidating dependent selection/preparation. Physical cleanup retires removed managed clips/manifests after leases and reference barriers allow deletion. Source media and supplied subtitles remain immutable.

## Identity, model lineage and correction

Names and aliases use explicit IDs, language/scope and provenance. Equal name text does not merge people automatically. Shared commands persist canonical names and alternate spellings with language, scope, provenance and active state. Inactive identities and spellings are excluded from active recognition context. Broad merge/split and similarity retain separate delivery contracts; they use the same current-reference rule when delivered.

A merge or split updates current mappings and invalidates affected corpus preparation. Completed model weights/checkpoints can retain originating source IDs, hashes, run identity and staleness diagnostics. They do not preserve an old transcript or assignment list. An invalidated corpus cannot be used again without resolving current evidence and rebuilding its preparation. [Speaker audio and voice models](voice-models.md) defines those relationships.

Configured model-generated assignments can be applied automatically when their contracts validate. Users can correct them without reviewing every voice or segment. Optional embeddings and model comparisons name their actual input/model provenance; a prediction derived from an earlier assignment is not independent corroboration.

## Specialized terms

The terms contract covers stable IDs, canonical spelling, variants, language/context, active state, revision and optional explicit speaker/alias links. Explicitly selected active speaker names and relevant aliases join active general terms and matching language/context terms when compiling supported recognition hints. Leaving the speaker selection empty does not elect every catalog identity. IDs govern the join, not accidental text equality.

Compile and deduplicate hints deterministically within the chosen recognizer's supported UTF-8 byte budget. The compiled result records source revisions separately from the digest of the canonical effective hint array, and reports unsupported, duplicate, inactive, filtered or budget-omitted candidates. Equivalent effective hints retain the same digest even when source provenance changes. Local faster-whisper supports a 200-byte joined hint budget; a hosted worker declares its supported budget up to 8192 bytes. A term is recognition context, not evidence that a named person spoke. A vocabulary rerun replaces the current subtitle result after acceptance; guided/unguided alternatives and obsolete text are not retained as selectable transcripts.

## CLI and desktop parity

Saved identity, alias and terminology operations use the shared runtime and desktop `Operate` bridge:

```sh
insonic speakers list --json
insonic speakers show SPEAKER_ID --json
insonic speakers set SPEAKER_ID --input speaker-update.json --json
insonic speakers aliases SPEAKER_ID --json
insonic speakers aliases SPEAKER_ID --input alias-update.json --json
insonic speakers select SPEAKER_ID --input selection.json --json
insonic speakers diagnostics SPEAKER_ID --input selection.json --json
insonic terms list --json
insonic terms show TERM_ID --json
insonic terms set TERM_ID --input term-update.json --json
insonic terms compile --input context.json --json
```

Speaker updates carry `expected_revision`, `speaker` and the complete replacement `aliases` array. Alias-only updates carry `expected_revision` and `aliases`; omission of input reads aliases. Term updates carry `expected_revision` and `term`. Zero expected revision creates a new identity; positive revisions prevent overwriting concurrent edits. Mutations require the payload ID to match the command ID. Equal spelling never merges IDs automatically.

A term mutation can be as small as:

```json
{
  "expected_revision": 0,
  "term": {
    "id": "77777777-7777-4777-8777-777777777777",
    "canonical": "Specialized vocabulary",
    "variants": ["Alternate spelling"],
    "language": "en",
    "state": "active"
  }
}
```

Use the returned revision as `expected_revision` for the next edit. Alias arrays belong to the speaker aggregate; each alias has its own `id`, matching `speaker_id`, `text` and optional `language`, `scope`, `state` and `provenance`.

Speaker selection and diagnostics accept optional `cursor`, `limit` (1 through 100), `recording_id` and `quality`. A `quality` object accepts `enabled`, defaulting to true; `{"quality":{"enabled":false}}` suppresses optional correlation diagnostics. The response reports the effective setting. Current-reference validation, timing resolution, deduplication and explicit untimed participation remain mandatory with diagnostics disabled. Current evidence binds recording, document, mapping, source and source-map revisions/digests. Intervals are computed from current embedded assignments when requested, with overlap and untimed participation disclosed. Repeated source evidence is deduplicated across cursor pages; changing page size preserves unique spans and aggregate diagnostic counts. Missing timing does not yield an invented clip. Changing the current recording or mapping invalidates an old selection cursor rather than silently returning mismatched evidence. Selection prepares references and measurements; it does not train a model or publish audio clips.

Term compilation accepts optional `pipeline_id`, `pipeline_revision`, `speaker_ids`, `language`, `context` and `max_hint_bytes`. It can run without input, using active general terms and a default 200-byte budget. A selected pipeline supplies the recognizer's capability, budget and configured hints. When no language filter is supplied, compilation uses that recognizer's language. Pipeline inspection applies the same rules to its effective saved or overridden stage, so its hint preview matches a generated recognition election. Compilation runs no model and proves no speaking identity. Dedicated pipeline/speaker/terms GUI controls follow the desktop delivery contract. Corrections use the same IDs, expected revisions and current-reference validation in every client.

## Declared recording rosters

Add and remove require at least one speaker reference in every interface; blank edits fail without changing roster authority. Replace accepts an explicit empty list, and clear declares an empty roster.

A declared roster expresses intended speaker context for a recording, including one with no transcript. Its header anchors to the stable library recording ID; each unique membership references an existing centralized speaker ID. It stores no copied names, profiles or assignments. An undeclared roster has revision zero; an explicitly empty roster has a positive header revision. Names and active/inactive states are read from the current speaker catalog.

```sh
insonic recordings roster show RECORDING_ID --json
insonic recordings roster add RECORDING_ID --speaker SPEAKER_ID --speaker "Exact alias" --expected-revision 0 --json
insonic recordings roster remove RECORDING_ID --speaker SPEAKER_ID --expected-revision REVISION --json
insonic recordings roster replace RECORDING_ID --speaker SPEAKER_ID --expected-revision REVISION --json
insonic recordings roster clear RECORDING_ID --expected-revision REVISION --json
insonic media import session.wav --known-speaker SPEAKER_ID --known-speaker "Exact alias" --json
```

Mutations compare the independent roster revision and reconcile repeated request IDs. Set no-ops preserve the roster revision. References resolve by UUID, unique exact canonical name or active alias; ambiguity and inactive new elections fail. Use `name:` for a UUID-shaped name. Existing inactive members remain visible and removable by ID. Membership is bounded to 1,000 speakers and selectors to 512 UTF-8 bytes. An empty replacement or clear declares an empty roster.

Initial known-speaker references resolve once when work is queued; later alias edits cannot retarget the frozen IDs. Audio and membership publish in one current transaction, leaving no orphan roster on failure. Transcript or mapping edits preserve membership. Audio replacement explicitly retains or clears an existing roster. Portable snapshots validate parent identities, unique membership, revisions and accepted current proofs on SQLite/PostgreSQL. Graph projection exposes separate `declared-speaker` context edges.

The desktop Library exposes initial roster input, a current roster table and add/remove/replace/clear controls using the displayed revision. A conflict refreshes current roster authority and reports the failed election. Declared membership does not assign a transcript voice, create training evidence, run acoustic matching or infer how many speakers actually spoke. Automatic matching remains a separate delivery contract.
