# Research: S037

## 2026-10-17: compact desktop composition

Decision: use documented consumer CSS composition with delivered semantic/spacing tokens. Keep kit bytes and the44-unit interaction target unchanged. Use square native selection inputs inside an associated label whose complete target is at least44 by44; ordinary actions use intrinsic widths and tighter token-based gaps. Do not invent a missing density prop.

Authority: pinned brand archive `brand/kit/enforcement/distributions/shruggie-brandbuilder-3.0.1.skill`, `references/11-interface-implementation.md`33/37 permits native labeled controls, consuming-host CSS and specificity-aware overrides. Delivered IMPLEMENTATION.md137-148 leaves product composition outside bounded controls. interface-canon.json retains the interaction target. The generated adapter has no density prop and its inline field grid stretches minimum-only selection inputs. These are read-only findings from the required planning research agent; the primary inspected the pinned archive independently.

Alternatives: hand-editing generated kit bytes violates authority; reducing the target breaks the contract; waiting for an upstream API is unnecessary for this documented consumer adaptation. Record the adaptation in development docs and a local capability record if reuse beyond the desktop is later needed.

Verification: real browser measurements across six workflow groups at1200x800/720x600, device scales1/1.25/1.5/2 and200% text; distinguish browser scale from OS display settings. Click label padding, keyboard-toggle and observe focus. Add actual Wails geometry observations during existing native journeys. Representative images use the existing clean-source capture transport, plus a separate density receipt validated by the site's generic capture provenance checker.

## 2026-10-17: native Exit and lifecycle

Decision: Wails2.16.0 native menu with Windows `&File` / `E&xit`; plain File/Exit on Linux because the pinned GTK adapter uses label APIs rather than mnemonic-label APIs. Use native menu navigation, no global text-entry shortcut. Quit through the existing Wails lifecycle and preserve playback cleanup and detached-runtime ownership.

A synchronized quit controller queues an early request until startup supplies its context and suppresses repeats. Wails creates menus before asynchronous OnStartup, so an unsynchronized captured context risks a race/nil quit. Portable tests cover this ordering.

Native proof uses a dedicated mode of the actual desktop binary and owned platform menu activation, with no-workspace and existing-workspace cases. The driver holds a tiny synthetic acquisition in a separate detached runtime, exits the desktop and verifies the same session/work survives before releasing the barrier to success. Existing webview smoke intentionally hosts/cancels its own runtime and cannot prove survival. Reuse private fixture/process ownership, existing playback authority and the loopback acquisition pattern. Linux uses owned GTK menu activation on its main loop; Windows exercises its native menu keyboard message path and ordinary window close separately.

Sources: [Wails menus](https://wails.io/docs/reference/menus/), pinned dependency source under the Go module cache; [Windows menu access keys](https://github.com/MicrosoftDocs/win32/blob/docs/desktop-src/menurc/about-menus.md); [GTK menu item activation](https://docs.gtk.org/gtk3/method.MenuItem.activate.html).

## Publication and verification decisions

Owner authorization covers push/PR/two rounds and overrides the generic pre-push pause. Use hidden foreground-owned verification commands with redirected output; no local native application or installer needs to open. Run focused red/green probes before full CI. Preserve every existing native/package/recovery gate and660-second turnaround. No extension hooks are registered.
