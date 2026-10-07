# S005 research decisions

## Exact upstream document contract

Decision: exact Cueson1.2.0 executable boundary and immutable schema; no imports of upstream internal Go packages.
Rationale: actual Windows package encode/inspect/render verified by research agent. Tag revision6a1fd7a541767c0b19dcc10022fafc0e46903cff; schema191170bytes SHA256f2661a3d52effbab4a82a4d47197b5c7fae58496dc30a397ea3f2f668358b654.
Alternatives: handwritten native parser/renderer rejected because upstream owns restoration/normalization/loss behavior.
Semantics: cue-contained attribution with paired int64 absolute-ms endpoints or untimed participation; overlap/duplicate occurrences retained; known duration separate from cue coverage; no-cue SRT/VTT cannot fabricate document. Native export can refuse loss accounting beyond8192 omissions.
Use raw JSON field projections to preserve exact nanoseconds/unowned fields and cue IDs.

## Local engines

Decision: lazy Python workers, exact faster-whisper1.2.1/CTranslate2 4.8.2 and pyannote.audio4.0.7, separate elected Python environments, CPU default and explicit CUDA without fallback.
Rationale: actual locally available environments and cached Whisper tiny / Community-1 model trees provide attainable explicit maintainer qualification. Go service materializes verified registered models by confined relative file roles; never downloads inside workers.
Alternatives: invoking large model servers/gated hosted processing would add unrelated service/routing prerequisites; fake diarization or transcription output is unacceptable.
Offline env, bounded JSON input/output, closed/noninteractive supervised processes and process.Capture prevent output intermixing; credentials stay out of workers unless an explicitly configured adapter needs them.
Research model identities: Whisper tiny revisiond90ca5fe260221311c53c58e660288d3deb8d356; Community-1 revision3533c8cf8e369892e6b79ff1bf80f7b0286a54ee. Exact files/hashes are qualification manifests, not test-suite auto-loads.

## Fixtures

Decision: lossless CC BY3.0 Speech_12dB_s16.flac and a30s Sintel dialogue candidate134-164s from a smaller source rendition, committed directly to Git.
Rationale: licensed real audio/video, intelligible speech/dialogue and small repository cost. Commons audio revision973705085; video description1212979998 and English subtitle source175198974. Verify original file/source and actual audible turns/crop properties before final selection.
Alternatives: historical presidential recordings are public domain rather than CC-licensed; a full4K movie is unnecessary. Do not confuse source page licenses with asset licenses or mislabeled other subtitle tracks.
Source adaptation/license/hash/probe records and independently checked expected text/timing belong alongside bytes. Tests use fixture annotations only as test oracles.

## Current authority and CI

Decision: current recording embedded document, external known mapping, reference-only consumers, digest-only accepted receipts/journals and recoverable physical cleanup.
Rationale: owner #21 overrides old alternate/frozen assignment retention; reject rollback/omission via latest proofs and preserve original sources.
Decision: no inference, model imports/load or weights in any CI path. Real engines only explicitly elected maintainer qualification outside requiredchecks.
Rationale: owner expense constraint is categorical, including cached/small models; deterministic tests cannot prove model quality.
