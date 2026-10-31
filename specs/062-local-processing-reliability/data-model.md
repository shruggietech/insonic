# S062 data model

## Device policy and execution

New local request device values are auto, cpu or cuda. Omission normalizes to auto at admission only. Persisted legacy omission means cpu. Explicit CPU never attempts CUDA. Auto and CUDA allow one CUDA attempt followed by at most one CPU attempt for a classified CUDA failure within the same enclosing deadline.

Accepted worker provenance keeps device as the actual concrete cpu/cuda value and adds requested_device, attempted_devices (one or two concrete device values in execution order) and optional fallback_reason (allowlisted generated category). The attempted sequence is either [cpu], [cuda] or [cuda,cpu]; fallback requires the last sequence and actual CPU. Model digest and source/sample identity are independent and unchanged. Legacy provenance without these optional fields remains readable.

## Speaker execution

Training configuration and matching input carry the same device election only for local pyannote-profile. Embedding callbacks return vectors plus bounded provenance. Enrollment manifest/work and matching catalog/work results retain observations without changing embedding format or profile compatibility. Historical manifests without observations remain valid.

## Diagnostics

Known language failure names recognition.language. Device/dependency/resource/unknown-engine failures use generated categories, never raw text. Exact digital silence and recognition/diarization disagreement are bounded observations. Artifact refusal reasons distinguish retained references, active leases, grace and another retirement owner; existing conflict/state fencing remains.
