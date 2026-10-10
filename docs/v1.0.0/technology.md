# Technology decisions

## Language and desktop comparison

Go is the orchestration language because the application is dominated by storage, process supervision, streaming I/O, adapters and a scriptable CLI. The decision does not require AI libraries to be rewritten in Go. Dedicated workers use the runtime best supported by each model. The CLI is a fixed application foundation and the GUI wraps the same contracts, with declared temporary lag for advanced/experimental capabilities.

| Option | Strengths for insonic | Material costs | Decision |
| --- | --- | --- | --- |
| Go core and Wails GUI | Shared Go application logic; native webview; conventional CLI; straightforward supervision | Ladybug uses Cgo; webview/native packaging varies by OS; model workers still need packaging | Orchestration and desktop choice |
| Rust core and Tauri GUI | Native integration and strong compile-time contracts; mature desktop packaging | More complex core implementation and FFI; changing language does not eliminate model/native dependencies | Strong fallback if Go packaging proves unsuitable |
| Electron with TypeScript | Consistent bundled browser; large desktop ecosystem | Larger runtime and update footprint; native graph binding and shared CLI contracts need care | Consider if native webviews cannot meet product needs |
| Python application and Qt | Direct local AI integration; established desktop toolkit | Runtime/environment distribution, native dependencies and CLI/GUI packaging remain substantial | Use Python for suitable workers, not the default core |

