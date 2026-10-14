# S027 research

2026-10-13: Research agents dispatched by installed speckit-plan instruction. All feature-scope choices are resolved; execution evidence belongs in verification.md.

## Installer authoring

Decision: native Windows Installer APIs and Windows cabinet tools, not an externally accepted WiX tool installation. Rationale: genuine MSI standard engine tables/actions support feature selection and transactional maintenance; Python ctypes avoids removed Python msilib bindings. Alternatives: maintained WiX versions were researched through [release notes](https://docs.firegiant.com/wix/whatsnew/releasenotes/) and [tooling terms](https://docs.firegiant.com/wix/osmf/); no new commercial EULA is silently accepted.

Decision: per-user context, three nonempty client selections, mandatory shared core, no global PATH or bootstrapper. Sources: [installation context](https://learn.microsoft.com/en-us/windows/win32/msi/installation-context), [ProductVersion](https://learn.microsoft.com/en-us/windows/win32/msi/productversion), [component rules](https://learn.microsoft.com/en-us/windows/win32/msi/what-happens-if-the-component-rules-are-broken), [upgrade sequencing](https://learn.microsoft.com/en-us/windows/win32/msi/removeexistingproducts-action). Standard installer transactions and owned integrations preserve rollback and user data.

## Capture

Decision: local application capture uses actual compiled frontend and a bounded isolated real-runtime proxy, with verified headless Chromium and windowsHide. Bundled Playwright 1.62.1 is available for a prototype; repository dependencies must remain explicit/reproducible. Native installer capture only runs in disposable CI; prior owner-host preview/watchdog is retired after the foreground-window incident. No generic desktop/window-close watchdog is permitted.

Decision: capture receipts label browser/native rendering honestly. Real fixture identities, source version, browser/OS, locale, viewport and image hashes are mandatory. Only actually observed OS prompts receive screenshots; conditional prompts use primary textual guidance.

## Release and documentation

Decision: preserve four archives and add a separate installers list with exactly one Windows amd64 MSI. Assets become seven package/documentation files and nine published files including manifest/checksums. Historical contracts retain their original counts. Actual MSI payload verification must independently match exact inventory/source/help/license authority.

Decision: use a versioned screenshots asset root and carry it into offline output. Existing offline export otherwise omits PNG assets. Reverified primary guidance: [Edge SmartScreen](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-security-smartscreen), [first application prompts](https://learn.microsoft.com/windows/apps/package-and-deploy/publish-first-app), [application reputation](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/smartscreen-reputation), [UAC](https://learn.microsoft.com/en-us/windows/security/application-security/application-control/user-account-control/how-it-works), [Smart App Control](https://support.microsoft.com/en-us/windows/security/threat-malware-protection/smart-app-control-frequently-asked-questions), [S mode](https://support.microsoft.com/en-us/windows/experience/platform-variants/windows-10-and-windows-11-in-s-mode-faq), [app-source recommendations](https://support.microsoft.com/en-us/windows/apps/change-your-app-recommendation-settings-in-windows). SAC's current FAQ permits re-enable on recent updates and has no per-app override. Reputation, UAC, licensing and prerequisites remain separate.
