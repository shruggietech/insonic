# insonic documentation site

Markdown in `docs/v<version>/` is the documentation source. `docs/versions.json` defines the published versions, page order, and latest version. v0.0.0 is a system specification, not an application release.

Use Node.js 22.12 or newer. From the repository root:

```sh
npm --prefix site ci
npm --prefix site run build
```

The build copies the committed upstream brand kit, bundles Mermaid locally, renders Markdown into Next.js static pages, and checks exported links. `site/out/` is the public website, including the product landing page at the root and versioned documentation under `/docs/`. It includes no CDN resources or production server requirement. `npm --prefix site run preview` serves a local preview on `127.0.0.1:4173`. `npm --prefix site run dev` starts a development server.

The build also produces `site/offline/` for desktop help. It includes the versioned documentation, JSON schemas, public references and local assets, and excludes the public landing page. Open `site/offline/index.html` directly to enter the documentation. `offline-manifest.json` identifies its version and entrypoint. The application release process must build and bundle the help version matching its release; the native packaging implementation supplies that integration.

Use fenced `mermaid` diagrams with `flowchart TB` or `graph TD` for top-down flow. Supply `accTitle` and `accDescr` in diagrams when a prose equivalent is not already nearby. Diagrams retain their readable source when JavaScript is disabled or rendering fails. Code blocks use local syntax highlighting; tables, task lists, strikethrough, and automatic links are supported. Raw HTML is escaped.

Next.js produces static HTML and CSS. A postbuild pass removes unused Next.js hydration scripts and rewrites links for the portable export. A small blocking theme initializer resolves the saved light/dark preference or the operating-system default before body rendering. A small shared runtime handles accessible archive-preview tabs and the sun/moon toggle. Every documentation page loads a separate local Mermaid runtime for diagrams and theme recoloring; the landing page loads only the shared controls. Ordinary links and server-rendered content keep the public site and offline help portable. Storage denial falls back to the system theme and still permits changing the current page.

The landing hero and metadata read the approved message roles directly from the exact upstream `brand.json`. Product composition uses delivered color, type, spacing and radius tokens. The download CTA leads to platform availability on the landing page; package links are added when actual release artifacts exist.

Syntax highlighting is rendered at build time. JSON keys, values and literals use distinct semantic colors in both themes; CLI examples preserve shell syntax and highlight the executable, flags and placeholders. Purely sequential procedures use numbered steps. Retained diagrams show branches, relationships, dependencies, feedback or retry behavior.

Keep the version manifest in lock-step with release preparation. Add the release's Markdown snapshot and change `latest` together; do not relabel an older snapshot as a new version. Publish the checked export for the latest repository release and retain earlier versions for users of earlier installs.

The contracts page derives its reference tables and examples from `schemas/v<version>/`. Root checks compile all registered schemas and validate their examples through the master schema. The changelog page reads the root changelog while a version is a specification. Before marking a version as released, copy its tagged root `CHANGELOG.md` to `docs/v<version>/references/CHANGELOG.md`. The build requires that snapshot and renders it for that release.

Repository sources directly linked from the version's Markdown pages are rendered locally and included as plain-text assets. Links inside those references may point to additional online repository sources. Before publishing a release version, freeze each bundled source at `docs/v<version>/references/<repository-path>` so archived documentation retains the matching source. A specification version can read the current public repository files directly. Each rendered reference also links to its canonical repository source for provenance.
