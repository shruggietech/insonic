# S017 launch decisions

- Finalize the existing prepared 1.0.0; `release.py prepare` requires a newer version and must not overwrite the prepared version. Update both changelogs with the actual date and keep software/schema identities unchanged.
- Reuse the six-package guarded workflow. Qualification precedes the exact tag; publication requires that matching tag, a clean protected-main source and completed successful external checks. Avoid parallel unfinished qualification on the same launch revision and keep main stable throughout launch.
- Publication verifies ten assets in a draft before making the release public. Highlights are capped at 180 words and end with the tagged changelog link.
- Promote original published documentation to existing legacy Pages `gh-pages:/` on HTTPS insonic.io. Retry promotion-only from the original release after a deployment-only failure; never rebuild/overwrite immutable published assets.
- Retain truthful source candidate status with elected released public/offline exports. Publisher identity remains unconfigured; built-in macOS ad-hoc loader signatures remain. Acoustic accuracy testing is excluded by the owner.

Read-only Spec Kit research reviewed `scripts/release.py`, `scripts/pages.py`, `.github/workflows/release.yml`, the site release renderer and S016 launch instructions. No infrastructure blocker was found.
