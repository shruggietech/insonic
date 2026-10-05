# Documentation

Start with the [v0.0.0 system specification](v0.0.0/index.md). It defines the product, system contracts and delivery outcomes.

Markdown under `docs/<version>/` is the documentation authority. `versions.json` defines the navigation and default version. The static Next.js renderer lives in `site/`; `site/out/` is the public website and `site/offline/` is the documentation-only bundle for the GUI; see [the publication contract](v0.0.0/releases.md) for synchronized software, documentation and JSON Schema versions, and offline packaging. The site also renders the [master changelog](v0.0.0/changelog.md) and [JSON contracts](v0.0.0/contracts.md).
