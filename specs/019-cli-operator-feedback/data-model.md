# S019 response model

- Error: existing code/message/diagnostics plus optional allowlisted generated reason. Reasons are fixed product conditions; they never contain external text. Existing exit mappings remain.
- Startup report: kind/version/startup_id/workspace_id/state/stage/code/reason. One line at most 4096 bytes; unknown fields/identities/protocol are invalid.
- Diagnostic event: timestamp/level/event/request_id/workspace_id/startup_id/code/reason/state/count. No arbitrary message/value/path/input/body fields.
- Work observation: identity/generation/state/phase plus available result counts. No payload or owner token rendering; warning item references use known results/pages.
- Model readiness: exact model/digest/registry state and existing acquisition work state, verified-now outcome and declared file roles/publication identities. Generic artifact leases and cleanup remain authoritative.
- Dataset response: state/reference_count/readiness, where current nonempty means ready_for_preparation, current empty means empty, and stale means invalidated. No persisted dataset schema changes.
