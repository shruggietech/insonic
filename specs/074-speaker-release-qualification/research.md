# S074 research

## 2026-11-03 decisions

1. **Decision**: Prepare 1.3.0. **Rationale**: Additive shared CLI/runtime speaker operations since 1.2.0 warrant a minor release. Catalog schema 11 migrates to 12 and old binaries reject the upgraded catalog; disclose that boundary. **Alternative**: Patch release understates new capability; no API removal establishes a major release.
2. **Decision**: Separate synthetic contextual semantics from licensed acoustic quality. **Rationale**: Eight known-truth synthetic conversations exercise introductions/collisions/quotes/contradictions/unknowns; frozen independent acoustic clips measure real profiles. **Alternative**: Attaching fictional speech text to licensed recordings and calling it acoustic truth would misstate evidence.
3. **Decision**: Reuse owned-runtime and frozen-corpus helpers in a dedicated speaker campaign. **Rationale**: Existing journey_cpu acceptance is intentionally managed-package CPU-only and must retain historical authority. **Alternative**: Generalizing its validator or omitted-device helpers risks accepting auto/CUDA execution as CPU.
4. **Decision**: Explicit cpu/cuda model elections with actual provenance. **Rationale**: Readiness found a compatible NVIDIA device, pyannote libraries and offline pinned snapshots. **Alternative**: Auto fallback alone cannot establish NVIDIA execution.
5. **Decision**: Local qwen2.5:14b through an owned loopback Ollama server. **Rationale**: A complete cached non-thinking model fits short product discovery deadlines. Record exact manifest/API/model identity and reject drift. **Alternative**: New hosted routing is unnecessary; thinking models risk consuming the unchanged 60-second aggregate discovery budget.
6. **Decision**: Report measurements without a universal pass percentage. **Rationale**: No accuracy threshold is approved, corpus is small and errors are useful. Execution, accounting, independence and integrity remain gates. **Alternative**: Treating all deterministic expectations as a model accuracy threshold would hide actual quality.
7. **Decision**: Copy forward current docs/schema, then restore published 1.2.0 authority; obtain fresh 1.3.0 captures on hosted isolated desktops. **Rationale**: Current feature docs are newer than the published release and screenshot identities are immutable. **Alternative**: Relabeling historical images or docs is false provenance.
8. **Decision**: Keep package inference metadata not-executed and attach elected receipts separately. **Rationale**: Required CI deliberately uses controlled models; release loader/schema enforce that distinction. **Alternative**: Editing package metadata would misrepresent what packaged acceptance executed.
9. **Decision**: Reuse current installed speaker/recovery/backend suites and no new tracking issue. **Rationale**: #233 is the milestone acceptance task and #211 closes after its full criteria are met; code already implements the workflows.
10. **Decision**: Automatic push/official PR with at most two review rounds, owner merge. **Rationale**: Explicit current user authorization supersedes the autopilot skill's default pre-push pause. Publication remains separate.

## Evidence entry points

- scripts/acoustic_reference.py and tests/fixtures/acoustic: frozen licensed reader/reference identities, distinct enrollment and held-out chapters, unseen reader, independently defined metrics.
- scripts/journey_cpu.py and scripts/qualify-journey.py: installed model/tool/runtime authority and current receipt validation.
- scripts/cli_speaker_workflow_qualification.py: installed configured application/enrollment, correction/retraction, cancellation/retry/restart; controlled fake weights labeled not-measured.
- internal/app/speaker_discovery.go; internal/speakerdiscovery/parse.go; internal/assistance/ollama.go: provider budget, strict citations, current request shape and configured local generation.
- scripts/release.py; site/lib/screenshots.mjs; specs/070-v1-2-release-preparation: version preparation, historical restoration, capture provenance and terminal retention.
