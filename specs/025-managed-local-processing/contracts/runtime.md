# Shared runtime contracts

All requests use existing runtime-request/runtime-response 1.0.1 envelopes and workspace/request identities.

- `workers.catalog`: no input; versioned profiles, assets, locks and builtin model choices.
- `workers.inspect`: role and optional absolute python_path; inspect an elected candidate or current tools, with calculated hashes and bounded dependency/device facts.
- `workers.plan`: setup fields without required expected_revision; read-only plan, costs/offline options and current settings revision.
- `workers.setup`: plan election plus required expected_revision; return normal durable work view. Use work show/wait/cancel/retry.
- `processing.readiness`: optional pipeline ID/revision, recognition/diarization model references and elected devices; return separate five-task readiness.
- `models.defaults`: no input; builtin immutable choices, licenses, roles/bytes and acquisition states.
- `models.prepare`: builtin choice, optional opaque credential_id, local_directory and destination_profile_id; preview is read-only unless acquire is true. Acquisition uses ordinary shared model work.

CLI mirrors operation groups with optional/required `--input FILE`, except catalog/default listing. No unattended prompts, secrets in argv or separate GUI launcher. Desktop Settings exposes equivalent fields/actions and ordinary Jobs progress; Pipelines inspection shows the same readiness.

Errors use existing typed codes with fixed corrective descriptions. Paths/configuration conflicts and inaccessible dependencies do not expose provider response bodies. Explicit device/profile changes apply to future elections only.
