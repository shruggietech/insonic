# S027 verification

## Release and documentation integration

2026-10-13: Release candidate keeps exactly four archive receipts and adds one separately typed Windows selectable-client MSI. Candidate/retrieval verifies complete MSI hash, source/version/format, exact independently qualified desktop payload receipt, corresponding-source/signing disposition and all eleven actual lifecycle/capture qualification results. Native MSI verifier independently validates database/cabinet bytes. Historical 1.0.0 retains its six-package/ten-asset record. Current published asset count is nine, including manifest/checksums.

Focused verification: 44 Python release tests passed, including missing/duplicate/wrong-format/source/qualification and recomputed-hash inventory rejection; 12 Node site/release-version tests passed, including byte-identical offline PNG/receipt copying and exclusion of unrelated sentinel files; 25 schemas, 22 contracts and 30 examples passed. Actual public/offline application capture export passed with four current-source images. Native installer figures await disposable CI.

Exact launcher wording remains proposed for final owner shipping signoff: Start menu folder insonic; GUI shortcut insonic; CLI shortcut insonic CLI; desktop shortcut insonic; completion checkbox Launch insonic for desktop and Open insonic CLI for CLI-only. The guide uses those authored strings. Microsoft Edge/SmartScreen/UAC/SAC/S mode/app-source primary guidance was rechecked on 2026-10-13; unobserved OS dialogs are described in text and never represented as captures.

## Integration checks

2026-10-13: Configured integrity checks passed over 1178 text files, 25 schemas, 22 contracts and 30 examples. Root JavaScript tests passed (93), complete Go test/vet passed, Python tests passed (188), desktop frontend typecheck/tests/build passed (97 tests), and both 81-page public/offline exports passed. Real headless Chromium rendered 16 diagrams in each export in light/dark modes. Later cancellation/context and launcher token changes receive focused checks before publication and actual disposable CI qualification.

Application capture executes the compiled frontend with the real shared desktop bridge and CLI in isolated fixture state. Four screenshots include location, backends, processing and reopened library setup, production fonts and complete controls. It does not perform model inference. Installer UI and installed lifecycle never execute locally; guarded CI-only paths record actual observations. The retired owner-host preview/watchdog was deleted; it is excluded from repository deliverables.

## Delivery state

Official PR publication is authorized. Exact-head native CI, native figure integration, external review resolution and final clean convergence remain pending. No merge, tag, release or deployment is performed. Proposed labels require owner final-review signoff before shipping.
