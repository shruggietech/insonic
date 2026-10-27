# S052 validation guide

1. Run `python -m unittest discover -s tests -p '*terminal*test.py'` for transport/receipt regressions, including the actual current-host terminal probe.
2. Build the current CLI with the pinned Go toolchain and run existing `scripts/qualify.py` source CLI qualification. Its nested terminal receipt must report every required case and exact executable identity.
3. Run normal native/package qualification on each supported CI platform. Extracted-package receipts must bind the terminal campaign to the packaged CLI digest and retain existing archive/inventory/source checks.
4. Run full Python tests, `npm run check`, `npm test` and affected site checks. Converge before publication and audit final-head CI terminal metadata.

Use hidden noninteractive process launchers on Windows. A local Windows pass cannot establish Linux execution. Actual inference and installed-owner behavior remain separate.
