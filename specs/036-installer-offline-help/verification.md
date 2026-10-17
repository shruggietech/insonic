# S036 verification

## Analysis gate (2026-10-17)

Read-only analysis covered nine functional requirements, five success criteria, three stories and sixteen tasks. All requirements have implementation and native verification tasks. No critical/high conflicts or unresolved scope decisions remain. Constitution gates pass. Six interaction checklist questions remain reviewer-owned; owner autopilot authorization permits implementation. Actual full-UI transactions and installed help activation are required evidence, separate from model/dialog/DOM checks.

## Implementation and local checks (2026-10-17)

Shared manifest/path validation, Windows association dispatch with linked-token handling, controlled embedded desktop help and local-only response policy, native modeless progress, operation/result controls and independent completion elections are implemented. The actual native transaction and twelve-case completion harness runs only in the existing owned disposable ordinary-user/private-desktop actor. Its bound grows from 90 to 240 seconds for these additional real transactions and offline renderers; the job budget remains 660 seconds. No local installer/desktop UI was launched.

Local checks pass: root check and 118 Node tests; 306 Python tests; native Go tests and vet; desktop source tests; frontend type checking and 102 tests; site checks, 15 tests and build. Earlier red tests demonstrated the absent progress/election model and legacy help anchor. The canonical Windows temp path assertion now compares file identity rather than short/long spelling.

Native evidence is pending. The old installer capture source was a non-ancestral pre-squash commit; every recorded source hash matches the integrated S027 commit e79aedda4f6161df0e5aeee67e6e50ad577fd93d. The receipt is not rewritten to claim new rendering. Its images/receipt are temporarily removed while exact-head disposable CI produces replacements; replacement captures and walkthrough must be restored before final handoff.

## First hosted qualification (2026-10-17)

PR #151 head b06997c6d8301ae499b551955394205b373ebd53, CI run 39025036905 (tested merge source 3afa8b4ded04f86fbeee35423b8b22c7af606799): all elected jobs except Windows desktop/MSI packaging pass. Native Windows/Linux help journeys pass, as do Linux CLI/desktop packages and Windows CLI. The Linux help-webview receipt explicitly records version 1.0.1, workspace not selected and local-assets-only network policy.

The new MSI actor failed before any authored dialog: native log reports effective UILevel 2 and the unaccepted-license rejection. This is a harness initialization failure, not interactive transaction proof. Observer registration now precedes selecting/asserting full UI, with configured levels retained in receipts. Nested actor/task/wrapper deadlines align at 230/240/260 seconds. Actual transaction screenshot export remains contingent on completed native proof.

Seven corrected native restricted-session dialog captures were restored from artifact 12098002531 (archive SHA256 8f44bb2deb62d805a9165c8c3f500838b8c997bc0819a46d0458998e85e0031f). Finish visibly has separate checked client launch, unchecked documentation and one Finish action. Their original source identity is retained; they do not claim transactions.
