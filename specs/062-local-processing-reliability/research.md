# S062 research

## Compatibility

Decision: Normalize new local recording/training/matching admissions to explicit auto, retaining empty execution fields as CPU. Rationale: old frozen work can contain empty devices because CPU defaulting previously occurred only inside processing/adapters.go. Alternative rejected: changing the execution empty default would silently retarget durable work. Train validation is reused at execution, so normalization belongs at admission after saved profile resolution.

## Shared worker and native readiness

Decision: Keep one generated device/fallback vocabulary for transcribe, diarize, embed and embed-batch, with bounded observations. Readiness and execution use elected verified native-library roots. Rationale: bare ctypes lookup currently differs from actual recognition and every CUDA exception is mislabeled missing libraries. Avoid ambient module discovery; account for embedded probe execution and standalone pinned worker packaging.

## Speaker provenance

Decision: Return embedding vectors plus execution observations through callbacks, persist enrollment manifest/work and matching result/catalog provenance, and preserve vector format. Rationale: both current embedding functions discard provenance and hardcode CPU. Changing only the device literal would not meet issue #204.

## Silence and errors

Decision: Reuse existing no-speech acceptance with null document; do not fabricate an empty invalid Cueson document. Reject known unsupported local recognition languages at admission; hosted adapters remain independent. Preserve only allowlisted generated failures. Unknown engine exceptions currently become engine_failed and are lost by the Go allowlist, so both boundaries need repair.

## Operator feedback

Decision: Concise task-specific work output; preserve generic public query/settings fields. Rationale: global removal of humanFields would break existing output contracts. Gate actions by real state; only interrupted/timed-out unobserved waits may synthesize continuation targets. Preserve ErrRetirementDeferred while adding distinct retention/lease/grace/owner reasons.

## Documentation and verification

Decision: Correct models inspect to models show/verify, reconcile whisper.cpp availability and new device policy, and extend maintained executable workflows. Existing help qualification does not extract commands embedded in prose; the changed setup journey needs an explicit fixture. No new backend or model quality claim is introduced.

## Final fallback placement

Decision: The shared Go worker launcher performs CUDA then a single CPU retry with fresh children under the same worker context. The Python worker executes one concrete device and reports bounded generated failures. This improves CUDA resource cleanup over reusing a failed inference process. Probe and worker use the same embedded native-library helper injected into an in-memory module, avoiding ambient imports. Maintainer direct-worker qualification uses the same explicit helper/bootstrap.
