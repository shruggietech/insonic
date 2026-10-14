# S027 validation quickstart

1. Run root npm check/test, meaningful MSI model/receipt and launcher tests, Go tests/vet, configured Python suites, frontend check/test/build and site public/offline checks. All local tooling is hidden and noninteractive.
2. Generate application screenshots with the current CLI and compiled frontend, isolated fixture and headless browser. Inspect actual PNGs and their exact-source receipts.
3. In disposable Windows CI only, assemble the MSI from exact qualified desktop payload, qualify all nonempty client combinations and maintenance/rollback/removal, and capture authored native dialogs with owned-process cleanup. Never run this native UI path on the owner host.
4. Verify candidate collection rejects missing/duplicate/stale MSI while preserving all four archive gates. Check online/offline PNG and document parity.
5. Publish the authorized official PR, address at most two review rounds, verify final exact-head CI and present the concrete proposed launcher wording for owner final-review signoff before shipping.
