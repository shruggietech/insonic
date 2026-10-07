# S005 data model

## Current recording

One record keyed to the library entry, source digest, current revision, state (ready/no-speech/no-timed-subtitles), embedded Cue JSON or explicit null, exact document digest, current mapped-audio publication, source map, noncontent processing provenance and diagnostics. Only this record stores assignments. Cue JSON follows exact upstream local schema. Absolute metadata instants retain exact int64 nanoseconds; original-clock mapping uses explicit integer/rational units.

## External mapping and evidence

Recording/local speaker UUID maps to known SpeakerID, with current document identity and revision. Mapping correction preserves document text/bytes. Segment/evidence stores current recording/document/cue/local speaker references and optional clip publication; never copies timed assignments/attribution arrays. Replacement removes absent local mappings and invalidates obsolete dataset/preparation/evidence before reuse. Legacy source artifacts remain immutable.

## Work and cleanup

Existing Work owner/generation/lease/cancellation fence applies to processing/reruns. Persistent payload/results/checkpoints contain bounded configuration, input references, identities/digests and diagnostics only. Temporary generated subtitles/turns are private bounded scratch or memory. Atomic accepted current commit queues superseded publications in Cleanup, and reference-fenced physical retirement is retried after restart. Latest accepted current-record/mapping/cleanup proofs govern snapshots.

## Model and fixture manifests

Registered base-model files use exact confined relative roles/names, digests/sizes/upstream revisions/licenses. Worker reconstructs managed model directory, never resolves an ambient latest model. Fixture manifest separately records original source/rendition/revision, creator/license, full and committed hashes/sizes, crop/tool recipe, measured streams/source mapping and independently checked reference facts.
