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

## Second hosted qualification and review round 1 (2026-10-17)

Head 4f3b9646eff7f704ba0cb11479ae71a2f6c27beb, CI run 39025849613: all jobs except Windows desktop packaging pass. Windows/Linux controlled help and desktop journeys pass. The MSI initializer now proves its private Session0 station coerces FULL to NONE before any installation. The correction holds one unique inactive desktop in the existing CI caller interactive session, verifies WinSta0 visibility and that session identity, moves only the newly created ordinary token session, and restores its exact temporary station grant. It does not activate a desktop or modify Default desktop access. Native proof is still pending.

A separate Windows archive image-release failure retained an unavailable CIM diagnostic. The mandatory 31-second idle lease and ten-second image-release deadline remain intact; a bounded native exact-image process snapshot replaces the diagnostic subprocess. This is a diagnostic correction, not proof that the image-release defect is resolved.

Code review round 1 found two P2 issues: the help root could resolve outside the program root, and ordinary external anchors could navigate the offline frame. Resolution now confines the canonical help root before manifest reads and defers Windows help resolution until limited-token handoff. Explicit web references use the native dispatcher; controlled help keeps local navigation and intercepts external activation. Regression tests cover aliases, invalid references, no-workspace dispatch and unchanged frame navigation; actual native qualification now also exercises that dispatcher. Security review round 1 completed without findings. Review threads remain open until the committed fix is verified.

## Third hosted qualification and focused correction (2026-10-17)

Head a63f8a1ff5a5e119ba6ee02ffdda669d7d40bac7, run 39027118511 (tested source e99106a7204266f23c929dddfc683802a1121359): Windows/Linux core, native engine, native desktop and Linux packages pass. The Windows desktop archive also passes, including image release. The independent CLI archive reached its 30-second import wait with a still-running job and no job error; that existing deadline is retained. MSI stopped before actor creation because the profile privilege helper rejected the new one-entry SeTcbPrivilege scope. The correction permits that one scope only for the existing SYSTEM broker, uses the actual requested count, and tests exact privilege restoration and non-SYSTEM refusal. Station/profile/account cleanup completed for the failed native attempt. Installer artifact 12099320565 archive SHA256 6b8e235205cdb6e755b7bcf3a0806d2b3370ea06cee8c3c26f522caecf1ed1cb retains original evidence.

All required local checks after round-one fixes passed: root check/118 tests, 308 Python tests, full Go/vet, production desktop source, frontend type checking/103 tests, site check/15 tests/build. The additional manifest-file alias boundary also passes focused Go checks. Native transaction and completion proof is still pending; no transaction screenshot is claimed.

## Fourth hosted qualification (2026-10-17)

Head 669f3d8662a5d504ca905652be8612b26f721dfd, run 39027784480 (tested source 41820c6dd982397daabcd0d2a13666b40ec839a0): every elected job except Windows desktop/MSI passes. Windows CLI passes its unchanged import deadline; the desktop archive passes image release again. The broker verifies the existing interactive session and moves only the new ordinary token SessionId from 0 to 2, preserving reported SID/LUID/integrity/elevation/privileges. The process exits with code 1 before any worker receipt. The next correction uses the same interpreter distribution pythonw.exe, redirects streams before runpy imports and records final station restoration even on worker failure. This removes console startup dependence and retains the missing diagnostic boundary; it does not yet prove full UI. Artifact 12099197591 archive SHA256 cd6f5857e2ed4656129870aaaf1871738956cbaf6362339170d2008e128da35d retains the actual attempt.

An independent local stock Edge headless file-URL probe, with networking disabled and an owned temporary browser profile, loaded the offline 1.0.1 heading, all requested resources, images and used font faces. No file-access override was supplied. This proves local file rendering, not native installer completion dispatch.
