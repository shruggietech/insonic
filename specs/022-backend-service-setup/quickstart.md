# S022 validation guide

1. Run `insonic workspace plan --input backend-plan.json --json` to inspect generated nonsecret selections and field corrections.
2. Run `insonic workspace preflight DIRECTORY --input backend-plan.json --json`; compare target files and fixtures before/after to prove no provisioning occurred.
3. Register credentials through protected input, then elect `insonic workspace provision DIRECTORY --input backend-plan.json --secret-input --json`. Inspect persisted profiles, then exercise configured adapters.
4. In desktop workspace creation/Settings, use the matching backend forms and preflight; invalid/failed input must leave existing selections intact.
5. Run `insonic query services --json` and inspect source/time, model capabilities and routes. Explicit custom endpoints use a discovery input file.
6. Elect adapter ollama, endpoint, model and route with `query assistance-set`; run an explicitly elected `query assistance-test`, then `query assist` against a deterministic fixture.
7. Use desktop Explore's matching discovery/model selection and assistance actions. Verify no silent model adoption or cloud route change.
8. Run repository checks and affected product/site suites, then inspect exact-head CI and all PR review threads before handoff.
