# Answer workflow contracts

## CLI and runtime

- `answers request --input answer-request.json` -> `answers.request`: admit durable work and return work ID/effective election.
- `work wait WORK_UUID`, `work show WORK_UUID`, `work results WORK_UUID`, `work cancel WORK_UUID`: existing durable operations remain authoritative.
- `answers retry WORK_UUID` -> `answers.retry`: create linked retry retaining frozen election; `work retry` on answer work uses the same semantics.
- `answers show PACKAGE_UUID` and `answers export PACKAGE_UUID`: support existing evidence packages and new grounded answer artifact IDs with kind-specific result and live freshness.
- `answers prepare` and `answers validate` preserve existing behavior. New synthesis claim kind is additive; strict typed synthesis requires it.
- `pipelines set/show/inspect`: optional answer_synthesis configuration supports answer-only or combined definitions and explicit local/hosted route consistency.

Answer request data contains `request` (existing evidence request), optional `configuration` (provider plus repair policy), or `pipeline_id` with optional `pipeline_revision`. Evidence request provider metadata is derived from the frozen election, never a second route selector. Generated provider request contract is answer-synthesis-v1. Provider responses are bounded structured JSON envelopes with exact package ID, states, typed claims, supporting citations and limitations.

## Limits and errors

Provider request cap131072bytes, configured response default262144/max524288bytes, claims128, distinctreferences128, maxrepairs1, synthesis deadline configured default30/max60seconds shared by both calls. Existing retrieval limits remain unchanged; overall work cap180seconds. Oversized serialized input fails with input_limit rather than silently dropping evidence. Provider errors use generated categorized reasons; raw error bodies remain absent.

No HTTP redirects, no tool calls, no provider-selected routes. Local Ollama verifies model inference route before sending evidence and checks returned remote indicators. Configuration changes affect future submissions; credential values resolve from the encrypted store at execution.

## Publication authority

Artifact publication is provisional until a catalog transaction establishes matching current source epoch, live work owner/generation and verified publication metadata. Restart reads the candidate journal and reconciles actual publication; accepted work receipts are idempotent. Cancellation and changed evidence cannot commit. Exports include exact references and safe provenance, with separate acceptance and freshness.

## Compatibility

Existing preparation and standalone envelope validation remain compatible. New CLI/runtime/schema fields are additive. Existing pipeline definitions continue to validate. Dedicated answer GUI controls are deferred explicitly in public CLI documentation; shared runtime remains the authority. Current release versions stay aligned at1.1.0; recommended next release1.2.0.
