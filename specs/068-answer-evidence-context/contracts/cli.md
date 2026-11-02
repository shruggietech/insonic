# S068 shared CLI/runtime surface

- answers prepare --input request.json: prepare bounded evidence/context under the selected workspace and explicit routing identity. Return package/publication identity, evidence/context, budget usage, limitations and cache observation.
- answers show PACKAGE_ID: load the verified artifact and report current freshness plus immutable original evidence/context.
- answers export PACKAGE_ID: return the same complete versioned inspectable document for redirection/export, with current freshness observation.
- answers validate --input envelope.json: validate a future answer envelope against its retained package and current source authority; reject invented/stale/foreign citations and mismatched quotes/timing.

All commands support normal concise output and full --json. Preparation can report incomplete or insufficient evidence; cancellation is non-success and never promotes partial work to a completed answer. These operations do not invoke synthesis or silently publish/rebuild the graph. No provider endpoint/credential or original local source location is exported. Desktop shared runtime dispatch accepts the same contracts; dedicated GUI controls remain a documented follow-on.
