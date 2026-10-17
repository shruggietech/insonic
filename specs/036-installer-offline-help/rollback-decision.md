# Upgrade rollback decision

Status: expansion authorized by the owner on 2026-10-17 ("Expand S036 recovery scope"). No product recovery or source-retention behavior is implemented yet. S036 remains draft in PR #151.

## Established evidence

The repeated documentation-completion failure was a fixture desktop mismatch. MSI selected WinSta0/Default while the disposable account was granted another desktop. The corrected fixture passes12 independent completion cases and six actual offline browser renders.

A separate native failed-upgrade case on disposable Windows Server2022 build20348 returns1603 and restores old program bytes but changes Windows Installer registration from installed5 to advertised1. Both the direct API and normal msiexec.exe client reproduce it. Fresh processes and two supported sequence candidates do not fix it; the late-removal candidate is worse and is rejected. No inference about an attended owner installation or Windows client build is claimed.

Public MsiConfigureProductEx maintenance cannot recover the advertised state without the original source MSI (native1612). Source-present explicit prior-feature configuration reaches RegisterProduct but returns1602 when SecureRepair requests elevation during a silent operation. Neither public maintenance probe restores installed registration. Source retention alone is not a demonstrated fix. Windows documents cached MSI files as internal data and directs maintenance callers to public configuration APIs, so consuming or editing the internal cache/registry is not a proposed workaround.

## Scope decision

Recommended: keep PR #151 draft and address the independently reproduced rollback/source-resiliency blocker explicitly. Do not repeat the complete CI matrix until a focused correction passes the native registration and byte assertions. Do not claim restored files alone prove a restored installation.

Expanding S036 would require a separate design for source availability and trust, supported failure recovery across interactive and unattended installation, exact prior directory/client selections, source/cache lifecycle and capacity, and legacy installations without retained source. Retaining source only for new installations does not repair the legacy case. No new administrator requirement, global policy/registry permission change or unapproved external source download is proposed.

The alternative is a separately tracked blocking work slice while S036 remains unmerged. That preserves the current product requirement and keeps the three UI/help issues distinct from a new installer-recovery subsystem.

## Required proof for any recovery implementation

Exercise ordinary-user native install, failure/cancellation after old removal, registration5 and exact old bytes, original source unavailable, exact retained feature/path choices, missing/corrupt recovery data, cleanup after successful upgrade/uninstall, and the existing no-elevation/ownership/inventory gates. Keep the660-second budget. Final captures, full CI and the reserved second review round remain pending.

[Owner scope boundary](../../AGENTS.md): Ask for material scope changes; do not repeat routine authorization.

## Authorized research and implementation order (2026-10-17)

Prove a public-API recovery path before designing source retention. The next focused comparison removes only the exact owned advertised diagnostic product through MsiConfigureProductEx and reinstalls the original full MSI through MsiInstallProduct, preserving the old path and feature choices. It refuses removal when that source is absent. This tests whether SecureRepair belongs to advertised maintenance rather than fresh ordinary-user installation. It is diagnostic only and cannot qualify the product.

The upstream file-only per-user rollback report provides independent related evidence, not proof of an identical cause: https://github.com/wixtoolset/issues/issues/8574. Retain the existing gate until a supported recovery path passes. Production integration must cover direct MSI interactive and unattended invocation, transaction completion ordering, source trust/lifecycle and legacy cases; a helper that returns after an unrecovered failure is insufficient.

## Public recovery consumer result (2026-10-17)

The focused native public-removal/full-reinstall comparison now restores old registration5 and Core/CLI feature state3 with exact old hashes, under the same ordinary medium account. Both operations return0. The earlier advertised-maintenance failures do not imply that fresh reinstallation after supported removal requires elevation. Native direct-MSI completion ordering is the next independent gate; no production cache or helper is implemented merely from this result. The recovery work is tracked in #152 under the existing S036 milestone and PR #151.

## Entry-point decision after native controls (2026-10-17)

The owner-approved recovery expansion stays in S036 and is tracked by #152. Direct external coordinator controls pass in both silent and actual full UI; the same STA/pinned-thread coordinator inside the tested MSI containers still fails. The external coordinator waits for its fresh ordinary-user recovery child before returning the original1603, with old registration5, new absent and exact old hashes. It creates no second product/anchor, and account/profile/stage cleanup completes. This proves a supported recovery consumer with the original source available, not a production cache or completed S036.

An additional setup EXE is proposed as the recommended upgrade route while retaining the raw MSI and stating its recovery limits. That changes the documented entry-point contract and the current FR-010 direct-MSI recovery wording. An explicit owner choice is pending. See recovery-design.md for the concrete proposal. Do not adopt that contract or weaken the current native registration gate before the choice.
