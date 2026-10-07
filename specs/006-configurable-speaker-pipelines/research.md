# Research decisions: S006

2026-10-07, before implementation.

## Current catalog entities

Decision: typed current pipeline/speaker/alias/term entities with CAS revisions and snapshot proofs. Reuse catalog write/replay and existing mapping reconciliation. Alternatives: machine-local files or generic settings history would lack bounded entity listing and direct admission/proof semantics. Frozen schema4 plus schema5 preserves historical restore identity.

## Elected processing

Decision: immutable effective election in work payload; selected stage models/options/routes and context are fixed at submission. Alternatives: resolve mutable defaults while executing would change an elected job without a new request. Full-stage override prevents stale route credentials surviving a mode change.

## Hosted protocol

Decision: document synchronous Insonic HTTP worker contract1, implemented with actual bounded multipart HTTP fixture tests. Recognition returns timed text, diarization returns temporary voice turns. Credential resolution occurs in memory only; no redirects/raw errors/provider provenance persistence. Alternatives: treating arbitrary text endpoints as audio-capable is false; multiple vendor-specific APIs add unnecessary parallel contracts. Connected and Custom use explicitly selected compatible workers.

## Context and quality

Decision: deterministic active term/name/alias compilation with explicit scope/language and UTF-8 byte budget. Local recognition uses initial_prompt; hosted worker metadata carries supported hints. Preserve effective digest and omitted/unsupported diagnostics. Quality defaults on, remains distinct from schema/timing correctness, and never creates an approval queue.

## Speaker evidence

Decision: current mappings and document digests drive paged library-wide evidence selection. Resolve intervals from embedded assignments/source map at read/use time, keeping stored segment records reference-only. Cosmetic identity/context changes affect future hints; mapping/result changes invalidate dependent preparation through existing reconciliation. Alternatives: copied assignment datasets violate the current-result contract.

## Shared clients

Decision: bounded JSON CLI input and versioned shared runtime operations; desktop generic Bridge.Operate already uses this surface. Dedicated GUI panels remain #10. All required tests use synthetic credentials and deterministic stages, never real inference.
