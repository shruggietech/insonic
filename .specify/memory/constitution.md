# insonic Constitution

## Core Principles

### I. Owner requirements define the product

Agents MUST preserve approved scope and explain deviations. Potential misuse alone MUST NOT create new eligibility rules, approval queues or content restrictions. Technical integrity and access controls MUST remain distinct from product-use policy. Unapproved suggestions MUST NOT become binding requirements. Material scope reductions require owner agreement.

### II. CLI-first foundation and shared application behavior

The application MUST be CLI-first. Core operations MUST have versioned machine-readable and human CLI interfaces. The GUI MUST install the CLI and wrap the shared application/runtime contracts, maintaining core feature parity as the goal without GUI-only core logic. Advanced or experimental CLI operations MAY precede GUI controls when the matching release documents that lag and follow-up. Windows, macOS and Linux installable packages remain required. Platform implementation choices MUST be validated in bounded slices before shipping claims.

### III. Preserve source identity and uncertainty

Owner-owned source inputs MUST remain immutable. New admission MUST retain canonical audio and, when a transcript is selected, one current validated Cue JSON document, capture source metadata before conversion, and retire application-owned original video and redundant native subtitle artifacts after verified acceptance. Canonical audio MUST preserve required tracks, channels, detail and timing independently of engine scratch formats. Original local file locations MUST NOT persist as new accepted recording authority. Pre-release format changes MUST NOT acquire a bulk media-migration requirement without deployed data requiring it. Explicit audio replacement MUST preserve accepted data until the new current authority commits; normal catalog/schema evolution and transcript-version interoperability remain separate requirements. Available raw EXIF, container, stream and other source metadata MUST be captured before transforms and retained with extraction provenance. Derivatives, cues, voices, speaker audio segments, training datasets/models and assertions MUST retain source identity, time mapping and processing provenance. Unknown dates and speakers MUST remain explicit; date/time input MUST preserve precision, time zone, assumptions and conflicting observations. Cueson MUST govern subtitle interchange. Catalog-to-graph publication MUST be recoverable and revision-aware. Optional user-elected training MUST associate versioned model artifacts with the originating speaker and an immutable corpus snapshot.

### IV. Configurable adapters and protected credentials

Pipelines MUST support replaceable local and hosted adapters with readable configuration. Storage MUST default to filesystem and fully support generic S3-compatible providers. Operational state, settings, metadata and associations MUST default to SQLite and fully support PostgreSQL. Graph querying MUST default to LadybugDB and fully support ArcadeDB. Portable contracts and schema migrations MUST avoid silently reducing either supported backend to a lesser feature set. Optional end-user extensions MAY support additional backends without making them bundled project dependencies.

Defaults MUST work without requiring users to understand model servers. Credential values MUST remain encrypted, absent from persisted job configuration and logs, and hidden from status responses. Local failures MUST NOT silently route data to an unselected provider. Configured imports, transforms, attribution, jobs and elected training MUST NOT acquire mandatory per-item human review gates. Ask only for a concrete unresolved decision or action outside existing authorization; retain uncertainty and correctability through data contracts.

### V. Efficient, truthful development

Implementation MUST follow coherent Spec Kit slices with linked Issues, milestones and a GitHub Project. Controlled automated checks MUST target meaningful change risks. User observations ordinarily arrive after release as versioned issues; agents MUST NOT invent a pre-release attended verification ritual. CI turnaround MUST stay below ten minutes through measured scope, caching and parallel jobs. Public status MUST distinguish specified, implemented, tested and shipped behavior.

## Repository and publication constraints

Markdown documentation lives in `docs/`. The software, documentation and master JSON Schema MUST share the release version. Static Next.js output MUST render the master changelog and JSON contracts within the documentation site. Prefer backward and forward compatibility where practical; neither is an absolute requirement. Breaking changes MUST be described in the changelog and concise release notes, with their affected interface and migration implications.

Public documentation MUST state authoritative system contracts. Slice identifiers, task records and verification evidence belong in internal planning and MUST NOT be linked or rendered as official documentation. Historical and version-change descriptions belong in `CHANGELOG.md`. The documentation index carries one concise specification-status statement. For every major release, maintainers MUST rescan all documentation for technical terms and refresh the glossary with precise definitions, external primary learning references and links to the local pages that use each term most heavily.

Purely linear procedures MUST use numbered steps. Diagrams describe meaningful branches, relationships, dependencies, feedback or retry paths; Mermaid flowcharts MUST use top-down orientation. Upstream owns the brand kit; maintainers MUST adopt checksum-verified formal updates without editing governed kit bytes. Files MUST use UTF-8 without BOM. Public documentation and planning MUST be self-contained and contain no private external design-material references or case-specific corpus information.

Windows console children MUST use hidden creation guarantees and noninteractive I/O. Direct Git/GitHub command-runner operations are permitted. Fix poor logic proportionally to the slice rather than copying it during a port or refactor.

## Development workflow

Specify, clarify where necessary, plan, tasks, analyze, implement and converge a slice before proposing publication. Resolve routine choices autonomously under approved requirements; do not add owner review gates between routine stages. Record unresolved decisions and verification evidence. Squash merges follow green automated checks and resolved review findings, with automatic merged-branch deletion. Human PR approval MUST NOT be mandatory. Agents follow the owner's authorization for the merge itself. Existing publication authorization MUST be honored without repeatedly requesting it. Releases MUST have concise highlights ending in a link to the master changelog; full detail remains in Keep a Changelog. No push, repository publication or release is authorized by this local foundation alone.

## Governance

Owner instructions take precedence. Amendments MUST record their changed product or process effect in the changelog and update the constitution version using semantic versioning. Reviewers compare changes against this constitution, the current system specification and the slice acceptance criteria. Exceptions MUST be concrete, justified and reported; passing unrelated checks does not resolve unmet requirements.

**Version**: 2.2.1 | **Ratified**: 2026-10-04 | **Last Amended**: 2026-10-09
