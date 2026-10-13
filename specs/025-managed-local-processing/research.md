# Research: Managed local processing

## 2026-10-13: Environment and engine identity

Decision: Keep worker pins faster-whisper 1.2.1, CTranslate2 4.8.2 and pyannote.audio 4.0.7. Managed CPU profiles use CPython 3.12.15 from python-build-standalone release 20261009, verified uv 0.13.0 and complete platform-specific hash locks. Recognition adds PyAV 18.1.0; diarization uses torch 2.14.0+cpu, torchaudio 2.11.0+cpu and torchcodec 0.17.0+cpu. Every transitive version/hash is frozen in a product-owned lock.

Rationale: Separate recognition/diarization environments avoid dependency coupling and preserve existing adapter pins. CPU wheels avoid implicit CUDA installation. TorchAudio 2.11 supports torch 2.11 and later; its former equal-minor rule is obsolete. GPU reuse must inspect actual native-library/device availability rather than infer it from hardware presence. Managed dependencies require Linux glibc 2.28 or newer; report incompatibility explicitly.

Alternatives: Global pip modifies unrelated applications. Default uv Python installation can write registry/global integration. Latest unconstrained packages are not a repeatable worker identity. Direct extraction of checksum-pinned standalone Python plus an explicit uv venv/sync changes only owned managed directories. Verified existing environments remain an alternative.

Sources: [Python assets](https://github.com/astral-sh/python-build-standalone/releases/tag/20261009), [uv release](https://github.com/astral-sh/uv/releases/tag/0.13.0), [faster-whisper](https://github.com/SYSTRAN/faster-whisper), [TorchAudio dependencies](https://github.com/pytorch/audio#dependencies), [TorchCodec matrix](https://github.com/pytorch/torchcodec), [pyannote.audio](https://github.com/pyannote/pyannote-audio).

## 2026-10-13: Model identity and acquisition

Decision: Consolidate the already qualified tiny Whisper and Community-1 required files into a builtin immutable model catalog. Expose license/access links, size, exact roles and ordinary acquisition requests. Extend existing acquisition with verified local-file reuse, byte progress and a frozen selected storage profile. Preserve custom manifests, aliases/sources, resumable downloads and independent consumers.

Rationale: Existing models/artifact services already implement exact hashes/sizes, repair, filesystem/S3, leases and durable acquisition. A worker-side downloader would duplicate those guarantees. Existing same-origin redirect handling cannot download Hugging Face CDN weights; add narrowly scoped HTTPS HF/CDN transitions and strip credentials on origin changes. Persist only original immutable locators, never signed redirect URLs.

Alternatives: Ambient model names or mutable cache paths cannot establish frozen execution. Local cache reuse verifies and publishes ordinary owned artifacts while leaving original inputs unchanged. A separate transfer system is unnecessary.

Sources: existing `scripts/qualify-processing.py` receipts; [HF download guide](https://huggingface.co/docs/huggingface_hub/guides/download), [Community-1 card](https://huggingface.co/pyannote/speaker-diarization-community-1), [gated models](https://huggingface.co/docs/hub/models-gated).

## 2026-10-13: Runtime, clients and evidence

Decision: Add shared workers catalog/inspect/plan/setup and processing readiness operations. Setup is durable work; successful publication uses settings compare-and-swap plus current work authority. Inspection/readiness never download weights or run inference. Preserve installed companion pins and advanced processing tools. Record a verified environment receipt alongside executable pins so later mutation is detected before elected execution. Both clients expose the same operations and normal work progress/cancel/retry.

Rationale: Existing settings ownership, process supervision and durable work provide the necessary boundaries. Doctor's backend summary alone does not prove processing readiness. Pipeline configuration/model compatibility alone is not engine or model verification. Remove unsupported GUI `auto` device and expose CPU/CUDA dependency states explicitly.

Alternatives: GUI-specific launchers would bypass runtime supervision. A first-run wizard belongs to #66 after this foundation. Requiring inference in CI contradicts the constitution; use deterministic probes in ordinary checks and opt-in measured real-engine evidence for changed combinations.

## 2026-10-13: Actual acquisition corrections

The installed-layout Windows journey identified package names accidentally used as offline wheel filenames and a current Hugging Face redirect to `us.aws.cdn.hf.co`. Retain exact decoded upstream wheel filenames, test installable suffixes and allow only the explicit official AWS/GCP CDN edges with stripped credentials. [Upstream download networking documentation](https://github.com/huggingface/hub-docs/blob/main/docs/hub/datasets-downloading.md) corroborates these hosts; no wildcard host privilege is introduced.
