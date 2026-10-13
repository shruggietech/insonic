# S021 distribution contracts

Current host/package builders accept Windows/Linux x86-64 only. Future candidate/release contracts require exactly four variants, qualification and complete exact-revision inventories. Historical 1.0.0 validation remains version-specific. Unsupported hosts/targets return explicit diagnostics before side effects.

Windows ZIP helper accepts one absent destination, validates containment/aliases before creation, copies the complete package and describes actual included clients/dependencies. Failure retains the cause, returns failure and removes only a verified owned new partial tree. Existing/substituted targets remain untouched. No MSI, PATH, shortcut or uninstaller registration is implied.
