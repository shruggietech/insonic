# S007 research

2026-10-07. Spec Kit research agents examined the installed source and pinned dependencies; no implementation edits preceded analysis.

## Static desktop frontend

Decision: React/TypeScript static bundle inside Wails, with delivered AppFrame, fields, buttons and semantic CSS. Rationale: matches technology authority and shared bridge without a production server. Alternatives: expanding raw status HTML omits maintainable screen state; changing desktop framework repeats already qualified platform work. Request generations and authority readback prevent stale async state. Wails owns titlebar; AppFrame owns scrolling/focus and remaining environment responsibilities.

## Runtime contracts

Decision: reuse media.set-origin/relocate and work operations; add shared settings CAS, current cue projection and leased scoped playback. Rationale: existing mature core behavior should be wrapped, and direct CLI file writes must not become GUI-only duplicated logic. Alternatives: raw file URLs bypass current identity and access scope; full documents in JavaScript lose integer timestamp precision; unconditional backend profile overwrite can strand workspace content. Established backend migration stays with #14 and is explicit advanced configuration lag.

## Native delivery

Decision: Windows AMD64 ZIP, Linux AMD64 tar.gz with install helper/desktop entry and macOS ARM64 app bundle/ZIP, qualified after relocation to a path with spaces. Rationale: these match actual artifact pins and executed CI runners. Alternatives: unsupported universal packages overclaim architecture evidence; model bundles add unrequested downloads and CI expense. Include exact companion support trees, hashes/licenses/source provenance/help. Never persist installed absolute paths into portable workspace state.

Primary platform references: [Wails options](https://v2.wails.io/docs/reference/options/), [dynamic assets](https://v2.wails.io/docs/next/guides/dynamic-assets/), [platform dependencies](https://v2.wails.io/docs/gettingstarted/installation/), [Windows WebView2](https://v2.wails.io/docs/guides/windows/), [FFmpeg distribution guidance](https://www.ffmpeg.org/legal.html). Source/readme provenance must match actual redistributed companion bytes. Signing, notarization and official release publication are not claimed by build qualification.
