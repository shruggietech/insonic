# Web icons

Browser, touch, and installable-web assets. `favicon.svg` is preferred. At raster tiers, copy both ordinary `android-chrome-*` and dedicated opaque `maskable-icon-*` PNGs alongside `site.webmanifest`. The manifest declares their roles separately; do not reuse a transparent ordinary icon as maskable.

| Path | Use |
|---|---|
| `favicon.svg` | Preferred reduced-mark browser favicon |
| `favicon-full.svg` | Compatibility vector alternative using the configured square source |
| `favicon.ico` | Classic multi-size fallback |
| `apple-touch-icon.png` | Apple touch icon |
| `maskable-icon-192x192.png and maskable-icon-512x512.png` | Opaque PWA maskable artwork |
| `site.webmanifest` | Installable web metadata |
