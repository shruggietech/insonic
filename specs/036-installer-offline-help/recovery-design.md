# S036 recovery architecture proposal

Status: owner approved on 2026-10-17 ("Add setup EXE; retain raw MSI (Recommended)"); production implementation pending. The earlier recovery expansion and this entry-point selection authorize the additional recommended setup EXE with retained raw-MSI limits. The native direct-coordinator controls pass in actual full UI and silent mode; the embedded variants fail. This is prototype evidence, not product recovery qualification.

## Established boundaries

- Ordinary-user raw MSI failure restores old files but leaves old registration advertised.
- Public advertised removal followed by original full-package reinstall succeeds outside the active transaction (run39047287616), restoring registration5, selected features and exact old bytes without elevation.
- Execute-sequence failure actions receive1618 while the original installation remains active.
- The tested embedded-chainer variants do not restore registration before return. Some produce internal Installer failures and are rejected. These tests do not prove every possible MSI authoring design is impossible.
- Direct quiet/full coordinator controls in run39050679170 at8af788ec36387f42c476a2137d5e6dcd8a96906e pass: both return the actual inner1603 after fresh public recovery completes0, old registration5/new absent and exact old bytes. Full UI observes real progress and FatalError; all owned account/profile/stage cleanup completes. Embedded variants still fail with the same explicit STA initialization. A completed diagnostic trace never qualifies product recovery.

## Selected architecture: external setup coordinator

The selected architecture supported by the direct controls is an additional Windows setup EXE containing the exact payload MSI and coordinating its native full/silent lifecycle. Preserve MSI product identity, native pages, actual license/choice controls, all completion elections and ordinary-user execution. Keep the MSI available, with an explicit distinction between a coordinated recovery guarantee and the raw engine behavior. Update the distribution contract with a setup EXE in addition to the raw MSI.

The coordinator waits for the native result. On failure/cancellation after destructive work, supported recovery removes only the advertised old product and reinstalls the validated original full MSI with the prior directory, features and integrations. Recovery must finish before the coordinator returns, retaining the original1603/1602 result. Incomplete recovery remains an error with an actionable retained local record. Native MSI transactions and owned program bytes remain separate assertions.

Before upgrade, establish a validated local source and prior-choice snapshot. A new install retains its full MSI in a private application-owned per-user source directory. Legacy installs may establish source from the original local package after identity/content validation. If required source is unavailable or corrupt, stop before old removal. Do not consume or edit Windows Installer internal cache/registry as a substitute. Do not fetch an unselected external source or require elevation. Clean obsolete owned recovery data after successful upgrade/removal and retain recovery material after incomplete recovery.

The setup EXE must declare asInvoker explicitly, avoid installer-detection elevation, and use hidden noninteractive console child creation. Do not execute mutable per-user cached sources with administrator privileges; use the verified same-user limited-token handoff before processing user-owned recovery data. Preserve the existing unconfigured signing truth and release version. Reuse approved icon/brand bytes. A new setup asset requires receipt, release-assembly and documentation updates, not a new runtime framework.

## Owner entry-point decision (2026-10-17)

The owner explicitly selected the additional recommended setup EXE and retention of raw MSI with documented recovery limits. FR-010, the plan and tasks now describe that approved contract. The raw-engine advertised-registration defect remains documented; the setup recovery assertion still requires installed registration and exact prior choices/bytes.

## Implementation and proof after selection

1. Validate source extraction/identity, canonical private paths, digest and length, prior choices, locking and atomic state before any old-product removal. Cover missing/corrupt/reparse/foreign data and legacy source loss.
2. Integrate the selected coordinator with native full/silent install, cancellation and failure, keeping actual engine outcomes and completion choices. Keep original source unavailable during recovery tests.
3. Prove ordinary-user SID/integrity, old registration5, exact prior bytes/path/features/integrations, source lifecycle, cleanup and no detached critical work. Observe real native progress and actual Cancel controls.
4. Refresh all required native matrix/captures, run exact-head CI within660 seconds, use the reserved second code/security review round, resolve findings and stop for owner review/merge.
