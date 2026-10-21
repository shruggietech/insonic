# Local processing setup

Local recognition and diarization are optional. The CLI and desktop share the same worker setup, model acquisition and readiness operations. Ordinary imports and supplied subtitles do not require these engines. Settings provides local setup controls; the first-run wizard is a separate capability.

## Prepare an environment

Use the complete installed package, including its verified Cueson, FFmpeg and processing worker companions. In desktop Settings, select a role, preview its environment, then prepare it. The preview names its download size, temporary-space budget and destination. Setup runs as durable work visible in Jobs.

The managed CPU profiles use CPython 3.12.15 and uv 0.13.0, with complete platform-specific wheel hashes. Recognition uses faster-whisper 1.2.1 and CTranslate2 4.8.2. Diarization uses pyannote.audio 4.0.7 and CPU PyTorch 2.14.0, TorchAudio 2.11.0 and TorchCodec 0.17.0. Linux x86-64 requires glibc 2.28 or newer. These optional downloads are separate from package installation. They never install into global Python, modify PATH or replace an accepted environment in place.

| Role | Windows download | Linux download | Minimum installation space |
| --- | ---: | ---: | ---: |
| Recognition | 122,501,331 bytes | 179,432,818 bytes | 2 GiB |
| Diarization | 290,320,124 bytes | 392,060,735 bytes | 4 GiB |

The catalog contains exact runtime archives, wheels, source URLs and hashes. Runtime and installer archives are copied into the private owned attempt while hashing, and extraction uses only those verified snapshots. All required wheel bytes are verified before an offline, hash-locked installation. An unchanged accepted managed environment is verified and reused.

For the CLI:

1. Inspect the available profiles with `insonic --workspace LIBRARY workers catalog --json`.
2. Save `environment-plan.json` with the following input.
3. Run `insonic --workspace LIBRARY workers plan --input environment-plan.json --json`. Check the returned prerequisite, destination, space and effect fields.
4. Copy the returned `result.election` object into `environment-setup.json`. It includes the automatically computed `profile_digest` and observed `expected_revision`.
5. Run `insonic --workspace LIBRARY workers setup --input environment-setup.json --json`.
6. Use its returned `result.work_id` with `insonic --workspace LIBRARY work wait WORK_UUID --timeout-ms 30000 --json`. Repeat the wait if still running. Inspect or cancel through `work show` or `work cancel`; retry failed/interrupted work through `work retry`.
7. Repeat the plan and setup with `diarization` if needed. A settings conflict requires a fresh plan; replay with the original request ID returns the original accepted work.

```json
{"role":"recognition","mode":"managed"}
```

Environments and their receipts live in the chosen library control directory. Cancellation removes the fresh failed preparation attempt; verified partial acquisition bytes remain available for retry. Successful setup publishes only the selected role after verification and a compare-and-swap settings check. Already accepted environments are retained for frozen work. Processing checks their interpreter, runtime, standard-library and dependency bytes before engine invocation.

## Reuse or prepare offline

Select an existing absolute interpreter path with `mode: "reuse"` and `python_path`. Inspection imports only the selected libraries, reports their versions and devices, and loads no model. Setup records and verifies their byte identities without installing into that environment. Its receipt/probe space budget is 32 MiB, rather than the managed profile installation minimum. Changes to these dependencies require a new explicit setup election.

For fully offline managed setup, add `offline_assets` with an absolute directory containing every declared archive and wheel under its exact catalog filename. Missing or damaged files fail locally. The installer uses only these files; it does not resolve alternate versions or fetch missing dependencies. Offline model reuse is separate.

An offline plan reports `download_bytes: 0`. `profile_download_bytes` retains the nominal complete profile size. `local_assets` reports required and verified expected byte totals, plus `missing_bytes` for missing or unusable declared files. It checks only the elected runtime, installer and wheels. Each failure identifies a logical filename, fixed role and verification reason with corrective advice; unrelated files do not contribute to these totals. An already verified accepted environment needs no new asset inspection or transfer.

Worker and model file errors distinguish missing, unreadable, nonregular, wrong-size and wrong-hash assets. The optional `error.asset` contains only a generated role and bounded slash-relative logical name. The same reason and correction survive JSON responses, human output and durable work inspection. These errors exclude owner paths, resolved credentials and raw tool output.

Existing explicit processing-tools configurations remain supported under Advanced. Accepted work freezes its interpreter, worker and environment receipt. Package-derived companion defaults are resolved from the current complete installation for future work, preserving relocation; accepted work keeps its original bytes.

## Acquire exact model bundles

The product provides Whisper Tiny (MIT, 78,203,619 bytes) and Community-1 (CC BY 4.0, 32,821,421 bytes). Each choice has an immutable upstream revision and a complete named-file manifest. Inspect `insonic --workspace LIBRARY models defaults --json` for identities, license, model card and current state.

