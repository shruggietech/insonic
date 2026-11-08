# S076 data model

No catalog migration or new retained authority is introduced.

- Answer generation schema: self-contained JSON Schema, exact current version/package, complete/partial/insufficient states, at most 128 claims, kind required, references as structured objects, limitations as strings, optional uncertainty remains a string when present.
- Repair detail: optional locally generated reason plus InputViolation (pointer, expected type, constraint and numeric bound). At most one detail from strict raw schema validation; no provider values. Existing repair_reasons remains at most two strings of at most 128 characters.
- Startup error: incompatible_version/catalog_schema_incompatible; generated recovery text, unchanged StartupReport correlation/state/stage fields. Catalog marker and accepted records remain unchanged on refusal.
- Evidence: source/executable/model/corpus/configuration identity and complete scored results; no credentials, private owner corpus or provider endpoint values in committed receipts.
