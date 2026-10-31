# S062 processing contract

- New local device omission freezes auto. Explicit cpu stays cpu. Explicit cuda permits observable CPU fallback; legacy omitted payloads remain CPU.
- Worker requested_device, attempted_devices, device and optional fallback_reason must agree; maximum two attempts. Unknown/malformed execution claims fail validation.
- CPU fallback is limited to CUDA unavailability, native dependency/compatibility and resource failure. The original worker deadline bounds both attempts. No provider, model, language, source or accepted output substitution.
- Exact zero-signal returns no_speech without invented cues. Nonzero quiet speech and diarizer disagreement cannot erase valid recognition.
- Speaker training/matching preserve sample/model identity and include the same observations in durable results/manifests.
- Normal human work output contains one outcome and state-appropriate actions. Complete JSON remains available; raw engine stderr and sensitive inputs never enter durable errors.
- Public schemas accept old optional-field absence while validating new request/provenance structures. Current docs and desktop request generation agree with admission defaults.

Generated CUDA failure/fallback categories: cuda_no_device, cuda_library_missing, cuda_library_incompatible, cuda_compute_unsupported and cuda_resource_exhausted. Additional worker failures: unsupported_language and engine_failed. Only the five CUDA categories permit device fallback. Raw exception strings never become public messages.

Failed work retains error.device_attempt with requested_device, actual ordered attempted_devices and any generated fallback_reason. It never claims an actual successful device. A CUDA-only receipt can retain its failure cause when cancellation prevents the CPU launch; empty attempt receipts are omitted.
