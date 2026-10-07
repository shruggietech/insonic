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

PR25 is published: https://github.com/shruggietech/insonic/pull/25. Product commit `6d020b9d76bfabc7c6f7fa34e5189b4e3a935b20` is pushed and both target Project items are In review. Initial automatic Code Review completed with one P2 finding, discussion4213063626: omitted local recognition language defaults to English during execution but previously compiled an empty-language context. The regression fails before the fix. Shared normalization now elects/reports English before preview/compile/job context, preserving explicit languages, auto and hosted omission; direct local processing is covered too.

Initial adapter-fixture CI passed on6d020b9 (run37702175848), including the configured PostgreSQL/S3/ArcadeDB gate. Final-head three-platform CI remains required. Initial finding4213063626 is fixed in2176892389a24db97eaa083c21382c46f6e59673, replied to in4213079843 and its thread resolved. Full app acceptance9.022s, affected app race1.812s, app vet and repository/schema checks pass on the fix.

Exactly one combined second code/security request was posted: issuecomment6048961869, https://github.com/shruggietech/insonic/pull/25#issuecomment-6048961869. Code Review completed on2176892 with one P1 finding, discussion4213099084: paginated selection reset duplicate/overlap state. The real app regression imports the same committed source three times, produces two identical spans and one overlapping span with deterministic callbacks, maps all three recording-local UUIDs to one person and compares aggregate page sizes100/1/2. It fails before the correction (three spans/no diagnostics for limit1) and passes afterward (two spans/duplicate1/overlap1 for every page size).

The correction batches comparison flags per current reference against all earlier current embedded evidence inside the database. Flags contain assignment ordinals and scalar facts only; cursors contain no copied assignment arrays. Source digest/stream/channel and original integer milliseconds govern comparison, and unknown streams remain recording-specific. Page-epoch and exact-reference fences reject relevant concurrent changes, while unrelated heartbeats remain valid. Global voice diagnostics count once per recording; untimed participation and diagnostics disabling preserve mandatory deduplication. Shared SQLite/PostgreSQL regressions cover intracue/crosspage/cross-recording duplicates, overlap, scope, unknown streams and stale epochs.

Full Go acceptance/vet pass after this correction (app12.441s, catalog11.158s, artifact13.221s). Affected app/speaker races pass1.896s/1.066s; catalog comparator/epoch races pass1.334s. Integration tags compile and the PostgreSQL comparator regression is registered. Final hosted execution remains required. No separate Security Review report/run appeared despite the combined request. No third review is requested and no bot approval of the final fix is claimed.