1. Save `model-preview.json` using the input below, selecting `whisper-tiny` or `community-1`.
2. Run `insonic --workspace LIBRARY models prepare --input model-preview.json --json`. Inspect verified reuse, download bytes, scratch space and the elected storage profile. Filesystem capacity is observed when available. Colocated scratch and artifact storage use a combined peak budget, including download and staging copies; unknown volume relationships are conservatively treated as shared. S3 capacity remains unknown.
3. Add `"acquire": true` to that input and run the same command. Follow the returned `acquisition_work_id` through normal work commands or Jobs.
4. Confirm complete availability using `models inspect MODEL_UUID --json` and `models verify MODEL_UUID --json` before selecting it in a pipeline.

```json
{"choice":"whisper-tiny"}
```

An absolute `local_directory` reuses a complete existing snapshot after checking every declared role, size and SHA-256. It is published through the normal artifact service; source files remain unchanged. Omit `destination_profile_id` to elect the configured storage, or supply a saved storage profile UUID. Filesystem and S3 destinations use the same acquisition contract. A shared acquisition preserves its first destination and local/download route, and subsequent previews expose that frozen route.

Community-1 downloads require the upstream model access terms and a protected Hugging Face read token. Desktop Settings can register a token through native protected input and generate its opaque credential reference. CLI users register it through the documented [credential commands](secrets.md), then supply only `credential_id` to model preparation. Replace the same protected secret to rotate its token. Omitting the reference when preparing an existing choice retains its previously elected reference. A new gated download without the required protected reference is rejected before enqueueing; previews and local verified reuse need no download token. A supplied reference does not prove upstream authentication or model access. Credential values never enter manifests, work payloads or command arguments.

Preparation previews distinguish `credential_state`: `missing`, `locked`, `rejected`, `unavailable`, locally resolvable `available`, or `not_required`. An indexed reference alone does not establish readiness. The preview resolves required protected references locally and immediately clears the returned bytes; it sends no authentication request. `access_ready` describes this local prerequisite, while `upstream_authorization: "not_checked"` keeps upstream access separate. Complete verified installed or local reuse requires no download credential.

A failed local credential-status check reports `unavailable` without asserting that the reference is missing. Acquisition without its required locally resolvable credential returns `unavailable` with the generated reason `model_access_required`, before work is enqueued.

Pinned Hugging Face resolve requests may follow a bounded set of official model CDN redirects. Cross-origin requests strip credentials and cookies; signed CDN URLs are transient. All model bytes still require declared sizes and hashes. Custom sources retain their original same-origin transport policy.

## Inspect task readiness and select a pipeline

`insonic --workspace LIBRARY processing readiness --json` reports five independent tasks: import, existing subtitles, transcription, diarization and training. Each includes actual prerequisites, observed availability, identities and corrective actions. `doctor` and `pipelines inspect` expose the same facts. Add `--input FILE` to inspect a specific `pipeline_id`, `pipeline_revision` and optional `dataset_id`.

Readiness verifies tools, environment imports, model bytes, selected storage and the requested device. It establishes prerequisites, rather than successful acoustic inference. Local custom trainer readiness checks the elected executable and support-file hashes without invoking the trainer. `file_verified` proves those prerequisites, without claiming protocol compatibility or model quality. Hosted routes remain `configured_not_probed`; a task with only unprobed prerequisites reports `unknown`, while an observed missing prerequisite reports `needs_attention`. Training additionally requires an elected current nonempty dataset. Pipeline usability combines the elected tasks, so a training-only definition does not require unelected recognition or diarization environments. The pyannote-profile adapter enrolls speaker embeddings; it does not retrain Community-1.

Desktop Settings can save a Local CPU pipeline with the exact builtin model IDs. Select it in Library when processing media, or inspect/edit it in Pipelines. Pipeline saving does not download weights or begin inference. Existing CLI [pipeline definitions](pipelines.md) provide the same selection.

CPU and CUDA are explicit device choices. Managed profiles provide CPU dependencies. An existing CUDA environment must pass the selected engine/library probe; recognition also checks required cuBLAS 12 and cuDNN 9 libraries. Hardware count alone never establishes CUDA readiness. A failing CUDA job retains its original device, with a correction to verify compatible dependencies or explicitly elect CPU for future work. There is no automatic device fallback.

Pyannote receives an FFmpeg-decoded waveform tensor through the packaged worker. This path does not require TorchCodec shared-FFmpeg decoding. Recognition/diarization runtime success, CPU qualification, GPU execution and acoustic accuracy remain separate evidence.


## Portable restore

Worker environments are host resources. A portable backup excludes their preparation paths and local model snapshot routes while retaining completed history and verified model artifacts. Prepare or reuse compatible destination environments before resuming restored processing; their verified receipts bind alongside destination interpreter/companion tools, while models, stage settings and devices remain frozen. Missing local acquisition inputs require an explicit new model preparation election. That recovery creates one shared successor work record and leaves the originating history intact.
