# S011 data model

## Declared roster

RecordingRoster: id is the stable library recording UUID, persisted revision is a positive catalog revision (>0). Zero is reserved for the absent/uncreated expected revision. Missing header means undeclared; persisted header with no members means explicitly empty.

RecordingRosterMember: deterministic membership UUID, recording_id references roster header, speaker_id references centralized speaker, revision equals owning header revision. Unique workspace/recording/speaker pair. No names, profiles, turns or timing arrays. Membership maximum 1,000. Input selectors maximum 512 UTF-8 bytes, exact matching and active identity elections; inactive existing membership is readable.

Roster mutation: immutable operation UUID, target UUID, expected_revision (0 for absent), action add/remove/replace/clear and selectors. Set no-op does not advance roster authority. Revision conflict changes nothing. Response resolves current names/state at read time; membership records contain only references.

## Current source replacement

LibraryEntry retains its ID and owner metadata/date choices while elected source byte identity, canonical track facts, captured reports and publication authority change. Expected library revision and optional current Recording revision are frozen. Current Recording advances for every replacement; ready has validated document/digest, untranscribed has null document/empty digest and no acoustic claims. Mapped processing audio and source-dependent mappings/evidence are deselected.

Admission election includes replace_audio, existing_transcript clear/keep/replace, transcript_applies assertion, existing_roster retain/clear, optional initial known_speakers (nil unspecified versus [] empty), resolved IDs and expected_roster_revision. Replacement candidate and old accepted data remain distinct until one live-work-fenced catalog commit.

## Receipts and portability

Current roster and composite admission proofs bind current header/membership digests and accepted revisions. Historical receipts retain IDs/revision/digests without alternative mutable membership/assignment snapshots. Portable domains and schema migration preserve references and reject duplicate/missing parents, inconsistent revisions and forged current proofs. Graph declared-speaker links are context, not speaking intervals or training inputs.
