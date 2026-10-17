# S036 verification

## Analysis gate (2026-10-17)

Read-only analysis covered nine functional requirements, five success criteria, three stories and sixteen tasks. All requirements have implementation and native verification tasks. No critical/high conflicts or unresolved scope decisions remain. Constitution gates pass. Six interaction checklist questions remain reviewer-owned; owner autopilot authorization permits implementation. Actual full-UI transactions and installed help activation are required evidence, separate from model/dialog/DOM checks.

## Implementation and local checks (2026-10-17)

Shared manifest/path validation, Windows association dispatch with linked-token handling, controlled embedded desktop help and local-only response policy, native modeless progress, operation/result controls and independent completion elections are implemented. The actual native transaction and twelve-case completion harness runs only in the existing owned disposable ordinary-user/private-desktop actor. Its bound grows from 90 to 240 seconds for these additional real transactions and offline renderers; the job budget remains 660 seconds. No local installer/desktop UI was launched.

Local checks pass: root check and 118 Node tests; 306 Python tests; native Go tests and vet; desktop source tests; frontend type checking and 102 tests; site checks, 15 tests and build. Earlier red tests demonstrated the absent progress/election model and legacy help anchor. The canonical Windows temp path assertion now compares file identity rather than short/long spelling.

Native evidence is pending. The old installer capture source was a non-ancestral pre-squash commit; every recorded source hash matches the integrated S027 commit e79aedda4f6161df0e5aeee67e6e50ad577fd93d. The receipt is not rewritten to claim new rendering. Its images/receipt are temporarily removed while exact-head disposable CI produces replacements; replacement captures and walkthrough must be restored before final handoff.
