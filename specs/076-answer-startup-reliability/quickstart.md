# S076 validation

1. Prepare pinned Go/native tools, Node dependencies, matching docs/frontend and a disposable complete CLI package. Verify model manifest/blob identity separately before starting the qualification clock.
2. Run npm run check, npm test, Go tests/vet, focused schema/answer/catalog/runtime/CLI regressions and affected frontend/site checks and builds through hidden noninteractive launchers.
3. Use a disposable catalog with valid identity and accepted data. Observe valid cold catalog show, wait for owned runtime exit and exclusive export, inject schema 999 only while closed, then verify catalog show and catalog migrate both exit 3 with the catalog-specific reason and unchanged data/marker.
4. Start an owned hidden local Ollama service using the exact verified qwen2.5:14b manifest. Run scripts/qualify-answers.py INSTALLED_CLI NEW_WORKSPACE REPORT.json --elect-model --configuration CONFIGURATION.json with max_repairs 1 and supported provider timeout. Inspect all seven frozen scores and the corroborated Monday result. Record actual inference separately from fixture results; release owned processes/workspaces.
5. Converge against spec/plan/tasks, retain compact receipts, push/open the issue-linked PR, satisfy all review comments and exact-head CI within the two-round cap, then hand off for owner merge.
