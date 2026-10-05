# Brand conformance evidence

**Date**: 2026-10-04\
**Kit**: insonic 1.0.0 / BrandBuilder 3.0.1\
**State**: Exact upstream bytes retained; one semantic proof failure remains

The official release archive was checked against upstream SHA256SUMS and its GitHub asset digest, extracted with path/type/size validation, and verified against the complete manifest. All 574 integrated files match the receipt; the upstream manifest inventories 570 governed entries. Git attributes preserve exact vendored bytes rather than normalizing line endings.

The pinned BrandBuilder recovery archive was verified and extracted into an ignored temporary verifier environment. The verifier used a separate copy of the exact kit; it did not mutate `brand/kit/`.

| Check | Result |
| --- | --- |
| Offline kit byte integrity | Pass |
| Pinned glyph validator | 15 checks, 0 warnings, 0 failures |
| Pinned semantic verifier | 36 checks: 35 pass, 1 fail, 0 skips |
| Consuming site ESLint | Pass |
| Consuming site stylelint | Pass |

The remaining semantic error is `identity-continuity: current production proof drift: full-16-black.png`. The verifier's other checks include manifest hashes, logo derivative rerenders, fonts, contrast, PDF messaging, tokens and contracts. This is a proof mismatch within the formal upstream distribution, not a local byte change. Full semantic kit conformance is not claimed.

The continuity record declares SHA-256 `354944c7fdf20a721b4cdce5f663ead137ccd46ec9cf924a180e8250c7506202` for the full/16/black proof. Delivered proof bytes hash to `a78393b8844196823456cc828933ec66b5632b9a9804e723ad510f6527161921` in both the original kit and verifier copy.

Resolve the proof mismatch upstream and adopt a newer formal kit through the maintainer updater. Local modification of a governed proof would break the upstream receipt and is not the integration remedy. No upstream issue, source change or publication was performed by this foundation.
