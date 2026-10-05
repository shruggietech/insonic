# Product and requirements

## Product boundary

insonic owns orchestration, user configuration, the library catalog, processing history and the presentation of source-linked results. It integrates media tools, Cueson, model runtimes, storage and databases through explicit adapters. Local processing and hosted providers are both supported routes. Speaker-tagged audio segments across the library can form a user-elected training corpus; trained artifacts belong to their originating speaker records. Training is optional and never a prerequisite to using the library.

Users can import local files or acquire user-targeted remote sources through acquisition adapters. Acquisition adapters declare supported protocols and services; failures explain unsupported formats or authentication requirements.

The primary journey is Add media, Choose processing, Process, Explore. A user can begin without learning model servers, database syntax or pipeline internals. Advanced users can select models, endpoints, channels, storage/database backends, adapters and query parameters. The CLI defines core application operations, and the GUI wraps those operations with a parity goal; explicitly documented advanced or experimental capabilities may reach the GUI later.

## Requirement map

Stable requirement IDs identify observable product behavior. The named contracts define acceptance.

| ID | Required behavior | Contract |
| --- | --- | --- |
| R01 | Installable CLI on Windows, macOS and Linux | [Technology](technology.md), [storage](storage.md) |
| R02 | CLI-first shared core; GUI installs CLI and maintains core parity, with documented advanced/experimental lag permitted | [Desktop](desktop.md) |
| R03 | Track audio and video originals, provenance and duplicates | [Media](media.md) |
| R04 | Acquire user-targeted sources through adapters | [Pipelines](pipelines.md) |
| R05 | Correlate optimized audio derivatives with original time and channels | [Media](media.md) |
| R06 | Accept optional subtitles for every supported media type | [Subtitles](subtitles.md) |
| R07 | Produce a correlated subtitle artifact for completed speech processing | [Subtitles](subtitles.md) |
| R08 | Require Cueson as the unified subtitle layer | [Subtitles](subtitles.md) |
| R09 | Configure user-defined pipelines with model and hosting overrides | [Pipelines](pipelines.md) |
| R10 | Provide practical local AI defaults including faster-whisper where suitable | [Technology](technology.md) |
| R11 | Store provider keys encrypted with usable setup and replacement | [Credentials](security.md) |
| R12 | Track voices, speakers, aliases and corrections | [Speakers](speakers.md) |
| R13 | Supply specialized terms and speaker names as transcription context | [Speakers](speakers.md) |
| R14 | Chunk source-backed speaker assertions through adapters into the configured graph database | [Graph](graph.md) |
| R15 | Organize application, workspaces, media and downloaded models | [Storage](storage.md) |
| R16 | Query assets, speakers, words and times through CLI and GUI | [Graph](graph.md), [desktop](desktop.md) |
| R17 | Render a graphical timeline of all library media | [Graph](graph.md) |
| R18 | Save graph queries and render force-directed selected results | [Graph](graph.md) |
| R19 | Offer optional AI-assisted query suggestions through chosen adapters | [Graph](graph.md) |
| R20 | Use upstream-owned branded assets and manual scriptable updates | [Releases](releases.md) |
| R21 | Author versioned Markdown and compile static Next.js docs | [Releases](releases.md) |
| R22 | Ship release-matched offline docs in GUI packages | [Releases](releases.md) |
| R23 | Use Spec Kit slices and GitHub Issues, milestones and Projects | [Development](development.md) |
| R24 | Keep CI turnaround below ten minutes and use squash merges with branch deletion, without mandatory human PR approval | [Development](development.md) |
| R25 | Maintain Keep a Changelog and brief release highlights ending with its link | [Releases](releases.md) |
| R26 | Retain source-timed speaker audio segments across the library and revisioned speaker corpus selections | [Speakers](speakers.md), [speaker audio](voice-models.md) |
| R27 | Optionally train speaker-associated voice models and list/fetch their artifacts through CLI contracts | [Voice models](voice-models.md), [schema](schema.md) |
| R28 | Capture immutable raw EXIF, container, stream and other available metadata before transforms, with field-level provenance | [Media](media.md), [schema](schema.md) |
| R29 | Accept per-item origination dates, date-only values and explicit time zones through import flags and batch manifests | [Media](media.md), [desktop](desktop.md) |
| R30 | Default to local filesystem storage and fully support generic S3-compatible object storage | [Storage](storage.md), [schema](schema.md) |
| R31 | Default to SQLite and fully support PostgreSQL for operational state, metadata, settings and associations | [Schema](schema.md), [storage](storage.md) |
| R32 | Default to LadybugDB and fully support ArcadeDB graph projections and querying | [Graph](graph.md), [schema](schema.md) |
| R33 | Run configured import, transforms, attribution, jobs and elected training without mandatory per-item review | [Pipelines](pipelines.md), [desktop](desktop.md), [development](development.md) |

## Completion and exceptions

Speech-capable input does not require supplied subtitles. Successful processing ends with a valid correlated transcript and an exportable subtitle file. Pending, cancelled and failed processing remains visible and resumable; it is never labelled complete. Music and ambient sound classifications may skip automatic speech work. Mixed media can still contain speech, and a user can force transcription or override a mistaken classification. Neither a model's classification nor the absence of speech confidence makes a file ineligible for the library.

Keep uncertain speaker identities and uncertain calendar dates usable. The application records uncertainty explicitly instead of guessing. Direct querying and exploration work without AI query assistance enabled.

Configured routines run without repeated approvals. Only the affected operation requires additional input when a date cannot be interpreted, a destructive effect is not already authorized, or a new external processing route has not been selected. Confidence filters and attribution policies are user-configurable automation inputs; uncertain results remain labelled and correctable without mandatory review queues.

## Delivery dependencies

Runtime, storage and media contracts establish the import-to-subtitle workflow on all three operating systems, with full S3-compatible storage and PostgreSQL support. Provider flexibility and speaker continuity support graph extraction, desktop exploration and optional speaker training. Model discovery and retrieval begin with CLI contracts; desktop presentation follows the shared operation model. See [delivery outcomes](roadmap.md) for dependencies. Multi-user collaboration and an insonic-hosted library service are outside these application contracts.
