# insonic brand integration

The [ShruggieTech brand repository](https://github.com/shruggietech/shruggie-brand) owns the insonic identity. The [brand essentials](https://brand.shruggie.tech/insonic/guidelines/brand-essentials/) describe approved use. Files in `kit/` are exact upstream release bytes; edit upstream, then import a newer kit here. `lock.json` records the release, source revision, archive and manifest SHA-256 digests, and bundled legal-file digests.

Run these commands from the repository root:

```console
node scripts/sync-brand-kit.mjs --check
node scripts/sync-brand-kit.mjs --update
node scripts/sync-brand-kit.mjs --verify
```

`--check` validates the installed kit, finds the newest stable insonic kit in official upstream releases, and verifies its archive checksum, manifest inventory, and package identity without writing files. `--update` stages and verifies a newer kit before replacing the installed copy. It restores the previous kit if replacement fails, rejects downgrades, and refuses silently replaced versions. `--verify` checks installed files offline. Add `--json` for machine-readable results. A newer BrandBuilder compiler release can update the kit even when the identity version stays the same.

Review and commit the resulting brand changes with their consuming application or documentation changes. Nothing is fetched during ordinary builds. Release checksum verification establishes consistency with the official download; it is not a claim of a cryptographic publisher signature.

Use `kit/logos/named/insonic-logo-lockup-wide-full-color-clear-for-light.svg` on a light surface and its `clear-for-dark.svg` counterpart on a dark surface. Fonts, tokens, component recipes, platform icons, conformance material, and upstream licenses ship with the complete kit. Product and site builds copy required assets from this directory instead of maintaining independent artwork.
