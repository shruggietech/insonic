# Data model: Grounded answer workflows

## Request and frozen election

An answer workflow request contains a normalized existing evidence request and either an optional explicit synthesis configuration or a saved pipeline ID/revision. Omission selects saved assistance configuration. Mutually exclusive elections are validated. The frozen payload includes normalized effective configuration, pipeline ID/revision, configuration digest, request digest, stable publication UUID and optional parent work UUID. Credential IDs are opaque; secret values are never serialized.

Synthesis configuration reuses assistance Config and adds max_repairs (0 or1, default1 via omitted pointer). Provider execution uses one aggregate configured timeout (default30000ms, max60000ms), response cap524288bytes, serialized request cap131072bytes, maximum128claims and128distinct cited evidence references. Full retained grounded answer cap1048576bytes. Aggregate answer work deadline180000ms.

## Provider request and claims

Provider request contract answer-synthesis-v1 binds package ID, question, visible cue extracts, authoritative references, source-family IDs, conflicts/limitations and generated repair reasons. Only actual context cue nodes are eligible. Prompt/schema are fixed and source text is data. Provider output uses existing Envelope plus required claim kind quote or paraphrase for synthesis. Quote claim text must match a supporting citation quote exactly. Existing manually validated envelopes may omit kind for compatibility.

Envelope states remain complete, partial and insufficient (cancelled is historical existing envelope vocabulary). Supported claims require citations; uncertain/conflicting/unsupported claims require reasons and cannot be complete. Complete also requires complete context/evidence. Invalid output never becomes a verified complete answer. Repair exhaustion returns a diagnosed insufficient result with no unsupported accepted claims and retained generated attempt reasons/digests.

## Durable work and candidate

Work kind answers.request follows pending -> running -> succeeded/failed/cancelled/interrupted. Running phases are retrieval, context, synthesis, validation and publication. Lease owner/generation fences every checkpoint and acceptance. Public progress stores only references and bounded counts/route identity. The internal publication checkpoint also retains the exact candidate bytes (at most 1 MiB, base64 encoded within the existing 4 MiB journal limit) until terminal acceptance; accepted full text resides in artifacts.

Before publication, one fenced checkpoint stores exact candidate bytes together with publication ID, content digest/size, package ID, source epoch and selected media IDs. On restart, an available matching artifact is verified and accepted under current work/source authority; confirmed missing or pending publication resumes from independently validated retained bytes without another provider call. Available-object integrity failures stop recovery instead of overwriting evidence. Accepted receipt replay prevents duplicate effects. Cancellation prohibits acceptance even if bytes became available.

Retry creates a new work ID with parent_work_id and the same normalized configuration election; original attempts and their artifacts remain inspectable. A fresh request is required to select changed route/configuration. A retry refreshes evidence under the original scope instead of blessing stale evidence.

## Grounded answer artifact and view

Kind grounded-answer, schema_version1.1.0, UUID, workspace UUID, originating work UUID, package UUID, source epoch, envelope, safe configuration identity, independent-family counts, attempt observations, diagnostics and limitations. No endpoints, credential IDs/values or original private file locations in exported artifacts. Byte identity is verified by artifact publication digest/size. Work acceptance is checked separately from artifact availability.

Inspection/export returns artifact plus acceptance and current/stale/unknown source freshness. Historical bytes never change. An unavailable/cancelled/unaccepted work cannot appear as a current successful answer merely because an artifact exists.

## Evaluation report

Corpus version and SHA256 identify frozen synthetic sources, questions, expected propositions/aliases, required evidence, independence counts and required uncertainty. Reports include mode deterministic or elected-model, exact nonsecret configuration/model identity when elected, metric numerator/denominator/threshold/pass, per-case failures and retained outputs. Expected rubrics never enter provider context. Unknown paraphrases fail the conservative support rubric and remain inspectable. Zero denominators yield null/not-applicable.