[Wails v2](https://wails.io/docs/introduction/) is the stable starting point; [v3](https://v3.wails.io/) must be evaluated against its release status before adoption. React and TypeScript provide the desktop UI. Next.js is the documentation renderer and is not a desktop production server. [Tauri's architecture](https://v2.tauri.app/concept/architecture/) remains an alternative.

## Subtitle and graph dependencies

Cueson is mandatory and upstream-owned. The [v1.2.0 public architecture](https://github.com/shruggietech/cueson/blob/v1.2.0/docs/architecture.md) defines a CLI and Cue JSON Schema contract, not a stable importable Go library. Integrate a pinned official executable and schema through the subtitle adapter. Do not import upstream `internal/` packages or invent a second subtitle interchange model.

[LadybugDB](https://docs.ladybugdb.com/) is the default embedded property graph and openCypher engine. Its [Go binding](https://github.com/LadybugDB/go-ladybug) uses Cgo and native libraries. Pin a compatible binding/core pair and its artifact digests. Package metadata records both revisions; builds never resolve a mutable latest-library download.

ArcadeDB is a required supported secondary graph backend, not a fallback if Ladybug fails. Its [Apache 2.0 licensed implementation](https://github.com/ArcadeData/arcadedb/blob/main/LICENSE) provides a server [HTTP/JSON API](https://docs.arcadedb.com/arcadedb/reference/http-api/http) suitable for a Go adapter without embedding a JVM in the default desktop package. Current [native OpenCypher documentation](https://docs.arcadedb.com/arcadedb/reference/cypher/cypher-introduction) distinguishes `opencypher` from deprecated legacy `cypher`; the actual server contract/version must be pinned and tested. SQL and Cypher support are not proof of dialect equivalence with LadybugDB. Both adapters implement normalized application operations and explicit native query dialects.

Only officially supported, license-compatible dependencies are shipped. Users can supply optional unofficial graph adapters with declared licenses and capabilities. The official product does not adopt or redistribute an additional graph engine merely because an extension can connect to it.

## Catalog and artifact adapters

SQLite is the default operational catalog. PostgreSQL is a required fully supported alternative for the same process/state/settings/metadata/association/lease/outbox functions. Qualify maintained SQLite and PostgreSQL drivers against the packaging and server-version matrices. Backend-specific SQL and migrations implement one [logical schema](schema.md). No processing or UI code depends on SQLite SQL or assumes an on-disk catalog exists.

Filesystem is the default immutable artifact store. A generic S3 API adapter supports local filesystem-backed S3 services and remote S3-compliant providers. Endpoint, bucket, prefix, region, addressing style and credentials are configuration. No particular S3 server product becomes a required dependency. Provider capabilities are qualified for correctness, including completed-object visibility, conditional creation, checksums, multipart/resume and ranged reads. Use a maintained Go S3 client against this contract; an S3 compatibility claim does not justify assuming every AWS-specific feature exists.

[Storage](storage.md) defines publication, materialization, catalog transactions, leases, migration and backup for these choices. Full adapter support covers those operations and their recovery behavior.

## Local AI and speaker training

[faster-whisper](https://github.com/SYSTRAN/faster-whisper) implements the local recognition adapter for CPU int8 transcription and suitable NVIDIA systems. The current source build uses an explicitly configured hash-pinned Python executable and worker with exact dependency versions. Managed installer delivery remains a packaging contract. [CTranslate2 hardware support](https://opennmt.net/CTranslate2/hardware_support.html) and model size determine CPU/GPU profiles.

[whisper.cpp](https://github.com/ggml-org/whisper.cpp) provides a native alternative, particularly for macOS acceleration and compact CPU setups. Users can override either default. The model catalog describes language coverage, download size, approximate memory needs, acceleration, license and source before acquisition. CPU processing remains available without acceleration.

For acoustic diarization, [pyannote Community-1](https://huggingface.co/pyannote/speaker-diarization-community-1) implements the local diarization adapter through a separately configured Python/PyTorch worker. Structural acceptance cannot establish voice accuracy: models may split one voice or merge several voices, and quality diagnostics disclose that uncertainty. Upstream model acquisition conditions and any Hugging Face token are shown as distribution requirements for that model. Users can select another local or hosted adapter. Text reasoning can help name voices but cannot substitute for acoustic segmentation.

[Optional speaker-model training](voice-models.md) reuses the library-wide segment corpus through versioned preparation/training adapters. Preparation and training algorithms are adapter choices; dataset and artifact contracts are independent of them. Persist datasets, training attempts, checkpoints and speaker-model versions before choosing a specific worker, so model/provenance storage is independent of the training library.

FFmpeg/ffprobe provide media probing and derivatives. Capture broad embedded metadata before transformations using the extractor contract in [ingestion](ingestion.md). Packaging selects redistributable artifacts and preserves [FFmpeg notices](https://ffmpeg.org/legal.html). Codec licensing and native-library requirements are checked for the exact shipped bytes. Packages may contain companion executables and libraries; a one-file dependency-free binary is not promised.

## Platform and dependency qualification

Prove CLI/runtime/GUI IPC, Cueson invocation, the Ladybug binding/core pair, SQLite and PostgreSQL catalog operations, filesystem and generic S3 publication, ArcadeDB sessions/query dialects, early metadata extraction, secrets storage, hidden process creation and static offline docs. The native installation matrix covers Windows, macOS and Linux; remote-service integration uses bounded fixtures and an explicit version/capability matrix. Prove the default and supported-alternative paths without bundling or launching every remote service for ordinary users.

Target x86-64 and ARM64 where the dependency matrix supports them; record gaps as work to resolve before claiming packages. All three operating systems remain required. Measure cold-start and build time before expanding implementation. Record exact Go, Wails, driver and native pins with the platform evidence.

Qualify the exact dependency contract and artifact digests; version labels alone do not establish compatibility.

Desktop packaging pins Windows/Linux x86-64 and macOS ARM64. The macOS FFmpeg/ffprobe companions build from exact upstream source with automatic external-library discovery and network access disabled; the system VideoToolbox encoder provides non-H.264 previews. Windows/Linux companion inventories retain their exact upstream build/source information and licenses. Public redistribution of GPL-enabled builds requires their complete corresponding source, including enabled external libraries, rather than only the FFmpeg archive. Local package qualification does not establish release signing or public distribution readiness.
