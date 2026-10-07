# S006 verification and delivery

Actual slice S006, branch `codex/006-configurable-speaker-pipelines`, issues #8/#9. Owner authorized autonomous push and official PR publication, two review rounds maximum, and owner-controlled final review/squash merge. No release or merge is performed by this slice.

## Implementation evidence

- Catalog schema5 freezes schema4 and admits current pipeline, speaker/alias and terminology domains with CAS, replay and latest snapshot proofs. SQLite and PostgreSQL share identity/selection contract suites, including unrelated-heartbeat continuation and relevant-mapping cursor invalidation.
- Shared app/CLI/desktop operations expose all three presets, declared capabilities, full-stage overrides, current evidence selection and optional correlation diagnostics. Known-person mappings remain separate from recording-local UUIDs and do not alter embedded document bytes.
- Jobs freeze effective route, context, resolved tool configuration and selected local manifest digests. Recognition and materialization recheck the frozen digest; unsupported capability, failed hosted route and mismatched payload do not elect alternatives. Preview and execution share the context compiler.
- Local hotwords apply across windows with conservative200-byte joined budget; declared hosted adapters support up to8192 bytes. Deterministic ordering, effective digest and bounded omission details are explicit.
- Automatic diarization quality defaults on and remains separate from mandatory timing/document acceptance. Reused diagnostics derive from current millisecond-quantized embedded assignments and elected settings, with an explicit basis. Changed audio mapping rejects reused assignments before inference or replacement.
- Existing current-result replacement, physical retirement, original-source retention and immutable noncontent lineage remain enforced. No duplicate durable transcript/assignment store is introduced.

## Local checks

2026-10-07 Windows:

- `npm run check`: PASS,450 project text files,19 schemas,17 contracts and21 examples. Includes UTF8/noBOM/LF/mojibake, links, versions and governed brand integrity.
- `npm test`: PASS55/55 deterministic maintainer tests.
- Python qualification unittest discovery: PASS9 tests. Deterministic processing-worker argument/quality tests: PASS14 tests; no engine imports or inference.
- `go test ./... -count=1` and `go vet ./...`: PASS all packages. Full affected races: app86.717s, pipeline1.182s, processing1.167s, speakers1.106s; catalog50.109s and artifact14.167s. CLI/schema/contracts/runtime/desktop full races also pass. Final catalog snapshot-proof refinement passes focused race2.332s; integration-tag catalog compiles.
- Actual cross-process CLI owner reuse/configuration/scoped-hints/artifact/jobs/catalog-roundtrip: PASS. Production desktop bridge, packaged help and actual native webview frontend IPC: PASS. Public and offline site export27 pages each, including links/anchors: PASS. Ignored `build/native/s006-final-client-receipt.json` binds exact built binaries and relevant inputs; inference is not-run.

The pre-publication integration audit found preview-language/hint divergence, missing local manifest binding and incompatible-map diarization reuse. They received explicit fixes and deterministic regressions before publication. Historical artifact cleanup tests were adapted to the actual frozen schema4 speaker shape while retaining failed-readback, physical-retirement and unavailable-backend obligations. Empty context hints are normalized across durable JSON omission and replay.

## Convergence

Spec Kit prerequisite checker resolves the requested S006 directory with spec/plan/tasks/contracts. Code inventory covers all16 functional requirements and all four stories. No new build gap remains after the integration corrections; the existing T028/T031..032 hosted verification and delivery gates remain pending. No additional tasks are appended by convergence. Real engine/provider service behavior is not claimed from fixtures. Dedicated GUI controls, similarity embeddings, training engines and installable releases remain outside #8/#9.

## External delivery

Not published yet. Hosted three-platform CI and alternative backend qualification must pass on the final PR head. Initial automatic review is round1; at most one combined `@codex review`/`@codex security review` follow-up is round2. Every received finding requires a pushed disposition, reply and resolved thread before owner handoff. No third review request is authorized.
