# Brand, documentation and releases

Current packages include verified optional worker catalogs, model manifests, probe code and hash locks alongside the processing worker. Python, uv, recognition/diarization libraries and model weights are separately elected downloads with declared identities and licenses; package qualification verifies this metadata without acquiring weights.

## Upstream brand authority

The [insonic brand](https://brand.shruggie.tech/insonic/guidelines/brand-essentials/) belongs to upstream `shruggietech/shruggie-brand`. The integrated formal kit is insonic 1.0.0 built with BrandBuilder 3.0.1. Governed bytes live under `brand/kit/`, with the consuming repository's receipt in `brand/lock.json`.

The kit's consumer contract, enforcement guidance, tokens, named logos, licenses and manifest are authoritative. Product copy uses `insonic` and `A ShruggieTech project`. Local code binds semantic tokens and asset references rather than recreating artwork. Brand adoption preserves product requirements.

1. A maintainer manually runs the freshness check.
2. Find the latest formal upstream release containing the insonic kit.
3. Compare its brand and builder versions with the local lock.
4. When an update is requested, download the archive and checksums.
5. Verify the archive digest, paths and full manifest.
6. Stage the exact kit and integration receipt.
7. Replace the integrated kit, rolling back on failure.
8. Review the diff and consumer conformance.

`npm run brand:check` is read-only and requires network access. `npm run brand:update` adopts the latest newer formal kit. `npm run brand:verify` checks byte integrity offline. The tools reject corrupted archives, unsafe paths, immutable-version drift and downgrades. Discovery uses official release assets. A newer BrandBuilder kit can require adoption even when the brand version is unchanged.

Freshness checking is outside ordinary CI, and application startup never downloads branding. Follow upstream semantic, glyph and consumer-conformance guidance when changing a UI. Exact bytes alone do not establish visual conformance.

## Synchronized documentation and schemas

Markdown under `docs/<version>/` is the documentation authority. `docs/versions.json` defines available versions and navigation. The root `VERSION`, software release, documentation version and [master JSON Schema](contracts.md) share the same version. The schema's individual record definitions carry their declared version inside that release contract. Upstream Cueson retains its own versioned schema.

Build the website, downloadable schemas and offline documentation from the exact release tag. Retain older version routes and immutable schema identities. `/docs/` selects the latest documented version with its explicit candidate or release status. The public root route is a product landing page; `/docs/` is the versioned documentation. `site/out/` contains the full static website, including that landing page, local Mermaid diagrams and brand assets. Readers need no production Node server or remote font/diagram CDN.

The GUI includes `site/offline/`, the documentation-only export from the same tagged build. It excludes the public landing page and retains the matching documentation, changelog, schema resources and local assets. A release manifest records documentation/schema versions, source revision and hashes. Offline help uses installed bytes and stable relative links without a network or development server. Publish the matching site artifact and promote the default version only after release publication succeeds.

Each release records a CLI/GUI capability matrix, including advanced or experimental CLI features awaiting GUI controls. Core operations share the runtime and aim for parity. Adapter protocols, database migrations, model formats and upstream dependencies record their compatibility separately within the synchronized release.

Prefer backward and forward compatibility where practical. A breaking change remains possible when needed; document its affected interfaces and migration in the [master changelog](changelog.md) and brief release highlights. Do not silently reinterpret incompatible data.

For every major release, rescan the complete documentation for technical terms and refresh the [glossary](glossary.md) with precise definitions, primary learning references and links to the local pages using each term most heavily.

## Changelog and highlights

The documentation site renders root `CHANGELOG.md` as the [changelog](changelog.md), preserving one master record. Released documentation freezes the tagged file at `docs/v<version>/references/CHANGELOG.md`; the build requires and renders that snapshot. It follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Add work under Unreleased; at release move entries into a dated version section and update comparison links. [Semantic Versioning](https://semver.org/spec/v2.0.0.html) describes releases and compatibility.

Release notes contain brief highlights, major migration actions and known material limitations. Do not paste full changelog sections. End with a link to the master changelog at the release tag.

## Release sequence

1. Start from reviewed source with green required checks.
2. Align the software, documentation, JSON Schema and changelog versions.
3. Build native packages with the CLI, dependencies and documentation.
4. Verify the inventory, checksums and dependency notices.
5. Publish the immutable tag and brief release highlights.
6. Promote the matching static documentation and schemas.
7. Receive versioned user observations as issues.

Package matrices include CLI-only and GUI-with-CLI options for Windows and Linux. Sign Windows packages using configured credentials and tooling. The release manifest identifies native libraries, Cueson, media tools, optional workers and their digests/licenses. User-selected model downloads remain separate assets.

The static export is portable to ordinary static hosting; its host and domain are deployment configuration.

## Candidate integrity and publication

### Toolchain acquisition and retry limits

Automation acquires the approved Go and Node setup actions in job steps before
invoking them through local action paths. The complete upstream action trees,
setup inputs and cache main/post hooks remain unchanged. The adjacent
`scripts/bootstrap-actions.lock.json` records immutable revisions and archive
SHA256 digests. Acquisition allows three attempts within 60 seconds per action,
including bounded backoff. Only transport failures, transient server responses,
rate limiting and the known public pinned-archive 404 case are retried.
Authentication errors, changed digests and unsafe or oversized archives fail.
No unverified action is executed and existing destinations are not reused.

Jobs qualifying older source commits obtain bootstrap tooling from the exact
revision defining their workflow, separately from product source. Checkout and
other remote actions still depend on GitHub runner preparation; these retries
cannot repair a provider failure before job steps begin. Upstream setup actions
retain their own toolchain-download behavior after acquisition. A failed product
test is not retried by this mechanism.

To update a setup action, review its immutable upstream commit, download its
official codeload archive independently, verify the complete tree and main/post
metadata, then update revision and SHA256 together. Run bootstrap failure tests
and both hosted qualification platforms before merging. Dependabot's ordinary
workflow-action discovery does not update this lock. GitHub can change archive
compression while retaining immutable extracted contents; a compression-only
digest change requires the same reviewed refresh, never an unchecked fallback.

### Public release and documentation verification

`verify-public --candidate DIRECTORY --source-root SOURCE_DIRECTORY --site
https://insonic.io/ --pages-commit FULL_COMMIT [--output RECEIPT_JSON]` is a
read-only `python scripts/release.py` command. Select the original complete
candidate retrieved by `published`, the matching historical source checkout and
the expected completed Pages commit. It validates original local assets,
tag/source/version, complete published asset names, sizes and API digests, then
compares live documentation with the original publication-target archive.
Pages promotion runs this verification after its exact build completes.

The receipt distinguishes locally verified original assets, public binary API
metadata evidence and downloaded live documentation byte matches. It records
every checked path and digest for the complete original public export,
including HTML, schemas, stylesheet imports, fonts, navigation targets and
resolved local render resources. It is not a
repeat download of every public product binary or a visual/accessibility audit.
The public content phase allows 120 seconds total, at most three attempts and
five same-origin HTTPS redirects per request, and at most 16 MiB per response.
The original archive is bounded to 10,000 regular files and 64 MiB; the resolved
request inventory is also bounded. Public requests share one disposable worker
with four concurrent requests. Its parent enforces the complete network-phase
deadline, including DNS, connection and TLS setup, and kills/reaps only that
owned worker on timeout. Single action acquisitions use the same parent-owned
deadline boundary. Windows workers use hidden, redirected, noninteractive I/O.
No GitHub token, ambient cookie or authorization header accompanies site GETs.
The metadata client permits GET only; `GH_TOKEN` is optional for public release
API reads and may be needed to avoid anonymous API rate limits. Exact Pages
control-plane verification requires an authenticated token with Pages read
access, even when the deployed website is public.

The selected Pages branch and completed build must match before and after
content requests. Omitting `--pages-commit` supports an explicitly selected
generic HTTPS site and produces content evidence without a Pages identity
claim. The selected archive must elect the deployed released latest version.
Stale bytes, a mismatching alias, incomplete inventory or a concurrent
deployment cause a nonzero exit. A verifier failure after promotion means the
deployment needs investigation; it does not roll back or republish anything.
Use the exact original candidate again after repairing the actual cause.

Maintainer commands use `python scripts/release.py`: `prepare --version VERSION --date YYYY-MM-DD --highlights FILE` prepares a version locally, `validate --revision FULL_COMMIT [--tag vVERSION]` checks owned version/source identity, `docs --revision FULL_COMMIT --output DIRECTORY` archives matching built documentation, and `candidate --revision FULL_COMMIT --assets DIRECTORY --output NEW_DIRECTORY RECEIPTS...` assembles the verified complete asset set. `publish --candidate DIRECTORY --highlights FILE` publishes only that candidate. `published --revision FULL_COMMIT --output NEW_DIRECTORY` retrieves and validates the original complete asset set from an existing exact-version release; `promote --candidate DIRECTORY --configuration FILE` uses the configured static-host command after published-release verification. Commands do not infer a release version or deployment host.

Release preparation preserves older versioned documentation and schemas and freezes the matching changelog and referenced public files. It updates owned software and runtime bindings, package manifests and lockfiles, documentation navigation and schema identities together. Upstream tool, model and subtitle versions keep their independent identities. Preparation records an unpublished candidate in the navigation manifest; it does not create a tag or publish product downloads.

Pull-request CI retains successful CLI and desktop archives for every native platform and matching documentation from the same source revision and workflow attempt. A read-only collection job verifies the complete candidate. The selected revision is the actual PR merge revision; after squash merge, a product release requires a new candidate built from its exact main revision. Candidate archives are temporary workflow artifacts, not official release downloads. Corresponding sources must match the selected checkout's recipe, pins and build helpers and contain every dependency group's verified receipt.

Go dependency notices derive from the module metadata embedded in the executables shipped by each variant: the CLI alone for CLI packages, and the CLI/desktop union for desktop packages. Assembly requires matching module versions, checksums and upstream source directories, including versioned replacements, with cached source contents hashed against the executable's Go module sum, nonempty upstream licenses and the compiler license. Copied notices use the bytes captured during source verification. Missing or inconsistent evidence aborts packaging. Unused tooling modules are excluded. Candidate collection independently compares the notice inventory with those binaries and verifies every referenced file; historical published inventories retain their original format.

Windows cold media builds compile the complete codec archive before final FFmpeg linking. Their verified continuation preserves configured sources, dependency headers and exact file timestamps; final Make must reuse every staged object. Decoder configuration and compile deadlines stay unchanged. Included rebuild instructions use `--stage dependencies`, `--stage codec` and `--stage complete` in order from the same root. Other native platforms retain their complete-build path.

Pages preparation preserves historical page and schema paths. When an older Next.js segment URL used a nested filename and the verified export supplies its flat equivalent beside the same page, preparation retains the old URL using those verified replacement bytes. Missing historical pages, schemas or unmatched segments still abort before destination writes.

Ordinary documentation builds render the source's candidate status. The elected release workflow sets `INSONIC_DOCUMENTATION_TARGET=release` to build a publication-specific public/offline snapshot, with released navigation, a release index and all four exact-tag download links. Its build marker retains that target alongside clean source/version identity; this rendering does not publish or deploy anything. Publication, published-asset retrieval and promotion require release-target documentation and reject a candidate snapshot. Promotion still waits for verified release publication and reuses that release's original archive. Source manifest and historical snapshots are not rewritten by rendering.

Candidate validation requires both package variants on every supported native platform, one clean exact source revision, matching software/documentation/schema versions, complete corresponding sources and build records, final inventory hashes and explicit signing disposition. Configured signing happens before final package hashes; a signing failure aborts assembly. An absent publisher identity does not become a signed release claim.

Publication first uploads the complete verified asset set to a draft release and checks the remote assets against the candidate manifest. Only a verified draft can become published. Interrupted upload remains a draft and cannot promote documentation. Documentation promotion verifies the published release's tag, source revision and asset set before sending the matching immutable site to the configured static host. A successful build alone never establishes release publication or documentation promotion.

Dispatch publication and promotion from the protected default `main` branch. The selected full source commit must belong to that branch's history; a historical reviewed commit remains eligible. Publication and promotion run verifier code and helpers from trusted `main`, while a separate checkout supplies the exact candidate's version and source data. Candidate code never supplies the privileged publisher's verifier.

For generic workflow documentation promotion, configure the repository variable `INSONIC_DOCUMENTATION_DEPLOYMENT_JSON` with an object containing `argv` and optional `environment_names`. `argv` selects an absolute executable available on the promotion runner and includes a `{archive}` argument for the verified documentation archive; `{version}` and `{revision}` can select its immutable destination. Configure the `documentation-deployment` environment with a custom deployment branch policy allowing exactly `main` with type `branch`. Mandatory reviewers are not required. Store the dedicated host credential in that environment's `INSONIC_DOCUMENTATION_DEPLOYMENT_CREDENTIAL` secret and select that name explicitly through `environment_names` when the host needs authentication. The configuration contains public command and destination settings and credential names, never secret values.

When `promote_docs` is selected, a read-only preflight job validates the environment branch policy, executable, configuration and selected credential availability before publication. The publisher holds repository write permission without deployment secrets. A separate promotion job holds read-only GitHub permission and receives the dedicated deployment secret. Its host command receives only explicitly named values and standard operating-system execution variables; automation credential names and the `GH_`, `GITHUB_`, `ACTIONS_` and `RUNNER_` namespaces cannot be forwarded. Missing or invalid configuration fails before a release becomes public. Selecting promotion with `publish=false` retrieves and verifies the already published release's original assets without rebuilding packages or creating another release. Rebuilding the same revision does not guarantee identical archive bytes; promotion reuses the published documentation. The generic route does not provision a host, environment, secret or publisher identity.

The source-buildable package and release tools provide these contracts; they do not make the development baseline an official installable release. Creating the product tag, publishing the release and selecting a documentation destination remain explicit maintainer actions.


## Existing GitHub Pages transport

The release workflow's `documentation-transport` input elects `pages` or `generic`; the default selects the existing Pages destination. Pages preflight requires this repository's legacy source `gh-pages:/`, custom domain `insonic.io`, HTTPS enforcement, `CNAME`, `.nojekyll` and a `github-pages` environment policy permitting exactly the source branch. It does not change the domain, source, environment policy or signing configuration.

After exact published-release verification, Pages promotion uses the original release documentation archive. It validates complete release-target identity, confined regular files and retained historical documentation/schema paths before creating Git objects. A parented commit advances `gh-pages` without force; concurrent ref changes fail. The transport preserves `CNAME` and `.nojekyll`, explicitly requests a Pages build and waits up to five minutes for that exact commit to finish. A queued build or a previous successful commit cannot establish deployment success. A lost ref-update response can reconcile only against the exact staged remote commit.

The Pages preflight job has read-only repository/Pages permissions. Its promotion job alone receives repository/Pages write permission, runs trusted-main verifier code and holds no generic-host secret. The existing `github-pages` environment remains scoped to `gh-pages`; the main-branch preparation job does not claim that environment. The generic host continues to use its separate main-only environment and dedicated credential. Selecting either transport with `publish=false` promotes only a previously published exact release.

The release workflow reuses the same independent engine, desktop/credential and CLI/desktop package lanes as pull-request qualification. Engine and desktop runtime caches have separate identities. Every required acceptance scenario, complete corresponding-source verification and package inventory remains mandatory; the collector requires all four package variants and same-revision, same-attempt documentation. Configured publisher-signing inputs reach package lanes only. Unsigned packages retain explicit dispositions.
