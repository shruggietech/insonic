# Windows icons

Embed `classic/app.ico` in Win32 executables and use the `msix` directory only for packaged Windows applications. The ICO and target-size assets have taskbar roles; scale, tile, and Store assets have separate plate rules. Confirm which resource the actual EXE or MSIX embeds and inspect 16, 24, 32, and 48-pixel light/dark taskbar appearances.

| Path | Use |
|---|---|
| `classic/app.ico` | Classic Win32 executable, shortcut, and taskbar icon |
| `msix/Assets` | MSIX scale/tile, target-size/unplated taskbar, and Store assets |
| `msix/ApplicationVisualElements.fragment.xml` | Merge into Applications/Application |
| `msix/PackageProperties.fragment.xml` | Merge into Package for the Store logo |
