# Shared setup operations

Pre-workspace: `setup location [PATH]` previews suggested/current path, existing workspace state, artifact/control/cache locations and capacity without creating the destination. `setup draft show` and `setup draft save --input FILE` read/save the private bootstrap draft. Save accepts expected_revision, path, plan, stage and status under the global request UUID. `setup provision` reuses the saved normalized plan and existing protected bootstrap channel.

After workspace selection: `setup show` reads current journey; `setup save --input FILE` accepts expected_revision, state and observation proof. `setup readiness` returns the existing task-specific processing readiness with saved pipeline election. `settings setup` aliases reopening/readback. All commands are noninteractive and support --json.

Native SetupDraft wraps the same pre-workspace package. Operate uses setup.show/setup.save/setup.readiness envelopes with current workspace ID. Journey saves do not provision backends, create pipelines, download models or import media. Those remain explicit existing operations.

Errors: invalid_request for malformed/secret/unknown fields; conflict for stale revision or changed request identity; unavailable for storage/prerequisites. Replay returns accepted revision separately from potentially newer current state.

## Exact journey shape

Save input: `{expected_revision: integer, state: {status: active|deferred|completed, current_stage: workspace|backends|processing|services|import, stages: [{id, state: pending|completed|skipped, proof: optional SHA256}], choices: {processing_mode: none|local|connected|custom, pipeline_id: optional UUID, pipeline_revision: optional positive integer}}}`. Exactly five unique stage IDs are required. No arbitrary settings, host paths, credential values or text blobs are allowed.

Show result: `{kind: first-run-setup, schema_version, workspace_id, revision, state, observations: {workspace,backends,processing,services,import}, stages: [{id,state,correction?}], paths, actions}`. Persisted state remains the input shape. Derived stages may become needs_attention. Proofs are current SHA256 configuration observations; only skipped stages may omit them. Setup readiness uses the selected pipeline or the generic existing readiness input.
