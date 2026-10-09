# Shared speaker work contract

## Runtime operations

- `models.dataset.create`: speaker ID plus bounded recipe; resolve all current pages server-side, freeze independent attribution and return dataset summary/ID.
- `models.dataset.list/show`: current/invalidated dataset summaries and reference authority.
- `models.train`: exact dataset, output family/name, selected local/hosted adapter declaration, immutable base reference if needed, parameters/limits and optional exact compatible checkpoint. Queue renewable durable work.
- `models.speaker.list/show/fetch`: speaker-filtered discovery, exact version inspection and exact artifact retrieval to an explicit output directory. Report hosted-only/no-download and missing objects.
- `models.profile.set`: revisioned speaker-to-exact-version association, including clearing current profile.
- `recordings.match`: recording plus explicit matching adapter/model and threshold/margin/minimum evidence. Freeze roster/current profiles and evidence before queuing; no transcription/diarization stage.

Inputs use the existing strict versioned Request envelope, request UUID and workspace authority. Excess fields and foreign references are rejected. CLI and desktop call these same operations; work list/show/cancel/retry supply shared lifecycle.

## Adapter contract

Local adapters use a pinned hidden noninteractive executable with literal arguments and bounded JSON I/O. Hosted adapters use explicit HTTPS or configured loopback HTTP, no redirects, encrypted credential lookup and bounded JSON/media bodies. The declaration names contract version, architecture, supported output kinds/consumers, preparation needs, train/resume/cancel capabilities and rights. Training receives exact prepared input paths/bytes, parameters/base and producing attempt. Output includes declared immutable artifacts with digest/size/role/format or explicit hosted-only handle; checkpoints additionally bind step/base/resume compatibility. A failed local operation cannot elect hosted routing.

Embedding enrollment is a declared profile output distinct from parameter training. Its offline pinned consumer extracts current selected clips and publishes a versioned normalized profile. Matching uses only compatible frozen roster profiles and exact current local-voice clips. Score semantics and thresholds are explicit; insufficient evidence and ambiguous candidates remain unknown/ambiguous.

## Acceptance and retrieval

Publish configured objects first, then atomically revalidate work and input authority before catalog acceptance. Stale/cancelled claims cannot publish accepted versions/mappings. Failure retires owned scratch/unaccepted publications through existing cleanup contracts. Exact fetch verifies every artifact and confines destinations, excludes secrets and writes portable output metadata without substituting versions.
