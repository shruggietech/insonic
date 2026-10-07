# S003 validation

1. Use Go 1.27.1 and configured native compiler, run npm run check and npm test.
2. Run go test ./... and go vet ./..., then artifact/catalog race tests.
3. Run integration-tag artifact and catalog suites against pinned S3/PostgreSQL fixtures.
4. Build CLI, initialize a disposable workspace, publish a file with --request-id and inspect/verify its returned publication ID.
5. Materialize it, retain it and confirm retirement conflicts; release reference and lease, wait configured grace, retire and verify historical state.
6. Export/restore its catalog into an empty workspace identity and confirm imported publication/cache owners are expired.
