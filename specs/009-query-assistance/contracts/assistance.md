# Assistance interface

Shared operations: query.assistance-show (no input); query.assistance-set (`expected_revision`, `configuration`); query.assist (`prompt`, optional `configuration`, `settings_revision`, `mode`, `context_query`). CLI: `insonic query assistance-show`, `insonic query assistance-set --input settings.json`, `insonic query assist --input question.json`. Desktop wraps the same requests.

HTTP request: contract_version 1, operation query-assistance, model, prompt, schema, capabilities, limits and optional current context. Response: contract_version 1, query (QueryInput), explanation. JSON POST only; redirects refused; credentials opaque and only header-resolved on the elected HTTPS route or explicit loopback exception. Provider errors return fixed diagnostics without bodies/credentials.

Schema includes actual Entity/EvidenceLink fields and normalized operation/filter vocabulary with dialect/parameter types; maintenance types omitted. Native proposal backend checks may remain pending for inspection but cannot auto-run until EXPLAIN succeeds. Suggest returns no execution. Explicit auto-run uses the shared bounded current query; displayed proposal is the exact execution input.
