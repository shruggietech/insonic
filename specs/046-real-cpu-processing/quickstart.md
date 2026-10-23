# Validation

1. Prepare pinned Go 1.27.1, current native dependencies and a clean complete Windows CLI package. Run all process tooling hidden and noninteractively.
2. Reproduce the selected licensed fixture corpus from its hash-bound original archive, preserving exact transcript/reader references and declared derivation. Validate its manifest and enrollment/evaluation separation without inference.
3. Run root npm check/test, full Python maintainer tests and affected Go/site checks. Negative corpus, receipt, mode and cleanup tests must fail before their implementations.
4. Outside CI, elect `scripts/qualify-journey.py --mode real --package ARCHIVE --package-receipt RECEIPT --revision SHA --reference-corpus tests/fixtures/acoustic/manifest.json --recognition-model EXACT_SNAPSHOT --diarization-model EXACT_SNAPSHOT --output OWNED_RECEIPT`. This mode prepares managed CPU roles; legacy explicit interpreter arguments are not required for it. The optional `--managed-offline-assets DIRECTORY` selects a complete declared dependency bundle.
5. Require actual preparation/model verification, real corpus processing/enrollment/matching, source-linked retrieval/query/timeline/playback and restore assertions. Inspect both functional acceptance and all measured quality limitations. Bind every identity to the actual package/source and retain the compact receipt; remove owned temporary libraries, archive copies and profiles.
6. Push the official PR, inspect exact-head terminal hosted CI and preparation/qualification/combined timing, resolve at most two Codex rounds and stop before owner merge. Keep #147/#154 open for their remaining criteria.

## Local validation (2026-10-23)

Root npm check/test passed (181 tests). Site build/check/test passed (111 exported pages, 17 tests). The full Python maintainer suite passed 546 tests after explicitly selecting cached Go 1.27.1 and the Windows C locale; the latest 15 focused journey tests also passed. The inference-free processing-worker suite passed.

The broader native CLI/internal Go batch passed every package except backendsetup and process under concurrent load. Isolated backendsetup passed. Windows process tests still observed roughly two seconds in process launch, exhausting their existing 1.5/2.5-second test budgets; their assertions and product deadlines remain unchanged. Hosted Windows qualification must establish its own terminal result. This local limitation is not presented as a passing process suite.
