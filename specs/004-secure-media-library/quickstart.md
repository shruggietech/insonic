# S004 validation guide

1. Build the CLI and initialize a private fixture workspace.
2. Configure a native credential using bounded stdin JSON; inspect status, restart, replace explicitly and delete. Repeat with selected encrypted vault and wrong-passphrase/tamper fixtures.
3. Register/download a small exact multi-file model fixture, interrupt/retry and inspect verified publication receipts after restart.
4. Import audio/video copies and references individually and through CSV/JSON manifests, including supplied subtitle inputs.
5. Inspect current raw metadata, provenance, measured timing and dates. Move/delete a reference, report availability, relocate only matching bytes.
6. Exercise date-only, explicit zone/offset, conflicting observations, fold/gap and unknown duration. Media admission continues with unresolved dates.
7. Refresh computed metadata and verify current references plus removal of superseded managed reports. Restart at each publication/commit/cleanup boundary.
8. Run npm integrity/tests, Go acceptance/vet/race, Python qualifications, both backend fixtures and static/offline docs. Hosted checks cover native operating systems before owner merge.

Exact command syntax is maintained in the runtime contract and public development/import documentation once implemented.
