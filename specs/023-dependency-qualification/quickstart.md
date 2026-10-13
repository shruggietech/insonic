# S023 validation guide

1. Use the configured Go 1.27.1, Node >=22.12 and Python tools. On Windows use verified hidden noninteractive launches.
2. Inspect research.md and compare all seven captured elections with the owned declarations/locks.
3. Run npm run check and npm test, including negative artifact-transport policy fixtures.
4. Run go test ./... and go vet ./..., including pipe/file terminal detection and existing credential/runtime fixtures.
5. Run python -m unittest discover -s tests -p '*_test.py'; retain source/cache/candidate negative tests.
6. Install the elected site lock with npm --prefix site ci --ignore-scripts and build/check public plus offline exports. Run frontend check/test and existing release-target/Pages retention tests.
7. Check frozen docs/schemas v1.0.0 have zero changes. Read-only Pages preparation may validate retained paths but never publishes.
8. Publish the official PR with the tracking issue and all seven original updates; let required Windows/Linux native/backend/package/candidate qualification complete on the exact head.
9. Address and resolve every review thread in at most two rounds, verify all final-head checks and stop for owner review/merge. Originals remain open until that merge.
