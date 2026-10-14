# S027 data model

- Installation: stable UpgradeCode, release-specific ProductCode, numeric ProductVersion, x64 platform, per-user directory, shared mandatory core, Desktop/CLI feature states and optional shortcut choices. At least one client is selected.
- Installer receipt: kind windows-installer-receipt, format msi, schema/product/documentation version 1.0.1, 40-hex source revision, Windows amd64 identity, MSI/hash/size, exact payload inventory, help/license/source closure and qualification evidence. No secrets or user paths.
- Capture receipt: exact source/version, fixture, renderer/browser/native OS identity, locale/viewport and named PNG hashes. Missing landmarks or stale source fails capture.
- Wording proposal: exact label set plus proposed/approved status and approval evidence. Approval cannot be inferred.
- Maintenance: accepted installation -> validated change -> transaction -> accepted replacement or rollback; uninstall removes owned components/integrations only. Active work is never killed by default.
