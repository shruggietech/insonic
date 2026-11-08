# S076 research

## Answer schema and repair

- Decision: Reuse packaged envelope/claim/citation/reference/interval definitions, expanding local refs into a self-contained schema; native local Ollama supports a schema object in format ([official documentation](https://docs.ollama.com/capabilities/structured-outputs)).
- Rationale: The field-name descriptor lacks types. Existing sanitized schema violations already retain pointers/types without submitted values. Go decoding alone permits null string fields.
- Alternatives: A prose-only template leaves ambiguity; weakening reference validation accepts invalid authority; forcing structured format on all adapters changes unrelated consumers. Reject all three.
- Evidence: schemas/registry.go, schemas/input_error.go, internal/app/answer_synthesis.go, internal/assistance/ollama.go, internal/answers/synthesis.go.

## Catalog startup

- Decision: Classify at checkVersion/migrate origins and preserve the specific typed reason through runtime startup.
- Rationale: Serve replaces every app error; cmd/insonic/startup.go also replaces reasonless errors. Other incompatible_version errors describe profile/backend versions, so blanket schema tagging is misleading.
- Alternatives: Preserve only code (still lost downstream); forward raw errors (leaks unrelated details); catalog downgrade (changes data). Reject all three.
- Evidence: internal/catalog/migration.go, internal/catalog/store.go, internal/runtime/runtime.go, cmd/insonic/startup.go.

## Readiness and delivery

- Decision: Verify the existing qwen2.5:14b manifest and every blob, reuse maintained hidden ProcessTree service/campaign patterns and package qualification. Model presence is observed but integrity remains a preparation task.
- Rationale: Completed invalid-output exchanges, not unrelated qwen3 timeouts, are the issue evidence. Product clocks remain unchanged.
- Alternatives: Mock-only closure or substitute-model results cannot prove the reported reproduction fixed.

Two focused research agents inspected the existing sources under speckit-plan Phase 0. No unknown remains that requires an architecture or owner decision.
