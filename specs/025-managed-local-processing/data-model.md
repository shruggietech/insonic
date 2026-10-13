# Data model

## Worker profile and assets

Profile ID is recognition or diarization; host is windows_amd64 or linux_amd64. Exact Python release, uv version, asset URL/size/SHA-256, engine package pins and full requirements-lock digest define the managed identity. Model bundles retain normal base-model manifests.

## Setup election and durable work

Fields: role, mode (managed/reuse), explicit interpreter for reuse, optional verified offline asset/wheel directories, expected processing-tools revision, frozen profile digest and selected worker pin. Setup is pending/running/succeeded/failed/cancelled/interrupted using existing work generation/lease semantics. Attempt outputs are owned paths; only verified success with current authority and matching settings revision publishes configuration.

## Environment receipt

Exact interpreter/engine versions, supported devices/dependency facts and dependency files with calculated SHA-256. Processing tools hold a pinned receipt reference per role. Accepted work freezes it with existing tool identities; verification failure blocks that elected local operation and offers reinspection/reconfiguration, without fallback.

## Model acquisition

Existing immutable manifest and installation/acquisition IDs remain authoritative. Internal acquisition payload freezes destination profile ID/revision/configuration and optional elected local reuse directory. Original immutable HF URLs stay in manifests; signed redirects and credential values never enter durable records. Progress exposes bounded downloaded/total bytes, current role and verified count.

## Task readiness

Each of import, existing_subtitles, transcription, diarization and training reports state, usable flag, prerequisites, corrective actions, elected model/device/adapter and verification level. A configured path, checked file, compatible imported engine, acquired model and complete task are distinct facts. Training additionally requires an elected adapter and suitable dataset/profile inputs. No readiness inspection downloads or infers.
