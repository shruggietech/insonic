# Research: Windows desktop startup and Exit reliability

## 2026-10-19: initial evidence and decisions

Decision: preserve process.Capture before changing lifecycle behavior. Rationale: runExitGUIChild forwards only stderr, while pinned go-webview2 formats the HRESULT on stdout and stack frames on stderr. Rejected alternatives: Python outer-wrapper-only changes, generic timeout increases and blind full-CI reruns.

Decision: use bounded disposable diagnosis with exact source/package identity. Existing package/pipeline helpers verify the incident ZIP and three-asset authority. An old-byte direct child exposes the missing stream but differs from the old Go parent; report that difference and never substitute it for production qualification.

Confirmed candidates: default profile is APPDATA plus executable basename because options do not set WebviewUserDataPath. Sequential children reuse it. Windows Capture waits for root exit and closes a KILL_ON_JOB_CLOSE job without explicit browser-release observation. Neither demonstrates this incident's cause. Shared profiles are supported when environment options agree.

Primary references: [WebView2 profile lifecycle](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/user-data-folder), [controller startup](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environment). Inspect the actual error before choosing a correction.

Decision: preserve exact-run failure gates and add evidence. wait-ci-artifact already rejects unsuccessful producers; retain its run/attempt/revision and failed scope/job before aborting instead of manufacturing a completed receipt.

Two read-only planning agents supplied matching references in cmd/insonic-desktop/exit_qualification.go, internal/process/capture.go, internal/process/tree_windows.go, scripts/package-desktop.py and scripts/wait-ci-artifact.mjs. No native GUI/installer ran locally.

## 2026-10-19: controller failure taxonomy

Primary Microsoft controller documentation distinguishes invalid-state/shared-environment mismatch from parent-window destruction and rare profile-permission errors. Wails issue4559 reports8007139f with incompatible browser flags, while issue3174 reports8000ffff plus a class-unregister message. These are separate upstream observations, not proof of this incident. Error1412 following controller failure cannot identify the original HRESULT.

References: https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environment3 , https://github.com/wailsapp/wails/issues/4559 , https://github.com/wailsapp/wails/issues/3174 .
