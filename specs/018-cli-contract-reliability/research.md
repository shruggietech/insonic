# S018 research

2026-10-11: Three read-only research agents inspected command inventories, response consumers and date/work persistence under speckit-plan.

- Decision: shared command metadata and pure token/arity validation. Rationale: current global scanner consumes literal/value tokens; 60 named flags and 42 domain flags need one grammar. Alternative: global-only patch leaves other parsers inconsistent.
- Decision: success stdout/error stderr, one envelope and retained result+error. Rationale: preserve established stream/exit contract while fixing route drift; current oneOf schema cannot validate work wait. Alternative: all responses stdout introduces a broader stream break.
- Decision: UseNumber and typed consumers. Rationale: dynamic response decoder loses integers; stored/IPC bytes are already exact. Alternative: stringify all numbers changes schema types.
- Decision: distinguish implicit captured zone from elected policy. Rationale: PrepareImport substitutes host zone before ResolveDate can honor offset; replay still needs captured rules. Alternative: discard local capture changes unqualified/retried dates.
- Decision: keep Work fields, expand generated error code validation. Rationale: whole Work participates in journal proof; extra required fields could invalidate historical receipt hashes. Alternative: generic operation_failed loses the underlying cause.
- Decision: explicit 1.0.0 document-read compatibility after 1.0.1 preparation. Rationale: release preparation preserves immutable schemas but exact-version runtime checks otherwise reject existing data. IPC remains exact.
