# Upgrade rollback decision

Status: proposed, owner decision pending. No new product recovery or source-retention behavior is implemented. S036 remains draft in PR #151.

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
