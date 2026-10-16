# Research: Media lifecycle correctness

## 2026-10-15

- Decision: reuse durable report cleanup, improving error precedence before sharing post-acceptance handling. Catalog commit already queues obsolete report rows and artifact retirement enforces leases/current references. Startup and periodic sweeps already recover pending cleanup. A new queue or blind conflict suppression would add risk.
- Decision: derive canonical MIME from canonical format/current facts. Original Metadata.MIME is deliberately historical source provenance. Preview encoding already assigns its own output MIME. Legacy media keeps its existing validated fallback.
- Decision: select encoder subrip with SRT output. The exact restricted FFmpeg recipe enables subrip; the srt encoder alias is absent. No dependency upgrade is needed.
- Decision: fix parser-level required-target registration while keeping per-item manifest validation. The domain manifest handler already rejects global selector/positional conflicts.
- Decision: classify elected acquisition timeouts separately from parent cancellation and oversized input separately from child output. Existing durable per-item/work error propagation should be reused, with bounded static messages and no provider response echo.
- Alternatives rejected: source MIME on canonical bytes, blanket cleanup conflict masking, unrestricted system FFmpeg, removing all transcript target validation, generic operation_failed for every HTTP cause, and a second qualification framework.
- Research agents inspected playback/retirement and transcript/transport contracts. No unresolved technology choice requires owner input.

- Implementation refinement: catalog retirement returns an internal deferred marker only for retention/live-claim guards; public callers still receive conflict. Genuine unmarked conflicts and storage failures outrank deferred cleanup. Existing error enums remain unchanged: transcript deadlines use acquisition_timeout, byte bounds use input_limit, and fixed budget/flag corrections persist in existing item notices. Human prose notices retain literal punctuation.
