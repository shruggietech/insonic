# S011 recording update contracts

Shared import additions: targeted media `record`, `replace_audio`, `existing_transcript=keep|clear|replace`, `transcript_applies`, `existing_roster=retain|clear`, repeated `known_speakers`, observed source/document/roster revisions and frozen resolved speaker IDs. Exact names/active aliases resolve once; ambiguity is conflict. Supplied new roster cannot silently replace a targeted recording's declared roster.

CLI concepts: `media import SOURCE --record ID --replace-audio --existing-transcript keep --transcript-applies --existing-roster retain`; initial `--known-speaker REF` repeats. Existing transcript commands remain unchanged. JSON and CSV manifests use the same elections and defaults/item precedence, with arrays represented explicitly in CSV cells.

Roster family: `recordings roster show ID`, `add/remove/replace ID --speaker REF` (repeatable) `--expected-revision N`, and `clear ID --expected-revision N`. Shared versioned runtime operations receive target, independent expected revision and selectors; results show declared state, revision, memberships/current names/state. Desktop wraps these operations and uses displayed revisions.

Invalid/contradictory parameters fail before publication. Known stale target/document/roster causes conflict; source target without audio permission skips without acquisition. Concrete configured replacement parameters need no extra confirmation. All accepted results bind work/ordinal/stable recording and current source/document/roster authority. No inference success is reported for candidate admission.
