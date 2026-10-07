# S005 shared runtime contract

Version remains0.0.0. Requests use existing workspace/request/item identities and typed bounded data. Routes cover processing configuration/status, durable media processing, transcription-only and diarization-only reruns, current recording inspection/assembly/export, and external known-speaker map list/correction. Exact CLI spelling follows existing media/models/work grouping and is documented after integration.

Configured product processing can invoke actual local engines; ordinary CI injects supplied/deterministic stage results only through test adapters. Generated subtitle/turn arrays never persist in request/work receipts as duplicate assignment authority. Response pages/reference IDs remain bounded; oversized current documents/export bytes use leased materialization or caller-supplied confined output, not unbounded IPC.

Current publication checks live work authority, expected recording/source revisions, local schema/timing and publication references in one write transaction. Rerun/correction conflict is explicit; retry learns accepted identities without returning old document copies. Native strict export validates omissions before publishing output; non-strict exports return diagnostics. No silent hosted or device fallback.

Native workers use literal argv, hidden creation, supervised cancellation, offline mode and bounded separate output. Secrets cannot travel in argv or diagnostic receipts.
