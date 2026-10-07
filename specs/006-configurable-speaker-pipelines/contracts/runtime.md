# S006 runtime and worker contracts

All runtime requests use the existing version 0.0.0 envelope, workspace/request identities, operation, optional item ID and bounded data. Mutations bind operation ID and expected revision. Results fit paged response limits and status/errors contain no resolved credentials.

## Shared operations

- `pipelines.list`, `pipelines.show`, `pipelines.set`, `pipelines.inspect`: current definitions, CAS mutation and effective configuration/capability inspection.
- `speakers.list`, `speakers.show`, `speakers.set`, `speakers.aliases`, `speakers.select` and `speakers.diagnostics`: current identity metadata and current reference-resolved evidence.
- `terms.list`, `terms.show`, `terms.set`, `terms.compile`: current terminology and bounded effective recognition context.
- `recordings.process`: optional selected pipeline ID/revision plus full-stage overrides and context scope; resolve election at enqueue. Existing direct local options remain supported.
- `recordings.map-speaker`: existing document/mapping revision fencing remains authoritative.

CLI uses plural group/action and `--json`; desktop `Operate` shares the same envelope and validation. Canonical command shapes are:

| Operations | CLI shape and data |
| --- | --- |
| Entity lists | `pipelines/speakers/terms list [--input JSON]`; optional `{after_id,limit}`; output `{items,next_id}` |
| Entity reads | `pipelines/speakers/terms show UUID`; no data |
| Pipeline mutation | `pipelines set UUID --input JSON`; `{expected_revision,pipeline:{id,name,preset,configuration,revision?}}` |
| Pipeline inspection | `pipelines inspect UUID [--input JSON]`; optional `{revision,overrides,context}` |
| Speaker mutation | `speakers set UUID --input JSON`; `{expected_revision,speaker:{id,name,state?,revision?},aliases:[...]}` |
| Alias read/replacement | `speakers aliases UUID [--input JSON]`; omitted input reads, replacement `{expected_revision,aliases:[...]}` |
| Current evidence | `speakers select/diagnostics UUID [--input JSON]`; optional `{cursor,limit,recording_id,quality:{enabled?}}` and compound `next_cursor` |
| Term mutation | `terms set UUID --input JSON`; `{expected_revision,term:{id,canonical,variants?,language?,context?,state?,speaker_id?,alias_id?,provenance?,revision?}}` |
| Hint compilation | `terms compile [--input JSON]`; no item ID, optional `{pipeline_id,pipeline_revision,speaker_ids,language,context,max_hint_bytes}` |
| Processing election | `recordings process UUID --input JSON`; optional `{pipeline_id,pipeline_revision,overrides,context}` with stage modes; direct model/option fields remain supported without a saved pipeline |

Mutations require entity ID equality with the envelope item ID. Expected revision zero creates; positive expected revision updates only the saved revision. Returned entity revisions are workspace-monotonic, so clients use returned values rather than assuming a new entity always has revision 1. Entity page limits are 1 through 100; evidence selection defaults to 25 and bounds pages at 100. Omitted aliases fields default active state, empty language/scope and empty provenance; omitted term variants become an empty array.

Pipeline `configuration` is `Definition{recognition,diarization,quality?}`. Stored presets are lowercase `local|connected|custom`; display labels are Local, Connected and Custom. Each supplied override is a full stage or quality replacement. The registered schema exposes reusable configuration/stage/options definitions and `ValidatePipelineConfiguration` compiles the bare configuration fragment offline. Limits and quality may be omitted at input and normalize at election. Inspection performs no network call or model initialization.

Names, aliases and terms are captured in one consistent catalog snapshot at enqueue, and compiled hints and effective definitions/tool configuration are immutable per work election. Source/context snapshot digests retain provenance separately from the effective hints digest. Global compilation defaults to 200 UTF-8 bytes and does not implicitly elect all speaker identities; explicitly configured budgets are bounded at 8192 and further constrained by recognizer capability.

## HTTP worker protocol 1

Explicit operator endpoint accepts multipart POST containing `request` JSON metadata and `audio` mapped WAV. Metadata declares protocol/version/operation/model/options and source sample facts. Bearer authorization is resolved from selected credential ID only. Redirects are refused; endpoint userinfo/query/fragment credentials are invalid. Bound upload, result, duration and response parsing.

Recognition response contains `segments` with integer mapped-relative `start_us`, `end_us`, text and `no_speech`. Diarization contains `turns` with label/integer mapped-relative timing and `no_speech`. Insonic validates/project timing, supplies local provenance, derives recording-local UUIDs and assembles the current document. Worker-supplied documents/identities/provenance are not accepted. Unsupported declared capabilities fail before the affected stage. Protocol fixtures execute no inference.

### Protocol example

The JSON `request` field for one mapped one-second audio file can be:

```json
{
  "contract_version": "1",
  "operation": "transcription",
  "model": "operator-selected-model",
  "audio": {"sample_rate": 16000, "sample_count": 16000, "format": "wav-pcm-s16le-mono"},
  "options": {"language": "en"}
}
```

The multipart `audio` field contains the actual mapped WAV, not a source path or remote locator. A valid recognition response is:

```json
{
  "contract_version": "1",
  "operation": "transcription",
  "segments": [{"start_us": 0, "end_us": 500000, "text": "Example text"}],
  "no_speech": false
}
```

A valid diarization response uses `operation: "diarization"` and `turns: [{"label":"voice-a","start_us":0,"end_us":500000}]` instead of segments. A no-speech response has an empty selected result array and `no_speech: true`. Version/operation mismatches, unknown fields, duplicate keys, contradictory no-speech flags and out-of-mapped-audio bounds fail before current acceptance. Stage-supplied provenance and documents are not accepted; source/model/tool provenance is generated locally. The production executor refuses CI; deterministic fixtures inject only a bounded loopback client.

Correlation quality defaults enabled; disabling it suppresses optional selection diagnostics while mandatory current evidence validation and selection run unchanged. Responses include the effective `{quality:{enabled:boolean}}` policy. Queued local model manifest digests are frozen and revalidated during execution. Reused diarization quality explicitly names the current embedded millisecond-assignment basis.
